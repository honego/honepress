package httpserver

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"path"
	"strings"

	"github.com/honeok/blog/model"
	"github.com/honeok/blog/option"
	"github.com/honeok/blog/service"
	"github.com/honeok/blog/web"
)

// Server 封装 net/http 路由，公开页面、后台和 API 都从这里进入。
type Server struct {
	options     option.Options
	blogService *service.BlogService
}

// New 创建 HTTP 服务实例。
func New(options option.Options, blogService *service.BlogService) *Server {
	return &Server{
		options:     options,
		blogService: blogService,
	}
}

// ListenAndServe 启动 HTTP 服务。
func (server *Server) ListenAndServe() error {
	log.Printf("博客服务正在监听：%s", server.options.Address)
	return http.ListenAndServe(server.options.Address, server.routes())
}

func (server *Server) routes() http.Handler {
	httpServeMux := http.NewServeMux()
	httpServeMux.Handle("/api/", server.basicAuth(http.HandlerFunc(server.handleAPI)))
	httpServeMux.Handle("/api", server.basicAuth(http.HandlerFunc(server.handleAPI)))
	httpServeMux.Handle("/admin/", server.basicAuth(http.HandlerFunc(server.serveAdmin)))
	httpServeMux.Handle("/admin", server.basicAuth(http.HandlerFunc(server.redirectAdmin)))
	httpServeMux.Handle("/", http.HandlerFunc(server.servePublic))
	return securityHeaders(httpServeMux)
}

func (server *Server) basicAuth(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if server.options.AdminPassword == "" {
			nextHandler.ServeHTTP(responseWriter, request)
			return
		}

		username, password, hasCredentials := request.BasicAuth()
		usernameMatches := subtle.ConstantTimeCompare([]byte(username), []byte(server.options.AdminUsername)) == 1
		passwordMatches := subtle.ConstantTimeCompare([]byte(password), []byte(server.options.AdminPassword)) == 1
		if !hasCredentials || !usernameMatches || !passwordMatches {
			responseWriter.Header().Set("WWW-Authenticate", `Basic realm="blog admin"`)
			http.Error(responseWriter, "需要后台认证", http.StatusUnauthorized)
			return
		}

		nextHandler.ServeHTTP(responseWriter, request)
	})
}

func securityHeaders(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("X-Content-Type-Options", "nosniff")
		responseWriter.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		nextHandler.ServeHTTP(responseWriter, request)
	})
}

func (server *Server) redirectAdmin(responseWriter http.ResponseWriter, request *http.Request) {
	http.Redirect(responseWriter, request, "/admin/", http.StatusMovedPermanently)
}

func (server *Server) serveAdmin(responseWriter http.ResponseWriter, request *http.Request) {
	adminDistFS, err := web.AdminDistFS()
	if err != nil {
		server.serveAdminFallback(responseWriter)
		return
	}

	cleanRequestPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/admin/"))
	if cleanRequestPath == "." || cleanRequestPath == "/" {
		server.serveAdminIndex(responseWriter, adminDistFS)
		return
	}

	fileInfo, err := fs.Stat(adminDistFS, cleanRequestPath)
	if err == nil && !fileInfo.IsDir() {
		fileServerRequest := request.Clone(request.Context())
		fileServerRequest.URL.Path = "/" + cleanRequestPath
		http.FileServer(http.FS(adminDistFS)).ServeHTTP(responseWriter, fileServerRequest)
		return
	}

	server.serveAdminIndex(responseWriter, adminDistFS)
}

func (server *Server) serveAdminIndex(responseWriter http.ResponseWriter, adminDistFS fs.FS) {
	indexFileContent, err := fs.ReadFile(adminDistFS, "index.html")
	if err != nil {
		server.serveAdminFallback(responseWriter)
		return
	}
	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write(indexFileContent)
}

func (server *Server) serveAdminFallback(responseWriter http.ResponseWriter) {
	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write([]byte("<!doctype html><html lang=\"zh-CN\"><head><meta charset=\"utf-8\"><title>blog 后台</title></head><body><p>后台前端尚未构建，请先构建后台前端。</p></body></html>"))
}

func (server *Server) servePublic(responseWriter http.ResponseWriter, request *http.Request) {
	publicFileServer := http.FileServer(http.Dir(server.options.PublicDir))
	publicFileServer.ServeHTTP(responseWriter, request)
}

func (server *Server) handleAPI(responseWriter http.ResponseWriter, request *http.Request) {
	cleanAPIPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/api"))
	if cleanAPIPath == "." {
		cleanAPIPath = "/"
	}

	switch {
	case cleanAPIPath == "/health" && request.Method == http.MethodGet:
		server.writeJSON(responseWriter, http.StatusOK, healthResponse{Status: "ok"})
	case cleanAPIPath == "/posts" && request.Method == http.MethodGet:
		server.handleListPosts(responseWriter)
	case cleanAPIPath == "/posts" && request.Method == http.MethodPost:
		server.handleCreatePost(responseWriter, request)
	case cleanAPIPath == "/preview" && request.Method == http.MethodPost:
		server.handlePreview(responseWriter, request)
	case cleanAPIPath == "/render" && request.Method == http.MethodPost:
		server.handleRender(responseWriter)
	case cleanAPIPath == "/settings" && request.Method == http.MethodGet:
		server.handleGetSettings(responseWriter)
	case cleanAPIPath == "/settings" && request.Method == http.MethodPut:
		server.handleUpdateSettings(responseWriter, request)
	case strings.HasPrefix(cleanAPIPath, "/posts/"):
		server.handlePostByID(responseWriter, request, strings.TrimPrefix(cleanAPIPath, "/posts/"))
	default:
		server.writeError(responseWriter, http.StatusNotFound, "接口不存在")
	}
}

func (server *Server) handleListPosts(responseWriter http.ResponseWriter) {
	postSummaries, err := server.blogService.ListPosts()
	if err != nil {
		server.writeError(responseWriter, http.StatusInternalServerError, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusOK, postsResponse{Posts: postSummaries})
}

func (server *Server) handleCreatePost(responseWriter http.ResponseWriter, request *http.Request) {
	var savePostRequest model.SavePostRequest
	if err := server.decodeJSON(request, &savePostRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	createdPost, err := server.blogService.CreatePost(savePostRequest)
	if err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusCreated, postDetailResponse{Post: createdPost, Message: "文章已创建并重新生成。"})
}

func (server *Server) handlePreview(responseWriter http.ResponseWriter, request *http.Request) {
	var previewRequest model.PreviewRequest
	if err := server.decodeJSON(request, &previewRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}

	renderedHTML, err := server.blogService.PreviewMarkdown(previewRequest.Markdown)
	if err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write([]byte(renderedHTML))
}

func (server *Server) handleRender(responseWriter http.ResponseWriter) {
	if err := server.blogService.RenderAll(); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "静态文件已重新生成。"})
}

func (server *Server) handleGetSettings(responseWriter http.ResponseWriter) {
	server.writeJSON(responseWriter, http.StatusOK, settingsResponse{Settings: server.blogService.GetSiteSettings()})
}

func (server *Server) handleUpdateSettings(responseWriter http.ResponseWriter, request *http.Request) {
	var settings model.SiteSettings
	if err := server.decodeJSON(request, &settings); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	if err := server.blogService.UpdateSiteSettings(settings); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusOK, settingsResponse{
		Settings: server.blogService.GetSiteSettings(),
		Message:  "站点设置已保存并重新生成。",
	})
}

func (server *Server) handlePostByID(responseWriter http.ResponseWriter, request *http.Request, rawPostPath string) {
	if strings.HasSuffix(rawPostPath, "/translate") {
		sourceFileName := strings.TrimSuffix(rawPostPath, "/translate")
		if request.Method != http.MethodPost {
			server.writeError(responseWriter, http.StatusMethodNotAllowed, "请求方法不允许")
			return
		}
		if err := server.blogService.RegenerateEnglishPost(sourceFileName); err != nil {
			server.writeError(responseWriter, http.StatusBadRequest, err.Error())
			return
		}
		server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "英文页面已重新生成。"})
		return
	}

	sourceFileName := rawPostPath
	switch request.Method {
	case http.MethodGet:
		postDetail, err := server.blogService.GetPost(sourceFileName)
		if err != nil {
			server.writeError(responseWriter, http.StatusNotFound, err.Error())
			return
		}
		server.writeJSON(responseWriter, http.StatusOK, postDetailResponse{Post: postDetail})
	case http.MethodPut:
		var savePostRequest model.SavePostRequest
		if err := server.decodeJSON(request, &savePostRequest); err != nil {
			server.writeError(responseWriter, http.StatusBadRequest, err.Error())
			return
		}
		updatedPost, err := server.blogService.UpdatePost(sourceFileName, savePostRequest)
		if err != nil {
			server.writeError(responseWriter, http.StatusBadRequest, err.Error())
			return
		}
		server.writeJSON(responseWriter, http.StatusOK, postDetailResponse{Post: updatedPost, Message: "文章已保存并重新生成。"})
	case http.MethodDelete:
		if err := server.blogService.DeletePost(sourceFileName); err != nil {
			server.writeError(responseWriter, http.StatusBadRequest, err.Error())
			return
		}
		server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "文章已删除并重新生成。"})
	default:
		server.writeError(responseWriter, http.StatusMethodNotAllowed, "请求方法不允许")
	}
}

func (server *Server) decodeJSON(request *http.Request, targetValue interface{}) error {
	limitedRequestBody := io.LimitReader(request.Body, 8*1024*1024)
	jsonDecoder := json.NewDecoder(limitedRequestBody)
	jsonDecoder.DisallowUnknownFields()
	if err := jsonDecoder.Decode(targetValue); err != nil {
		return fmt.Errorf("解析请求 JSON 失败：%w", err)
	}
	return nil
}

func (server *Server) writeJSON(responseWriter http.ResponseWriter, statusCode int, responseValue interface{}) {
	responseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	responseWriter.WriteHeader(statusCode)
	if err := json.NewEncoder(responseWriter).Encode(responseValue); err != nil {
		log.Printf("写入 JSON 响应失败：%v", err)
	}
}

func (server *Server) writeError(responseWriter http.ResponseWriter, statusCode int, errorMessage string) {
	server.writeJSON(responseWriter, statusCode, model.APIErrorResponse{Error: errorMessage})
}

type healthResponse struct {
	Status string `json:"status"`
}

type postsResponse struct {
	Posts []model.PostSummary `json:"posts"`
}

type postDetailResponse struct {
	Post    model.PostDetail `json:"post"`
	Message string           `json:"message,omitempty"`
}

type settingsResponse struct {
	Settings model.SiteSettings `json:"settings"`
	Message  string             `json:"message,omitempty"`
}
