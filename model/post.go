package model

import (
	"html/template"
	"time"
)

// Markdown Front Matter
type PostFrontMatter struct {
	Title       string   `yaml:"title"`
	Icon        string   `yaml:"icon"`
	Date        string   `yaml:"date"`
	Description string   `yaml:"description"`
	Draft       bool     `yaml:"draft"`
	URL         string   `yaml:"url"`
	Comments    bool     `yaml:"comments"`
	Aliases     []string `yaml:"aliases"`
	Tags        []string `yaml:"tags"`
}

// 完整文章模型
type Post struct {
	SourceFileName string
	SourceFilePath string
	Title          string
	Icon           string
	DateText       string
	PublishedAt    time.Time
	Description    string
	Draft          bool
	URL            string
	Aliases        []string
	Tags           []string
	Comments       bool
	BodyMarkdown   string
	BodyHTML       template.HTML
}

// 文章摘要
type PostSummary struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Icon        string   `json:"icon"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Draft       bool     `json:"draft"`
	URL         string   `json:"url"`
	PublicURL   string   `json:"publicUrl"`
	Comments    bool     `json:"comments"`
	Tags        []string `json:"tags"`
}

// 文章详情
type PostDetail struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Icon        string   `json:"icon"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Draft       bool     `json:"draft"`
	URL         string   `json:"url"`
	Aliases     []string `json:"aliases"`
	Tags        []string `json:"tags"`
	Comments    bool     `json:"comments"`
	Body        string   `json:"body"`
}

// 文章保存请求
type SavePostRequest struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Icon        string   `json:"icon"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Draft       bool     `json:"draft"`
	URL         string   `json:"url"`
	Aliases     []string `json:"aliases"`
	Tags        []string `json:"tags"`
	Comments    bool     `json:"comments"`
	Body        string   `json:"body"`
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
