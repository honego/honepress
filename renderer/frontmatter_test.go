package renderer

import "testing"

func TestParsePostDocumentStripsFrontMatter(t *testing.T) {
	markdownContent := []byte("---\ntitle: \"标题\"\ndate: \"2026-05-04 12:00:00\"\ndescription: \"摘要\"\ndraft: false\nurl: \"1.html\"\ncomments: true\naliases: []\n---\n\n正文内容")

	frontMatter, bodyMarkdownContent, err := ParsePostDocument("1.md", markdownContent)
	if err != nil {
		t.Fatalf("解析失败：%v", err)
	}
	if frontMatter.Title != "标题" {
		t.Fatalf("标题不一致：%s", frontMatter.Title)
	}
	if bodyMarkdownContent != "正文内容" {
		t.Fatalf("正文没有正确剥离 Front Matter：%q", bodyMarkdownContent)
	}
}
