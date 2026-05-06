package renderer

import (
	"strings"
	"testing"
)

func TestMarkdownRendererRendersEmojiShortcodes(t *testing.T) {
	renderedHTML, err := NewMarkdownRenderer().Render("Hello :sparkles:")
	if err != nil {
		t.Fatalf("渲染失败：%v", err)
	}
	renderedText := string(renderedHTML)
	if strings.Contains(renderedText, ":sparkles:") || !strings.Contains(renderedText, "&#x2728;") {
		t.Fatalf("emoji 短码未渲染：%s", renderedText)
	}
}
