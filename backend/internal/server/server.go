package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/core"
	"github.com/honeok/honepress/internal/model"
	"github.com/honeok/honepress/internal/service"
)

// HTTP 服务
type Server struct {
	optionsMutex      sync.RWMutex
	options           config.Options
	blogService       *service.BlogService
	adminSessionToken string
}

// 创建 HTTP 服务实例
func New(options config.Options, blogService *service.BlogService) (*Server, error) {
	adminSessionToken, err := newAdminSessionToken()
	if err != nil {
		return nil, fmt.Errorf("create admin session token: %w", err)
	}
	return &Server{
		options:           options,
		blogService:       blogService,
		adminSessionToken: adminSessionToken,
	}, nil
}

// 启动 HTTP 服务
func (server *Server) ListenAndServe() error {
	return http.ListenAndServe(core.DefaultAddress, server.routes())
}

func (server *Server) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(securityHeaders)

	router.Post("/api/login", server.apiHandler(server.handleLogin))
	router.Post("/api/logout", server.apiHandler(server.handleLogout))

	router.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(server.adminAuth)
		apiRouter.NotFound(server.apiHandler(func(_ http.ResponseWriter, _ *http.Request) error {
			return newResponseErrorMessage(http.StatusNotFound, "api endpoint not found")
		}))
		apiRouter.Get("/health", server.apiHandler(server.handleHealth))
		apiRouter.Get("/posts", server.apiHandler(server.handleListPosts))
		apiRouter.Post("/posts", server.apiHandler(server.handleCreatePost))
		apiRouter.Post("/preview", server.apiHandler(server.handlePreview))
		apiRouter.Get("/settings", server.apiHandler(server.handleGetSettings))
		apiRouter.Put("/settings", server.apiHandler(server.handleUpdateSettings))
		apiRouter.Get("/posts/{postID}", server.apiHandler(server.handleGetPost))
		apiRouter.Put("/posts/{postID}", server.apiHandler(server.handleUpdatePost))
		apiRouter.Delete("/posts/{postID}", server.apiHandler(server.handleDeletePost))
	})
	router.Group(func(adminRouter chi.Router) {
		adminRouter.Get("/admin", server.redirectAdmin)
		adminRouter.Get("/admin/*", server.serveAdmin)
	})
	router.HandleFunc("/*", server.servePublic)

	return router
}

const adminSessionCookieName = "honepress_admin_session"

type apiHandler func(http.ResponseWriter, *http.Request) error

type responseError struct {
	statusCode int
	err        error
}

func newResponseError(statusCode int, err error) error {
	if err == nil {
		err = errors.New(http.StatusText(statusCode))
	}
	return responseError{statusCode: statusCode, err: err}
}

func newResponseErrorMessage(statusCode int, message string) error {
	return newResponseError(statusCode, errors.New(message))
}

func (err responseError) Error() string {
	return err.err.Error()
}

func (err responseError) Unwrap() error {
	return err.err
}

func (server *Server) apiHandler(handler apiHandler) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		if err := handler(responseWriter, request); err != nil {
			statusCode := http.StatusInternalServerError
			errorMessage := err.Error()
			var responseErr responseError
			if errors.As(err, &responseErr) {
				statusCode = responseErr.statusCode
				errorMessage = responseErr.Error()
			}
			if err := server.writeError(responseWriter, statusCode, errorMessage); err != nil {
				log.Printf("write API error response: %v", err)
			}
		}
	}
}

func (server *Server) adminAuth(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		_, adminPassword := server.adminCredentials()
		if adminPassword == "" || server.hasValidAdminSession(request) || server.hasValidBasicAuth(request) {
			nextHandler.ServeHTTP(responseWriter, request)
			return
		}

		if err := server.writeError(responseWriter, http.StatusUnauthorized, "authentication required"); err != nil {
			log.Printf("write authentication error response: %v", err)
		}
	})
}

func (server *Server) hasValidBasicAuth(request *http.Request) bool {
	username, password, hasCredentials := request.BasicAuth()
	if !hasCredentials {
		return false
	}
	adminUsername, adminPassword := server.adminCredentials()
	if adminPassword == "" {
		return true
	}
	usernameMatches := subtle.ConstantTimeCompare([]byte(username), []byte(adminUsername)) == 1
	passwordMatches := subtle.ConstantTimeCompare([]byte(password), []byte(adminPassword)) == 1
	return usernameMatches && passwordMatches
}

func (server *Server) adminCredentials() (string, string) {
	server.optionsMutex.RLock()
	defer server.optionsMutex.RUnlock()

	return server.options.AdminUsername, server.options.AdminPassword
}

func (server *Server) setAdminCredentials(username string, password string) {
	server.optionsMutex.Lock()
	defer server.optionsMutex.Unlock()

	server.options.AdminUsername = username
	server.options.AdminPassword = password
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

func newAdminSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(tokenBytes), nil
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
	if _, err := responseWriter.Write(indexFileContent); err != nil {
		log.Printf("write admin index response: %v", err)
	}
}

func (server *Server) serveAdminFallback(responseWriter http.ResponseWriter, cause error) {
	if err := server.writeError(responseWriter, http.StatusInternalServerError, "admin frontend assets are missing; build frontend/admin: "+cause.Error()); err != nil {
		log.Printf("write admin fallback response: %v", err)
	}
}

func (server *Server) servePublic(responseWriter http.ResponseWriter, request *http.Request) {
	publicFileServer := http.FileServer(http.Dir(server.options.PublicDir))
	publicFileServer.ServeHTTP(responseWriter, request)
}

func (server *Server) handleHealth(responseWriter http.ResponseWriter, _ *http.Request) error {
	return server.writeJSON(responseWriter, http.StatusOK, healthResponse{Status: "ok"})
}

func (server *Server) handleLogin(responseWriter http.ResponseWriter, request *http.Request) error {
	adminUsername, adminPassword := server.adminCredentials()
	if adminPassword == "" {
		server.setAdminSessionCookie(responseWriter, request)
		return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
	}

	var loginRequest adminLoginRequest
	if err := server.decodeJSON(request, &loginRequest); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	usernameMatches := subtle.ConstantTimeCompare([]byte(loginRequest.Username), []byte(adminUsername)) == 1
	passwordMatches := subtle.ConstantTimeCompare([]byte(loginRequest.Password), []byte(adminPassword)) == 1
	if !usernameMatches || !passwordMatches {
		return newResponseErrorMessage(http.StatusUnauthorized, "invalid username or password")
	}

	server.setAdminSessionCookie(responseWriter, request)
	return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
}

func (server *Server) handleLogout(responseWriter http.ResponseWriter, request *http.Request) error {
	clearAdminSessionCookie(responseWriter, request)
	return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
}

func (server *Server) handleListPosts(responseWriter http.ResponseWriter, _ *http.Request) error {
	postSummaries, err := server.blogService.ListPosts()
	if err != nil {
		return newResponseError(http.StatusInternalServerError, err)
	}
	return server.writeJSON(responseWriter, http.StatusOK, postsResponse{Posts: postSummaries})
}

func (server *Server) handleCreatePost(responseWriter http.ResponseWriter, request *http.Request) error {
	var savePostRequest model.SavePostRequest
	if err := server.decodeJSON(request, &savePostRequest); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}

	createdPost, err := server.blogService.CreatePost(savePostRequest)
	if err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	return server.writeJSON(responseWriter, http.StatusCreated, postDetailResponse{Post: createdPost, Message: postSaveMessage(createdPost, true)})
}

func (server *Server) handlePreview(responseWriter http.ResponseWriter, request *http.Request) error {
	var previewRequest model.PreviewRequest
	if err := server.decodeJSON(request, &previewRequest); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}

	renderedHTML, err := server.blogService.PreviewMarkdown(previewRequest.Markdown)
	if err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	responseWriter.Header().Set("Content-Type", "text/html; charset=utf-8")
	responseWriter.WriteHeader(http.StatusOK)
	if _, err := responseWriter.Write([]byte(renderedHTML)); err != nil {
		return fmt.Errorf("write preview response: %w", err)
	}
	return nil
}

func (server *Server) handleGetSettings(responseWriter http.ResponseWriter, _ *http.Request) error {
	return server.writeJSON(responseWriter, http.StatusOK, settingsResponse{Settings: server.blogService.GetSiteSettings()})
}

func (server *Server) handleUpdateSettings(responseWriter http.ResponseWriter, request *http.Request) error {
	var settings model.SiteSettings
	if err := server.decodeJSON(request, &settings); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	if err := server.blogService.UpdateSiteSettings(settings); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	updatedSettings := server.blogService.GetSiteSettings()
	server.setAdminCredentials(updatedSettings.AdminUsername, updatedSettings.AdminPassword)
	server.setAdminSessionCookie(responseWriter, request)
	return server.writeJSON(responseWriter, http.StatusOK, settingsResponse{
		Settings: updatedSettings,
		Message:  "ok",
	})
}

func (server *Server) handleGetPost(responseWriter http.ResponseWriter, request *http.Request) error {
	postDetail, err := server.blogService.GetPost(chi.URLParam(request, "postID"))
	if err != nil {
		return newResponseError(http.StatusNotFound, err)
	}
	return server.writeJSON(responseWriter, http.StatusOK, postDetailResponse{Post: postDetail})
}

func (server *Server) handleUpdatePost(responseWriter http.ResponseWriter, request *http.Request) error {
	var savePostRequest model.SavePostRequest
	if err := server.decodeJSON(request, &savePostRequest); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	updatedPost, err := server.blogService.UpdatePost(chi.URLParam(request, "postID"), savePostRequest)
	if err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	return server.writeJSON(responseWriter, http.StatusOK, postDetailResponse{Post: updatedPost, Message: postSaveMessage(updatedPost, false)})
}

func (server *Server) handleDeletePost(responseWriter http.ResponseWriter, request *http.Request) error {
	if err := server.blogService.DeletePost(chi.URLParam(request, "postID")); err != nil {
		return newResponseError(http.StatusBadRequest, err)
	}
	return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
}

func (server *Server) decodeJSON(request *http.Request, targetValue interface{}) error {
	limitedRequestBody := io.LimitReader(request.Body, 8*1024*1024)
	jsonDecoder := json.NewDecoder(limitedRequestBody)
	jsonDecoder.DisallowUnknownFields()
	if err := jsonDecoder.Decode(targetValue); err != nil {
		return fmt.Errorf("decode request JSON: %w", err)
	}
	return nil
}

func (server *Server) writeJSON(responseWriter http.ResponseWriter, statusCode int, responseValue interface{}) error {
	responseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
	responseWriter.WriteHeader(statusCode)
	if err := json.NewEncoder(responseWriter).Encode(responseValue); err != nil {
		return fmt.Errorf("encode JSON response: %w", err)
	}
	return nil
}

func (server *Server) writeError(responseWriter http.ResponseWriter, statusCode int, errorMessage string) error {
	if err := server.writeJSON(responseWriter, statusCode, model.APIErrorResponse{Error: errorMessage}); err != nil {
		return fmt.Errorf("write error response: %w", err)
	}
	return nil
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
			return "draft_created"
		}
		return "draft_saved"
	}
	if created {
		return "post_created"
	}
	return "post_saved"
}

type settingsResponse struct {
	Settings model.SiteSettings `json:"settings"`
	Message  string             `json:"message,omitempty"`
}
