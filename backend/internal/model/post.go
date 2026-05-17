package model

import (
	"html/template"
	"time"
)

// Markdown Front Matter
type PostFrontMatter struct {
	Title          string   `yaml:"title"`
	Icon           string   `yaml:"icon,omitempty"`
	Thumbnail      string   `yaml:"thumbnail,omitempty"`
	Date           string   `yaml:"date"`
	Description    string   `yaml:"description"`
	SEOTitle       string   `yaml:"seoTitle,omitempty"`
	SEODescription string   `yaml:"seoDescription,omitempty"`
	Draft          bool     `yaml:"draft"`
	URL            string   `yaml:"url"`
	Tags           []string `yaml:"tags"`
}

// 完整文章模型
type Post struct {
	SourceFileName string
	SourceFilePath string
	Title          string
	Icon           string
	Thumbnail      string
	DateText       string
	PublishedAt    time.Time
	Description    string
	SEOTitle       string
	SEODescription string
	Draft          bool
	SourceURL      string
	Slug           string
	PostID         string
	URL            string
	OutputPath     string
	Tags           []string
	BodyMarkdown   string
	BodyHTML       template.HTML
}

// 文章摘要
type PostSummary struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Thumbnail   string   `json:"thumbnail"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Draft       bool     `json:"draft"`
	URL         string   `json:"url"`
	PublicURL   string   `json:"publicUrl"`
	Tags        []string `json:"tags"`
	WordCount   int      `json:"wordCount"`
}

// 文章详情
type PostDetail struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Icon           string   `json:"icon"`
	Thumbnail      string   `json:"thumbnail"`
	Date           string   `json:"date"`
	Description    string   `json:"description"`
	SEOTitle       string   `json:"seoTitle"`
	SEODescription string   `json:"seoDescription"`
	Draft          bool     `json:"draft"`
	URL            string   `json:"url"`
	Tags           []string `json:"tags"`
	Body           string   `json:"body"`
}

type PublicPostDetail struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Icon           string   `json:"icon"`
	Thumbnail      string   `json:"thumbnail"`
	Date           string   `json:"date"`
	Description    string   `json:"description"`
	SEOTitle       string   `json:"seoTitle"`
	SEODescription string   `json:"seoDescription"`
	URL            string   `json:"url"`
	PublicURL      string   `json:"publicUrl"`
	Tags           []string `json:"tags"`
	HTML           string   `json:"html"`
}

// 文章保存请求
type SavePostRequest struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Icon           string   `json:"icon"`
	Thumbnail      string   `json:"thumbnail"`
	Date           string   `json:"date"`
	Description    string   `json:"description"`
	SEOTitle       string   `json:"seoTitle"`
	SEODescription string   `json:"seoDescription"`
	Draft          bool     `json:"draft"`
	URL            string   `json:"url"`
	Tags           []string `json:"tags"`
	Body           string   `json:"body"`
}

// Markdown 预览请求
type PreviewRequest struct {
	Markdown string `json:"markdown"`
}

// 通用消息响应
type APIMessageResponse struct {
	Message string `json:"message"`
}

// 通用错误响应
type APIErrorResponse struct {
	Error string `json:"error"`
}
