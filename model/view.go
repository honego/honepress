package model

import "html/template"

// 模板固定文案
type TemplateLabels struct {
	Home             string
	Blog             string
	RSS              string
	LatestPosts      string
	AllPosts         string
	ReadMore         string
	PublishedAt      string
	NoPosts          string
	BackToList       string
	Footer           string
	ThemeButtonLabel string
}

// 站点模板数据
type SiteViewData struct {
	SiteTitle       string
	SiteDescription string
	SiteIconURL     string
	ThemeDefault    string
	Font            string
	CanonicalPath   string
	HomePath        string
	BlogPath        string
	RSSPath         string
	Labels          TemplateLabels
	Posts           []PostSummary
}

// 文章模板数据
type PostViewData struct {
	SiteTitle       string
	SiteDescription string
	SiteIconURL     string
	ThemeDefault    string
	Font            string
	CanonicalPath   string
	HomePath        string
	BlogPath        string
	RSSPath         string
	Labels          TemplateLabels
	Post            Post
	CommentHTML     template.HTML
}
