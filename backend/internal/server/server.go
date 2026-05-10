package server

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
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
	optionsMutex sync.RWMutex
	options      config.Options
	blogService  *service.BlogService
	jwtSecret    []byte
	apiLimiter   *apiRateLimiter
}

// 创建 HTTP 服务实例
func New(options config.Options, blogService *service.BlogService) (*Server, error) {
	jwtSecret, err := resolveJWTSecret()
	if err != nil {
		return nil, fmt.Errorf("create JWT secret: %w", err)
	}
	return &Server{
		options:     options,
		blogService: blogService,
		jwtSecret:   jwtSecret,
		apiLimiter:  newAPIRateLimiter(),
	}, nil
}

// 启动 HTTP 服务
func (server *Server) ListenAndServe() error {
	return http.ListenAndServe(core.DefaultAddress, server.routes())
}

func (server *Server) routes() http.Handler {
	router := chi.NewRouter()
	router.Use(securityHeaders)

	router.Route("/api", func(apiRouter chi.Router) {
		apiRouter.Use(server.apiLimiter.middleware)
		apiRouter.Use(apiAccessLog)
		apiRouter.NotFound(server.apiHandler(func(_ http.ResponseWriter, _ *http.Request) error {
			return newResponseErrorMessage(http.StatusNotFound, "api endpoint not found")
		}))
		apiRouter.Post("/login", server.apiHandler(server.handleLogin))
		apiRouter.Post("/logout", server.apiHandler(server.handleLogout))
		apiRouter.Post("/refresh-token", server.apiHandler(server.handleRefreshToken))
		apiRouter.Get("/site", server.apiHandler(server.handleGetPublicSite))
		apiRouter.Get("/posts", server.apiHandler(server.handleListPublicPosts))
		apiRouter.Get("/posts/{postID}", server.apiHandler(server.handleGetPublicPost))

		apiRouter.Route("/admin", func(adminRouter chi.Router) {
			adminRouter.Use(server.adminAuth)
			adminRouter.Get("/health", server.apiHandler(server.handleHealth))
			adminRouter.Get("/me", server.apiHandler(server.handleMe))
			adminRouter.Get("/stats", server.apiHandler(server.handleAdminStats))
			adminRouter.Get("/posts", server.apiHandler(server.handleListPosts))
			adminRouter.Post("/posts", server.apiHandler(server.handleCreatePost))
			adminRouter.Post("/preview", server.apiHandler(server.handlePreview))
			adminRouter.Get("/settings", server.apiHandler(server.handleGetSettings))
			adminRouter.Put("/settings", server.apiHandler(server.handleUpdateSettings))
			adminRouter.Get("/posts/{postID}", server.apiHandler(server.handleGetPost))
			adminRouter.Put("/posts/{postID}", server.apiHandler(server.handleUpdatePost))
			adminRouter.Delete("/posts/{postID}", server.apiHandler(server.handleDeletePost))
		})
	})
	router.Group(func(adminRouter chi.Router) {
		adminRouter.Get("/admin", server.redirectAdmin)
		adminRouter.Get("/admin/*", server.serveAdmin)
	})
	router.HandleFunc("/*", server.servePublic)

	return router
}

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
		if adminPassword == "" {
			nextHandler.ServeHTTP(responseWriter, request)
			return
		}
		if _, ok := server.authenticatedClaims(request); ok {
			nextHandler.ServeHTTP(responseWriter, request)
			return
		}

		if err := server.writeError(responseWriter, http.StatusUnauthorized, "authentication required"); err != nil {
			log.Printf("write authentication error response: %v", err)
		}
	})
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

func securityHeaders(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
		responseWriter.Header().Set("X-Content-Type-Options", "nosniff")
		responseWriter.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		responseWriter.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
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
	if err == nil {
		if fileInfo.IsDir() {
			if server.serveStaticFile(responseWriter, request, server.options.AdminDistDir, filepath.ToSlash(filepath.Join(cleanRequestPath, "index.html"))) {
				return
			}
		} else if server.serveStaticFile(responseWriter, request, server.options.AdminDistDir, cleanRequestPath) {
			return
		}
	}

	if !strings.HasSuffix(cleanRequestPath, ".html") {
		if server.serveStaticFile(responseWriter, request, server.options.AdminDistDir, cleanRequestPath+".html") {
			return
		}
		if server.serveStaticFile(responseWriter, request, server.options.AdminDistDir, filepath.ToSlash(filepath.Join(cleanRequestPath, "index.html"))) {
			return
		}
	}

	server.serveAdminIndex(responseWriter)
}

func (server *Server) serveStaticFile(responseWriter http.ResponseWriter, request *http.Request, rootDirectory string, cleanRequestPath string) bool {
	targetFilePath := filepath.Join(rootDirectory, filepath.FromSlash(cleanRequestPath))
	fileInfo, err := os.Stat(targetFilePath)
	if err != nil || fileInfo.IsDir() {
		return false
	}
	fileServerRequest := request.Clone(request.Context())
	fileServerRequest.URL.Path = "/" + strings.TrimPrefix(filepath.ToSlash(cleanRequestPath), "/")
	http.FileServer(http.Dir(rootDirectory)).ServeHTTP(responseWriter, fileServerRequest)
	return true
}

func (server *Server) servePublic(responseWriter http.ResponseWriter, request *http.Request) {
	cleanRequestPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/"))
	if cleanRequestPath == "." || cleanRequestPath == "/" || strings.HasPrefix(cleanRequestPath, "..") {
		http.FileServer(http.Dir(server.options.PublicDir)).ServeHTTP(responseWriter, request)
		return
	}

	if server.serveStaticFile(responseWriter, request, server.options.PublicDir, cleanRequestPath) {
		return
	}

	if !strings.HasSuffix(cleanRequestPath, ".html") {
		if server.serveStaticFile(responseWriter, request, server.options.PublicDir, cleanRequestPath+".html") {
			return
		}
		if server.serveStaticFile(responseWriter, request, server.options.PublicDir, filepath.ToSlash(filepath.Join(cleanRequestPath, "index.html"))) {
			return
		}
	}
	if strings.EqualFold(cleanRequestPath, "blog.html") {
		if server.serveStaticFile(responseWriter, request, server.options.PublicDir, "archive.html") {
			return
		}
	}
	if strings.HasSuffix(strings.ToLower(cleanRequestPath), ".html") {
		if server.serveStaticFile(responseWriter, request, server.options.PublicDir, "posts.html") {
			return
		}
	}
	http.FileServer(http.Dir(server.options.PublicDir)).ServeHTTP(responseWriter, request)
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

func (server *Server) handleHealth(responseWriter http.ResponseWriter, _ *http.Request) error {
	return server.writeJSON(responseWriter, http.StatusOK, healthResponse{Status: "ok"})
}

func (server *Server) handleLogin(responseWriter http.ResponseWriter, request *http.Request) error {
	adminUsername, adminPassword := server.adminCredentials()
	userID := adminUserID(adminUsername)
	if adminPassword == "" {
		if err := server.setAuthCookie(responseWriter, userID, "admin"); err != nil {
			return err
		}
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

	if err := server.setAuthCookie(responseWriter, userID, "admin"); err != nil {
		return err
	}
	return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
}

func (server *Server) handleLogout(responseWriter http.ResponseWriter, _ *http.Request) error {
	clearAuthCookie(responseWriter)
	return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
}

func (server *Server) handleRefreshToken(responseWriter http.ResponseWriter, request *http.Request) error {
	claims, ok := server.authenticatedClaims(request)
	if !ok {
		return newResponseErrorMessage(http.StatusUnauthorized, "authentication required")
	}
	if err := server.setAuthCookie(responseWriter, claims.UserID, claims.Role); err != nil {
		return err
	}
	return server.writeJSON(responseWriter, http.StatusOK, model.APIMessageResponse{Message: "ok"})
}

func (server *Server) handleMe(responseWriter http.ResponseWriter, request *http.Request) error {
	claims, ok := server.authenticatedClaims(request)
	if !ok {
		adminUsername, adminPassword := server.adminCredentials()
		if adminPassword == "" {
			userID := adminUserID(adminUsername)
			if err := server.setAuthCookie(responseWriter, userID, "admin"); err != nil {
				return err
			}
			return server.writeJSON(responseWriter, http.StatusOK, meResponse{UserID: userID, Role: "admin"})
		}
		return newResponseErrorMessage(http.StatusUnauthorized, "authentication required")
	}
	return server.writeJSON(responseWriter, http.StatusOK, meResponse{UserID: claims.UserID, Role: claims.Role})
}

func (server *Server) handleGetPublicSite(responseWriter http.ResponseWriter, _ *http.Request) error {
	settings := server.blogService.GetSiteSettings()
	return server.writeJSON(responseWriter, http.StatusOK, publicSiteResponse{
		Site: publicSiteSettings{
			Title:            settings.Title,
			Description:      settings.Description,
			IconURL:          settings.IconURL,
			CommentEnabled:   settings.CommentEnabled,
			GiscusRepo:       settings.GiscusRepo,
			GiscusRepoID:     settings.GiscusRepoID,
			GiscusCategory:   settings.GiscusCategory,
			GiscusCategoryID: settings.GiscusCategoryID,
			ThemeDefault:     settings.ThemeDefault,
			Font:             settings.Font,
		},
	})
}

func (server *Server) handleListPublicPosts(responseWriter http.ResponseWriter, _ *http.Request) error {
	postSummaries, err := server.blogService.ListPublicPosts()
	if err != nil {
		return newResponseError(http.StatusInternalServerError, err)
	}
	return server.writeJSON(responseWriter, http.StatusOK, postsResponse{Posts: postSummaries})
}

func (server *Server) handleGetPublicPost(responseWriter http.ResponseWriter, request *http.Request) error {
	postDetail, err := server.blogService.GetPublicPost(chi.URLParam(request, "postID"))
	if err != nil {
		return newResponseError(http.StatusNotFound, err)
	}
	return server.writeJSON(responseWriter, http.StatusOK, publicPostDetailResponse{Post: postDetail})
}

func (server *Server) handleAdminStats(responseWriter http.ResponseWriter, _ *http.Request) error {
	postSummaries, err := server.blogService.ListPosts()
	if err != nil {
		return newResponseError(http.StatusInternalServerError, err)
	}
	stats := adminStatsResponse{TotalPosts: len(postSummaries)}
	for _, currentPost := range postSummaries {
		if currentPost.Draft {
			stats.DraftPosts++
		} else {
			stats.PublishedPosts++
		}
	}
	return server.writeJSON(responseWriter, http.StatusOK, stats)
}

func (server *Server) handleListPosts(responseWriter http.ResponseWriter, request *http.Request) error {
	postSummaries, err := server.blogService.ListPosts()
	if err != nil {
		return newResponseError(http.StatusInternalServerError, err)
	}
	filteredPosts := filterAdminPosts(postSummaries, request)
	page, pageSize := paginationFromRequest(request)
	pagedPosts, totalPages := paginatePosts(filteredPosts, page, pageSize)
	return server.writeJSON(responseWriter, http.StatusOK, adminPostsResponse{
		Posts:      pagedPosts,
		Page:       page,
		PageSize:   pageSize,
		Total:      len(filteredPosts),
		TotalPages: totalPages,
	})
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
	if err := server.setAuthCookie(responseWriter, adminUserID(updatedSettings.AdminUsername), "admin"); err != nil {
		return err
	}
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

type adminPostsResponse struct {
	Posts      []model.PostSummary `json:"posts"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	Total      int                 `json:"total"`
	TotalPages int                 `json:"totalPages"`
}

type postDetailResponse struct {
	Post    model.PostDetail `json:"post"`
	Message string           `json:"message,omitempty"`
}

type publicPostDetailResponse struct {
	Post model.PublicPostDetail `json:"post"`
}

type publicSiteResponse struct {
	Site publicSiteSettings `json:"site"`
}

type publicSiteSettings struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	IconURL          string `json:"iconUrl"`
	CommentEnabled   bool   `json:"commentEnabled"`
	GiscusRepo       string `json:"giscusRepo"`
	GiscusRepoID     string `json:"giscusRepoId"`
	GiscusCategory   string `json:"giscusCategory"`
	GiscusCategoryID string `json:"giscusCategoryId"`
	ThemeDefault     string `json:"themeDefault"`
	Font             string `json:"font"`
}

type meResponse struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type adminStatsResponse struct {
	TotalPosts     int `json:"totalPosts"`
	PublishedPosts int `json:"publishedPosts"`
	DraftPosts     int `json:"draftPosts"`
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

func adminUserID(adminUsername string) string {
	if trimmedUsername := strings.TrimSpace(adminUsername); trimmedUsername != "" {
		return trimmedUsername
	}
	return "admin"
}

func paginationFromRequest(request *http.Request) (int, int) {
	page := positiveQueryInt(request, "page", 1)
	pageSize := positiveQueryInt(request, "pageSize", 10)
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func positiveQueryInt(request *http.Request, key string, fallback int) int {
	rawValue := strings.TrimSpace(request.URL.Query().Get(key))
	if rawValue == "" {
		return fallback
	}
	parsedValue, err := strconv.Atoi(rawValue)
	if err != nil || parsedValue < 1 {
		return fallback
	}
	return parsedValue
}

func filterAdminPosts(posts []model.PostSummary, request *http.Request) []model.PostSummary {
	query := request.URL.Query()
	searchText := strings.ToLower(strings.TrimSpace(query.Get("search")))
	draftFilter := strings.ToLower(strings.TrimSpace(query.Get("draft")))

	filteredPosts := make([]model.PostSummary, 0, len(posts))
	for _, currentPost := range posts {
		if draftFilter == "true" && !currentPost.Draft {
			continue
		}
		if draftFilter == "false" && currentPost.Draft {
			continue
		}
		if searchText != "" && !postMatchesSearch(currentPost, searchText) {
			continue
		}
		filteredPosts = append(filteredPosts, currentPost)
	}
	return filteredPosts
}

func postMatchesSearch(post model.PostSummary, searchText string) bool {
	searchFields := []string{
		post.Title,
		post.Description,
		post.URL,
		strings.Join(post.Tags, " "),
	}
	for _, searchField := range searchFields {
		if strings.Contains(strings.ToLower(searchField), searchText) {
			return true
		}
	}
	return false
}

func paginatePosts(posts []model.PostSummary, page int, pageSize int) ([]model.PostSummary, int) {
	if pageSize < 1 {
		pageSize = 10
	}
	totalPages := (len(posts) + pageSize - 1) / pageSize
	if totalPages == 0 {
		totalPages = 1
	}
	if page > totalPages {
		return []model.PostSummary{}, totalPages
	}
	startIndex := (page - 1) * pageSize
	endIndex := startIndex + pageSize
	if endIndex > len(posts) {
		endIndex = len(posts)
	}
	return posts[startIndex:endIndex], totalPages
}
