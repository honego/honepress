package renderer

import "testing"

func TestParsePostDocumentStripsFrontMatter(t *testing.T) {
	markdownContent := []byte("---\ntitle: \"Title\"\nicon: \":sparkles:\"\ndate: \"2026-05-04 12:00:00\"\ndescription: \"Summary\"\nseoTitle: \"Custom SEO Title\"\nseoDescription: \"Custom SEO Description\"\ndraft: false\nurl: \"1.html\"\naliases: []\ntags:\n  - Go\n  - Blog\n---\n\nBody content")

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
	if frontMatter.SEOTitle != "Custom SEO Title" {
		t.Fatalf("SEO title mismatch: %s", frontMatter.SEOTitle)
	}
	if frontMatter.SEODescription != "Custom SEO Description" {
		t.Fatalf("SEO description mismatch: %s", frontMatter.SEODescription)
	}
	if bodyMarkdownContent != "Body content" {
		t.Fatalf("front matter was not removed: %q", bodyMarkdownContent)
	}
}
