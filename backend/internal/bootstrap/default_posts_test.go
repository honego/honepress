package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateDefaultPostIfEmptyCreatesFirstPost(t *testing.T) {
	postsDir := filepath.Join(t.TempDir(), "posts")

	if err := GenerateDefaultPostIfEmpty(postsDir); err != nil {
		t.Fatalf("generate default post failed: %v", err)
	}

	defaultPostContent, err := os.ReadFile(filepath.Join(postsDir, defaultPostFileName))
	if err != nil {
		t.Fatalf("read default post failed: %v", err)
	}
	defaultPostText := string(defaultPostContent)
	for _, expectedSnippet := range []string{`title: "世界你好"`, `url: "hello.html"`, "欢迎使用 HonePress"} {
		if !strings.Contains(defaultPostText, expectedSnippet) {
			t.Fatalf("default post missing %q in:\n%s", expectedSnippet, defaultPostText)
		}
	}
}

func TestGenerateDefaultPostIfEmptyLeavesExistingPostsUntouched(t *testing.T) {
	postsDir := filepath.Join(t.TempDir(), "posts")
	if err := os.MkdirAll(postsDir, 0755); err != nil {
		t.Fatalf("create posts failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "existing.md"), []byte("existing"), 0644); err != nil {
		t.Fatalf("write existing post failed: %v", err)
	}

	if err := GenerateDefaultPostIfEmpty(postsDir); err != nil {
		t.Fatalf("generate default post failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(postsDir, defaultPostFileName)); !os.IsNotExist(err) {
		t.Fatalf("default post must not be generated when posts already exist")
	}
}

func TestGenerateDefaultPostIfEmptyIgnoresNonMarkdownFiles(t *testing.T) {
	postsDir := filepath.Join(t.TempDir(), "posts")
	if err := os.MkdirAll(postsDir, 0755); err != nil {
		t.Fatalf("create posts failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(postsDir, "note.txt"), []byte("ignored"), 0644); err != nil {
		t.Fatalf("write text file failed: %v", err)
	}

	if err := GenerateDefaultPostIfEmpty(postsDir); err != nil {
		t.Fatalf("generate default post failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(postsDir, defaultPostFileName)); err != nil {
		t.Fatalf("default post should be generated when only non-markdown files exist: %v", err)
	}
}
