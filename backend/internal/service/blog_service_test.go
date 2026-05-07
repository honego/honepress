package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honeok/honepress/internal/config"
)

func withTestRuntimeFiles(t *testing.T, dataDirectoryPath string, testOptions config.Options) config.Options {
	t.Helper()

	themeDistDir := filepath.Join(dataDirectoryPath, "theme-dist")
	if err := os.MkdirAll(themeDistDir, 0755); err != nil {
		t.Fatalf("create test theme dist directory failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDistDir, "theme.js"), []byte("https://giscus.app/client.js"), 0644); err != nil {
		t.Fatalf("write test theme script failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDistDir, "favicon.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0644); err != nil {
		t.Fatalf("write test favicon failed: %v", err)
	}

	templateDir, err := filepath.Abs(filepath.Join("..", "..", "..", "frontend", "theme", "templates"))
	if err != nil {
		t.Fatalf("resolve test template directory failed: %v", err)
	}
	testOptions.ThemeDistDir = themeDistDir
	testOptions.TemplateDir = templateDir
	return testOptions
}

func TestRenderAllGeneratesStaticFiles(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "honepress",
		Description: "测试博客",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	blogService := NewBlogService(testOptions)
	if err := blogService.InitializeAndRender(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	postHTMLPath := filepath.Join(testOptions.PublicDir, "hello.html")
	postHTMLContent, err := os.ReadFile(postHTMLPath)
	if err != nil {
		t.Fatalf("read post HTML failed: %v", err)
	}
	if strings.Contains(string(postHTMLContent), "title:") {
		t.Fatalf("post HTML must not contain front matter")
	}
	if !strings.Contains(string(postHTMLContent), `data-font="default"`) {
		t.Fatalf("post HTML is missing default font marker")
	}
	if !strings.Contains(string(postHTMLContent), "欢迎使用 HonePress") {
		t.Fatalf("post HTML is missing body content")
	}

	requiredGeneratedFiles := []string{
		filepath.Join(testOptions.PublicDir, "index.html"),
		filepath.Join(testOptions.PublicDir, "archive.html"),
		filepath.Join(testOptions.PublicDir, "blog.html"),
		filepath.Join(testOptions.PublicDir, "rss.xml"),
		filepath.Join(testOptions.PublicDir, "sitemap.xml"),
		filepath.Join(testOptions.PublicDir, "style.css"),
		filepath.Join(testOptions.PublicDir, "theme.js"),
	}
	for _, requiredGeneratedFile := range requiredGeneratedFiles {
		if _, err := os.Stat(requiredGeneratedFile); err != nil {
			t.Fatalf("missing generated file %s: %v", requiredGeneratedFile, err)
		}
	}
}

func TestRenderAllSkipsDraftPosts(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "honepress",
		Description: "测试博客",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("create posts directory failed: %v", err)
	}

	postFiles := map[string]string{
		"published.md": `---
title: "已发布文章"
date: "2026-05-04 12:00:00"
description: "公开内容"
draft: false
url: "published.html"
aliases: []
---

这篇应该公开。
`,
		"draft.md": `---
title: "草稿文章"
date: "2026-05-04 13:00:00"
description: "草稿内容"
draft: true
url: "draft.html"
aliases: []
---

这篇不应该公开。
`,
		"draft-conflict.md": `---
title: "冲突草稿"
date: "2026-05-04 14:00:00"
description: "草稿链接可以暂时和公开文章重复"
draft: true
url: "published.html"
aliases: []
---

这篇也不应该公开。
`,
	}
	for fileName, fileContent := range postFiles {
		if err := os.WriteFile(filepath.Join(testOptions.PostsDir, fileName), []byte(fileContent), 0644); err != nil {
			t.Fatalf("write post failed: %v", err)
		}
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(testOptions.PublicDir, "published.html")); err != nil {
		t.Fatalf("published post was not generated: %v", err)
	}
	if _, err := os.Stat(filepath.Join(testOptions.PublicDir, "draft.html")); !os.IsNotExist(err) {
		t.Fatalf("draft must not generate a public page, got error: %v", err)
	}

	generatedFiles := []string{
		filepath.Join(testOptions.PublicDir, "index.html"),
		filepath.Join(testOptions.PublicDir, "archive.html"),
		filepath.Join(testOptions.PublicDir, "blog.html"),
		filepath.Join(testOptions.PublicDir, "rss.xml"),
		filepath.Join(testOptions.PublicDir, "sitemap.xml"),
	}
	for _, generatedFile := range generatedFiles {
		fileContent, err := os.ReadFile(generatedFile)
		if err != nil {
			t.Fatalf("read generated file %s failed: %v", generatedFile, err)
		}
		generatedContent := string(fileContent)
		if strings.Contains(generatedContent, "草稿文章") || strings.Contains(generatedContent, "冲突草稿") || strings.Contains(generatedContent, "draft.html") {
			t.Fatalf("generated file %s must not contain draft content", generatedFile)
		}
	}
}

func TestRenderAllWritesGiscusPlaceholder(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "honepress",
		Description: "test blog",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
		Comment: config.CommentOptions{
			Enabled:          true,
			GiscusRepo:       "owner/repo",
			GiscusRepoID:     "repo-id",
			GiscusCategory:   "Comments",
			GiscusCategoryID: "category-id",
		},
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("create posts directory failed: %v", err)
	}
	postContent := `---
title: "Giscus Post"
icon: "H"
date: "2026-05-04 12:00:00"
description: "comment test"
draft: false
url: "giscus.html"
aliases: []
---

Comment body.`
	if err := os.WriteFile(filepath.Join(testOptions.PostsDir, "giscus.md"), []byte(postContent), 0644); err != nil {
		t.Fatalf("write post failed: %v", err)
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	postHTMLContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "giscus.html"))
	if err != nil {
		t.Fatalf("read post HTML failed: %v", err)
	}
	postHTML := string(postHTMLContent)
	requiredFragments := []string{
		`<section id="comments" class="comments" data-giscus-comments`,
		`data-repo="owner/repo"`,
		`data-repo-id="repo-id"`,
		`data-category="Comments"`,
		`data-category-id="category-id"`,
	}
	for _, requiredFragment := range requiredFragments {
		if !strings.Contains(postHTML, requiredFragment) {
			t.Fatalf("post HTML is missing giscus fragment %q", requiredFragment)
		}
	}
	if strings.Contains(postHTML, "https://giscus.app/client.js") {
		t.Fatalf("post HTML must not render the giscus script directly")
	}
	if !strings.Contains(postHTML, `rel="icon" href="data:image/svg`) {
		t.Fatalf("post HTML should render the front matter icon as favicon")
	}
	if strings.Contains(postHTML, "post-icon") {
		t.Fatalf("post icon must not render inside the title")
	}

	themeScriptContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "theme.js"))
	if err != nil {
		t.Fatalf("read theme script failed: %v", err)
	}
	if !strings.Contains(string(themeScriptContent), "https://giscus.app/client.js") {
		t.Fatalf("theme script should load the giscus client")
	}
}

func TestRenderAllUsesPostSEOFields(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "HonePress",
		Description: "site description",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
		SiteIconURL: "/site-icon.png",
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("create posts directory failed: %v", err)
	}

	postContent := `---
title: "Visible Post Title"
date: "2026-05-04 12:00:00"
description: "Visible summary"
seoTitle: "Custom SEO Title"
seoDescription: "Custom SEO Description"
draft: false
url: "seo-post.html"
aliases: []
tags: []
---

Post body.`
	if err := os.WriteFile(filepath.Join(testOptions.PostsDir, "seo.md"), []byte(postContent), 0644); err != nil {
		t.Fatalf("write post failed: %v", err)
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	postHTMLContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "seo-post.html"))
	if err != nil {
		t.Fatalf("read post HTML failed: %v", err)
	}
	postHTML := string(postHTMLContent)
	requiredFragments := []string{
		`<title>Custom SEO Title</title>`,
		`<meta name="description" content="Custom SEO Description" />`,
		`<meta property="og:title" content="Custom SEO Title" />`,
		`<meta property="og:description" content="Custom SEO Description" />`,
		`<meta property="og:image" content="/site-icon.png" />`,
		`<meta name="twitter:card" content="summary_large_image" />`,
		`<meta name="twitter:image" content="/site-icon.png" />`,
		`"description":"Custom SEO Description"`,
		`"image":"/site-icon.png"`,
	}
	for _, requiredFragment := range requiredFragments {
		if !strings.Contains(postHTML, requiredFragment) {
			t.Fatalf("post HTML is missing SEO fragment %q", requiredFragment)
		}
	}
}
