package renderer

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/yuin/goldmark"
	emoji "github.com/yuin/goldmark-emoji"
	"github.com/yuin/goldmark/extension"
	goldmarkHTML "github.com/yuin/goldmark/renderer/html"
)

// Markdown 渲染器
type MarkdownRenderer struct {
	markdown goldmark.Markdown
}

// 创建 Markdown 渲染器
func NewMarkdownRenderer() *MarkdownRenderer {
	return &MarkdownRenderer{
		markdown: goldmark.New(
			goldmark.WithExtensions(extension.GFM, emoji.Emoji),
			goldmark.WithRendererOptions(goldmarkHTML.WithUnsafe()),
		),
	}
}

// 把 Markdown 正文转换为 HTML
func (markdownRenderer *MarkdownRenderer) Render(markdownContent string) (template.HTML, error) {
	var renderedHTMLBuffer bytes.Buffer
	if err := markdownRenderer.markdown.Convert([]byte(markdownContent), &renderedHTMLBuffer); err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return template.HTML(renderedHTMLBuffer.String()), nil
}
