package model

import "html/template"

// 站点模板数据
type SiteViewData struct {
	SiteTitle       string
	SiteDescription string
	SiteIconURL     string
	FaviconHref     template.URL
	ThemeDefault    string
	Font            string
	ThemeScriptPath template.URL
	ThemeStylePaths []template.URL
	CanonicalPath   string
	CanonicalURL    template.URL
	SEOTitle        string
	SEODescription  string
	SEOType         string
	SEOImage        template.URL
	StructuredData  template.JS
	HomePath        string
	BlogPath        string
	RSSPath         string
	SitemapPath     string
	Posts           []PostSummary
	PostCount       int
	WordCount       int
}

// 文章模板数据
type PostViewData struct {
	SiteTitle       string
	SiteDescription string
	SiteIconURL     string
	FaviconHref     template.URL
	ThemeDefault    string
	Font            string
	ThemeScriptPath template.URL
	ThemeStylePaths []template.URL
	CanonicalPath   string
	CanonicalURL    template.URL
	SEOTitle        string
	SEODescription  string
	SEOType         string
	SEOImage        template.URL
	StructuredData  template.JS
	HomePath        string
	BlogPath        string
	RSSPath         string
	SitemapPath     string
	Post            Post
	CommentHTML     template.HTML
}
