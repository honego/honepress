package model

// SiteSettings 是后台配置页允许修改的站点配置字段。
type SiteSettings struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	BaseURL            string `json:"baseUrl"`
	Language           string `json:"language"`
	GitHubURL          string `json:"githubUrl"`
	TelegramURL        string `json:"telegramUrl"`
	CommentEnabled     bool   `json:"commentEnabled"`
	TranslationEnabled bool   `json:"translationEnabled"`
	ThemeDefault       string `json:"themeDefault"`
}
