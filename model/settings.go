package model

// 后台站点设置
type SiteSettings struct {
	Title                  string `json:"title"`
	Description            string `json:"description"`
	BaseURL                string `json:"baseUrl"`
	Language               string `json:"language"`
	IconURL                string `json:"iconUrl"`
	GitHubURL              string `json:"githubUrl"`
	TelegramURL            string `json:"telegramUrl"`
	CommentEnabled         bool   `json:"commentEnabled"`
	CommentProvider        string `json:"commentProvider"`
	GiscusRepo             string `json:"giscusRepo"`
	GiscusRepoID           string `json:"giscusRepoId"`
	GiscusCategory         string `json:"giscusCategory"`
	GiscusCategoryID       string `json:"giscusCategoryId"`
	GiscusMapping          string `json:"giscusMapping"`
	GiscusStrict           string `json:"giscusStrict"`
	GiscusReactionsEnabled string `json:"giscusReactionsEnabled"`
	GiscusEmitMetadata     string `json:"giscusEmitMetadata"`
	GiscusInputPosition    string `json:"giscusInputPosition"`
	GiscusTheme            string `json:"giscusTheme"`
	GiscusLang             string `json:"giscusLang"`
	ThemeDefault           string `json:"themeDefault"`
}
