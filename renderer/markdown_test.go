package renderer

import (
	"strings"
	"testing"
)

func TestMarkdownRendererRendersEmojiShortcodes(t *testing.T) {
	renderedHTML, err := NewMarkdownRenderer().Render("Hello :sparkles:")
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	renderedText := string(renderedHTML)
	if strings.Contains(renderedText, ":sparkles:") || !strings.Contains(renderedText, "&#x2728;") {
		t.Fatalf("emoji shortcode was not rendered: %s", renderedText)
	}
}
