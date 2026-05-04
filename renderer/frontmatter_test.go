package renderer

import "testing"

func TestParsePostDocumentStripsFrontMatter(t *testing.T) {
	markdownContent := []byte("---\ntitle: \"Title\"\nicon: \":sparkles:\"\ndate: \"2026-05-04 12:00:00\"\ndescription: \"Summary\"\ndraft: false\nurl: \"1.html\"\ncomments: true\naliases: []\ntags:\n  - Go\n  - Blog\n---\n\nBody content")

	frontMatter, bodyMarkdownContent, err := ParsePostDocument("1.md", markdownContent)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if frontMatter.Title != "Title" {
		t.Fatalf("title mismatch: %s", frontMatter.Title)
	}
	if frontMatter.Icon != ":sparkles:" {
		t.Fatalf("icon mismatch: %s", frontMatter.Icon)
	}
	if len(frontMatter.Tags) != 2 || frontMatter.Tags[0] != "Go" || frontMatter.Tags[1] != "Blog" {
		t.Fatalf("tags mismatch: %v", frontMatter.Tags)
	}
	if bodyMarkdownContent != "Body content" {
		t.Fatalf("front matter was not stripped: %q", bodyMarkdownContent)
	}
}
