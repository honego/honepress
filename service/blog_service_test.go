package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honeok/blog/option"
)

func TestRenderAllGeneratesStaticFiles(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := option.Options{
		BaseURL:     "https://example.com",
		Title:       "blog",
		Description: "测试博客",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
	}

	blogService := NewBlogService(testOptions)
	if err := blogService.InitializeAndRender(); err != nil {
		t.Fatalf("渲染失败：%v", err)
	}

	postHTMLPath := filepath.Join(testOptions.PublicDir, "1.html")
	postHTMLContent, err := os.ReadFile(postHTMLPath)
	if err != nil {
		t.Fatalf("读取文章 HTML 失败：%v", err)
	}
	if strings.Contains(string(postHTMLContent), "title:") {
		t.Fatalf("文章 HTML 不应包含 Front Matter")
	}
	if !strings.Contains(string(postHTMLContent), `data-font="default"`) {
		t.Fatalf("文章 HTML 缺少默认字体标记")
	}
	if !strings.Contains(string(postHTMLContent), "第一篇示例文章") {
		t.Fatalf("文章 HTML 缺少正文内容")
	}

	requiredGeneratedFiles := []string{
		filepath.Join(testOptions.PublicDir, "index.html"),
		filepath.Join(testOptions.PublicDir, "blog.html"),
		filepath.Join(testOptions.PublicDir, "rss.xml"),
		filepath.Join(testOptions.PublicDir, "sitemap.xml"),
		filepath.Join(testOptions.PublicDir, "style.css"),
		filepath.Join(testOptions.PublicDir, "theme.js"),
	}
	for _, requiredGeneratedFile := range requiredGeneratedFiles {
		if _, err := os.Stat(requiredGeneratedFile); err != nil {
			t.Fatalf("缺少生成文件 %s：%v", requiredGeneratedFile, err)
		}
	}
}

func TestRenderAllSkipsDraftPosts(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := option.Options{
		BaseURL:     "https://example.com",
		Title:       "blog",
		Description: "测试博客",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
	}

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("创建文章目录失败：%v", err)
	}

	postFiles := map[string]string{
		"published.md": `---
title: "已发布文章"
date: "2026-05-04 12:00:00"
description: "公开内容"
draft: false
url: "published.html"
comments: true
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
comments: true
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
comments: true
aliases: []
---

这篇也不应该公开。
`,
	}
	for fileName, fileContent := range postFiles {
		if err := os.WriteFile(filepath.Join(testOptions.PostsDir, fileName), []byte(fileContent), 0644); err != nil {
			t.Fatalf("写入文章失败：%v", err)
		}
	}

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("渲染失败：%v", err)
	}

	if _, err := os.Stat(filepath.Join(testOptions.PublicDir, "published.html")); err != nil {
		t.Fatalf("已发布文章没有生成：%v", err)
	}
	if _, err := os.Stat(filepath.Join(testOptions.PublicDir, "draft.html")); !os.IsNotExist(err) {
		t.Fatalf("草稿不应生成公开页面，实际错误：%v", err)
	}

	generatedFiles := []string{
		filepath.Join(testOptions.PublicDir, "index.html"),
		filepath.Join(testOptions.PublicDir, "blog.html"),
		filepath.Join(testOptions.PublicDir, "rss.xml"),
		filepath.Join(testOptions.PublicDir, "sitemap.xml"),
	}
	for _, generatedFile := range generatedFiles {
		fileContent, err := os.ReadFile(generatedFile)
		if err != nil {
			t.Fatalf("读取生成文件失败 %s：%v", generatedFile, err)
		}
		generatedContent := string(fileContent)
		if strings.Contains(generatedContent, "草稿文章") || strings.Contains(generatedContent, "冲突草稿") || strings.Contains(generatedContent, "draft.html") {
			t.Fatalf("生成文件 %s 不应包含草稿内容", generatedFile)
		}
	}
}

func TestRenderAllWritesGiscusPlaceholder(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := option.Options{
		BaseURL:     "https://example.com",
		Title:       "blog",
		Description: "test blog",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
		Comment: option.CommentOptions{
			Enabled:          true,
			GiscusRepo:       "owner/repo",
			GiscusRepoID:     "repo-id",
			GiscusCategory:   "Comments",
			GiscusCategoryID: "category-id",
			GiscusMapping:    "pathname",
			GiscusStrict:     "0",
			ReactionsEnabled: "0",
			EmitMetadata:     "1",
			InputPosition:    "top",
			Theme:            "noborder_light",
			Language:         "en",
		},
	}

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("create posts directory failed: %v", err)
	}
	postContent := `---
title: "Giscus Post"
date: "2026-05-04 12:00:00"
description: "comment test"
draft: false
url: "giscus.html"
comments: true
aliases: []
---

Comment body.`
	if err := os.WriteFile(filepath.Join(testOptions.PostsDir, "giscus.md"), []byte(postContent), 0644); err != nil {
		t.Fatalf("write post failed: %v", err)
	}

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	postHTMLContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "giscus.html"))
	if err != nil {
		t.Fatalf("read post html failed: %v", err)
	}
	postHTML := string(postHTMLContent)
	requiredFragments := []string{
		`<section id="comments" class="comments" data-giscus-comments`,
		`data-repo="owner/repo"`,
		`data-repo-id="repo-id"`,
		`data-category="Comments"`,
		`data-category-id="category-id"`,
		`data-mapping="pathname"`,
		`data-reactions-enabled="0"`,
		`data-emit-metadata="1"`,
		`data-input-position="top"`,
		`data-theme="noborder_light"`,
		`data-lang="en"`,
	}
	for _, requiredFragment := range requiredFragments {
		if !strings.Contains(postHTML, requiredFragment) {
			t.Fatalf("post html missing giscus fragment %q", requiredFragment)
		}
	}
	if strings.Contains(postHTML, "https://giscus.app/client.js") {
		t.Fatalf("post html should not render the giscus script directly")
	}

	themeScriptContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "theme.js"))
	if err != nil {
		t.Fatalf("read theme script failed: %v", err)
	}
	if !strings.Contains(string(themeScriptContent), "https://giscus.app/client.js") {
		t.Fatalf("theme script should load the giscus client")
	}
}
