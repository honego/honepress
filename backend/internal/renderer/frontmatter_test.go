package renderer

import "testing"

func TestParsePostDocumentStripsFrontMatter(t *testing.T) {
	markdownContent := []byte("---\ntitle: \"Title\"\nicon: \":sparkles:\"\ndate: \"2026-05-04 12:00:00\"\ndescription: \"Summary\"\nseoTitle: \"Custom SEO Title\"\nseoDescription: \"Custom SEO Description\"\ndraft: false\nurl: \"1.html\"\naliases: []\ntags:\n  - Go\n  - Blog\n---\n\nBody content")

	frontMatter, bodyMarkdownContent, err := ParsePostDocument("1.md", markdownContent)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if frontMatter.Title != "Title" {
		t.Fatalf("标题不一致：%s", frontMatter.Title)
	}
	if frontMatter.Icon != ":sparkles:" {
		t.Fatalf("icon 不一致：%s", frontMatter.Icon)
	}
	if len(frontMatter.Tags) != 2 || frontMatter.Tags[0] != "Go" || frontMatter.Tags[1] != "Blog" {
		t.Fatalf("标签不一致：%v", frontMatter.Tags)
	}
	if frontMatter.SEOTitle != "Custom SEO Title" {
		t.Fatalf("SEO 标题不一致：%s", frontMatter.SEOTitle)
	}
	if frontMatter.SEODescription != "Custom SEO Description" {
		t.Fatalf("SEO 描述不一致：%s", frontMatter.SEODescription)
	}
	if bodyMarkdownContent != "Body content" {
		t.Fatalf("文章元信息没有被移除：%q", bodyMarkdownContent)
	}
}
