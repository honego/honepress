package model

// 后台站点设置
type SiteSettings struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	BaseURL        string `json:"baseUrl"`
	Language       string `json:"language"`
	GitHubURL      string `json:"githubUrl"`
	TelegramURL    string `json:"telegramUrl"`
	CommentEnabled bool   `json:"commentEnabled"`
	ThemeDefault   string `json:"themeDefault"`
}
