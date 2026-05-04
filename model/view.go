package model

import "html/template"

// TemplateLabels 保存不同语言页面上的固定文案，避免模板里写条件分支。
type TemplateLabels struct {
	Home             string
	Blog             string
	RSS              string
	LatestPosts      string
	AllPosts         string
	ReadMore         string
	PublishedAt      string
	NoPosts          string
	LanguageSwitch   string
	BackToList       string
	Footer           string
	ThemeButtonLabel string
}

// SiteViewData 是首页和列表页共享的模板数据。
type SiteViewData struct {
	SiteTitle       string
	SiteDescription string
	BaseURL         string
	GitHubURL       string
	TelegramURL     string
	ThemeDefault    string
	Language        string
	CanonicalPath   string
	HomePath        string
	BlogPath        string
	RSSPath         string
	AlternatePath   string
	Labels          TemplateLabels
	Posts           []PostSummary
}

// PostViewData 是文章详情页模板数据。
type PostViewData struct {
	SiteTitle       string
	SiteDescription string
	BaseURL         string
	GitHubURL       string
	TelegramURL     string
	ThemeDefault    string
	Language        string
	CanonicalPath   string
	HomePath        string
	BlogPath        string
	RSSPath         string
	AlternatePath   string
	Labels          TemplateLabels
	Post            Post
	CommentHTML     template.HTML
}
