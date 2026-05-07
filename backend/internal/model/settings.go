package model

// 后台站点设置
type SiteSettings struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	IconURL          string `json:"iconUrl"`
	AdminUsername    string `json:"adminUsername"`
	AdminPassword    string `json:"adminPassword"`
	CommentEnabled   bool   `json:"commentEnabled"`
	GiscusRepo       string `json:"giscusRepo"`
	GiscusRepoID     string `json:"giscusRepoId"`
	GiscusCategory   string `json:"giscusCategory"`
	GiscusCategoryID string `json:"giscusCategoryId"`
	ThemeDefault     string `json:"themeDefault"`
	Font             string `json:"font"`
}
