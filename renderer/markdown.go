package renderer

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	goldmarkHTML "github.com/yuin/goldmark/renderer/html"
)

// MarkdownRenderer 封装 goldmark，保证前台页面和后台预览使用同一套渲染规则。
type MarkdownRenderer struct {
	markdown goldmark.Markdown
}

// NewMarkdownRenderer 创建 Markdown 渲染器，GFM 扩展能覆盖常见博客写作习惯。
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithRendererOptions(goldmarkHTML.WithUnsafe()),
		),
	}
}

// Render 把 Markdown 正文转换为 HTML，调用方负责在模板中控制输出位置。
func (markdownRenderer *MarkdownRenderer) Render(markdownContent string) (template.HTML, error) {
	var renderedHTMLBuffer bytes.Buffer
	if err := markdownRenderer.markdown.Convert([]byte(markdownContent), &renderedHTMLBuffer); err != nil {
		return "", fmt.Errorf("渲染 Markdown 失败：%w", err)
	}
	return template.HTML(renderedHTMLBuffer.String()), nil
}
