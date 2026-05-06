package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/core"
	"github.com/honeok/honepress/internal/model"
	"github.com/honeok/honepress/internal/service"
)

// HTTP 服务
type Server struct {
	options           config.Options
	blogService       *service.BlogService
	adminSessionToken string
}

// 创建 HTTP 服务实例
func New(options config.Options, blogService *service.BlogService) *Server {
	return &Server{
		options:           options,
		blogService:       blogService,
		adminSessionToken: newAdminSessionToken(),
	}
}

// 启动 HTTP 服务
func (server *Server) ListenAndServe() error {
	return http.ListenAndServe(core.DefaultAddress, server.routes())
}

func (server *Server) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(securityHeaders)

	router.Post("/api/login", server.handleLogin)
	router.Post("/api/logout", server.handleLogout)

	router.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(server.adminAuth)
		apiRouter.NotFound(func(responseWriter http.ResponseWriter, _ *http.Request) {
			server.writeError(responseWriter, http.StatusNotFound, "接口不存在")
		})
		apiRouter.Get("/health", server.handleHealth)
		apiRouter.Get("/posts", server.handleListPosts)
		apiRouter.Post("/posts", server.handleCreatePost)
		apiRouter.Post("/preview", server.handlePreview)
		apiRouter.Get("/settings", server.handleGetSettings)
		apiRouter.Put("/settings", server.handleUpdateSettings)
		apiRouter.Get("/posts/{postID}", server.handleGetPost)
		apiRouter.Put("/posts/{postID}", server.handleUpdatePost)
		apiRouter.Delete("/posts/{postID}", server.handleDeletePost)
	})
	router.Group(func(adminRouter chi.Router) {
		adminRouter.Get("/admin", server.redirectAdmin)
		adminRouter.Get("/admin/*", server.serveAdmin)
	})
	router.HandleFunc("/*", server.servePublic)

	return router
}

const adminSessionCookieName = "honepress_admin_session"

func (server *Server) adminAuth(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		if server.options.AdminPassword == "" || server.hasValidAdminSession(request) || server.hasValidBasicAuth(request) {
			nextHandler.ServeHTTP(responseWriter, request)
			return
		}

		server.writeError(responseWriter, http.StatusUnauthorized, "请先登录后台。")
	})
}

func (server *Server) hasValidBasicAuth(request *http.Request) bool {
	username, password, hasCredentials := request.BasicAuth()
	if !hasCredentials {
		return false
	}
	usernameMatches := subtle.ConstantTimeCompare([]byte(username), []byte(server.options.AdminUsername)) == 1
	passwordMatches := subtle.ConstantTimeCompare([]byte(password), []byte(server.options.AdminPassword)) == 1
	return usernameMatches && passwordMatches
}

func (server *Server) hasValidAdminSession(request *http.Request) bool {
	adminSessionCookie, err := request.Cookie(adminSessionCookieName)
	if err != nil || server.adminSessionToken == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(adminSessionCookie.Value), []byte(server.adminSessionToken)) == 1
}

func (server *Server) setAdminSessionCookie(responseWriter http.ResponseWriter, request *http.Request) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    server.adminSessionToken,
		Path:     "/",
		MaxAge:   7 * 24 * 60 * 60,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   request.TLS != nil,
	})
}

func clearAdminSessionCookie(responseWriter http.ResponseWriter, request *http.Request) {
	http.SetCookie(responseWriter, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   request.TLS != nil,
	})
}

func newAdminSessionToken() string {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		panic(fmt.Sprintf("生成后台会话令牌失败：%v", err))
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes)
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
	cleanRequestPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/admin/"))
	if cleanRequestPath == "." || cleanRequestPath == "/" || strings.HasPrefix(cleanRequestPath, "..") {
		server.serveAdminIndex(responseWriter)
		return
	}

	targetFilePath := filepath.Join(server.options.AdminDistDir, filepath.FromSlash(cleanRequestPath))
	fileInfo, err := os.Stat(targetFilePath)
	if err == nil && !fileInfo.IsDir() {
		fileServerRequest := request.Clone(request.Context())
		fileServerRequest.URL.Path = "/" + cleanRequestPath
		http.FileServer(http.Dir(server.options.AdminDistDir)).ServeHTTP(responseWriter, fileServerRequest)
		return
	}

	server.serveAdminIndex(responseWriter)
}

func (server *Server) serveAdminIndex(responseWriter http.ResponseWriter) {
	indexFileContent, err := os.ReadFile(filepath.Join(server.options.AdminDistDir, "index.html"))
	if err != nil {
		server.serveAdminFallback(responseWriter, err)
		return
	}
	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	responseWriter.WriteHeader(http.StatusOK)
	_, _ = responseWriter.Write(indexFileContent)
}

func (server *Server) serveAdminFallback(responseWriter http.ResponseWriter, cause error) {
	server.writeError(responseWriter, http.StatusInternalServerError, "后台前端构建产物不存在，请先构建 frontend/admin："+cause.Error())
}

func (server *Server) servePublic(responseWriter http.ResponseWriter, request *http.Request) {
	publicFileServer := http.FileServer(http.Dir(server.options.PublicDir))
	publicFileServer.ServeHTTP(responseWriter, request)
}

func (server *Server) handleHealth(responseWriter http.ResponseWriter, _ *http.Request) {
	server.writeJSON(responseWriter, http.StatusOK, healthResponse{Status: "ok"})
}

func (server *Server) handleLogin(responseWriter http.ResponseWriter, request *http.Request) {
	if server.options.AdminPassword == "" {
		server.setAdminSessionCookie(responseWriter, request)
		server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "后台已进入免密码模式。"})
		return
	}

	var loginRequest adminLoginRequest
	if err := server.decodeJSON(request, &loginRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	usernameMatches := subtle.ConstantTimeCompare([]byte(loginRequest.Username), []byte(server.options.AdminUsername)) == 1
	passwordMatches := subtle.ConstantTimeCompare([]byte(loginRequest.Password), []byte(server.options.AdminPassword)) == 1
	if !usernameMatches || !passwordMatches {
		server.writeError(responseWriter, http.StatusUnauthorized, "账号或密码不正确。")
		return
	}

	server.setAdminSessionCookie(responseWriter, request)
	server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "登录成功。"})
}

func (server *Server) handleLogout(responseWriter http.ResponseWriter, request *http.Request) {
	clearAdminSessionCookie(responseWriter, request)
	server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "已退出后台。"})
}

func (server *Server) handleListPosts(responseWriter http.ResponseWriter, _ *http.Request) {
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
	server.writeJSON(responseWriter, http.StatusCreated, postDetailResponse{Post: createdPost, Message: postSaveMessage(createdPost, true)})
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

func (server *Server) handleGetSettings(responseWriter http.ResponseWriter, _ *http.Request) {
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
		Message:  "站点设置已保存，站点已自动更新。",
	})
}

func (server *Server) handleGetPost(responseWriter http.ResponseWriter, request *http.Request) {
	postDetail, err := server.blogService.GetPost(chi.URLParam(request, "postID"))
	if err != nil {
		server.writeError(responseWriter, http.StatusNotFound, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusOK, postDetailResponse{Post: postDetail})
}

func (server *Server) handleUpdatePost(responseWriter http.ResponseWriter, request *http.Request) {
	var savePostRequest model.SavePostRequest
	if err := server.decodeJSON(request, &savePostRequest); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	updatedPost, err := server.blogService.UpdatePost(chi.URLParam(request, "postID"), savePostRequest)
	if err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusOK, postDetailResponse{Post: updatedPost, Message: postSaveMessage(updatedPost, false)})
}

func (server *Server) handleDeletePost(responseWriter http.ResponseWriter, request *http.Request) {
	if err := server.blogService.DeletePost(chi.URLParam(request, "postID")); err != nil {
		server.writeError(responseWriter, http.StatusBadRequest, err.Error())
		return
	}
	server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "文章已删除，站点已自动更新。"})
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

type adminLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
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

func postSaveMessage(postDetail model.PostDetail, created bool) string {
	if postDetail.Draft {
		if created {
			return "草稿已创建，未生成公开页面。"
		}
		return "草稿已保存，未生成公开页面。"
	}
	if created {
		return "文章已发布，站点已自动更新。"
	}
	return "文章已保存，站点已自动更新。"
}

type settingsResponse struct {
	Settings model.SiteSettings `json:"settings"`
	Message  string             `json:"message,omitempty"`
}
