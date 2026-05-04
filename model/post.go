package model

import (
	"html/template"
	"time"
)

// PostFrontMatter 表示中文 Markdown 文件开头的 Front Matter。
type PostFrontMatter struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Description string   `yaml:"description"`
	Draft       bool     `yaml:"draft"`
	URL         string   `yaml:"url"`
	Comments    bool     `yaml:"comments"`
	Aliases     []string `yaml:"aliases"`
}

// Post 是渲染阶段使用的完整文章模型。
type Post struct {
	SourceFileName string
	SourceFilePath string
	Title          string
	DateText       string
	PublishedAt    time.Time
	Description    string
	Draft          bool
	URL            string
	Aliases        []string
	Comments       bool
	BodyMarkdown   string
	BodyHTML       template.HTML
	Language       string
}

// PostSummary 是模板和后台列表共同使用的轻量文章摘要。
type PostSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Date        string `json:"date"`
	Description string `json:"description"`
	Draft       bool   `json:"draft"`
	URL         string `json:"url"`
	PublicURL   string `json:"publicUrl"`
	Comments    bool   `json:"comments"`
}

// PostDetail 是后台编辑页读取单篇文章时返回的完整数据。
type PostDetail struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Draft       bool     `json:"draft"`
	URL         string   `json:"url"`
	Aliases     []string `json:"aliases"`
	Comments    bool     `json:"comments"`
	Body        string   `json:"body"`
}

// SavePostRequest 是新建和更新文章时后台提交的数据结构。
type SavePostRequest struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Date        string   `json:"date"`
	Description string   `json:"description"`
	Draft       bool     `json:"draft"`
	URL         string   `json:"url"`
	Aliases     []string `json:"aliases"`
	Comments    bool     `json:"comments"`
	Body        string   `json:"body"`
}

// PreviewRequest 是 Markdown 预览接口的请求体。
type PreviewRequest struct {
	Markdown string `json:"markdown"`
}

// APIMessageResponse 是只需要返回中文状态信息的通用响应体。
type APIMessageResponse struct {
	Message string `json:"message"`
}

// APIErrorResponse 是所有 JSON API 的错误响应体。
type APIErrorResponse struct {
	Error string `json:"error"`
}
