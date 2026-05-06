package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/honeok/honepress/internal/option"
)

func withTestRuntimeFiles(t *testing.T, dataDirectoryPath string, testOptions option.Options) option.Options {
	t.Helper()

	themeDistDir := filepath.Join(dataDirectoryPath, "theme-dist")
	if err := os.MkdirAll(themeDistDir, 0755); err != nil {
		t.Fatalf("创建测试主题构建目录失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDistDir, "theme.js"), []byte("https://giscus.app/client.js"), 0644); err != nil {
		t.Fatalf("写入测试主题脚本失败：%v", err)
	}
	if err := os.WriteFile(filepath.Join(themeDistDir, "favicon.svg"), []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0644); err != nil {
		t.Fatalf("写入测试 favicon 失败：%v", err)
	}

	templateDir, err := filepath.Abs(filepath.Join("..", "..", "templates"))
	if err != nil {
		t.Fatalf("解析测试模板目录失败：%v", err)
	}
	testOptions.ThemeDistDir = themeDistDir
	testOptions.TemplateDir = templateDir
	return testOptions
}

func TestRenderAllGeneratesStaticFiles(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := option.Options{
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
		t.Fatalf("渲染失败：%v", err)
	}

	postHTMLPath := filepath.Join(testOptions.PublicDir, "hello.html")
	postHTMLContent, err := os.ReadFile(postHTMLPath)
	if err != nil {
		t.Fatalf("读取文章 HTML 失败：%v", err)
	}
	if strings.Contains(string(postHTMLContent), "title:") {
		t.Fatalf("文章 HTML 不应包含文章元信息")
	}
	if !strings.Contains(string(postHTMLContent), `data-font="default"`) {
		t.Fatalf("文章 HTML 缺少默认字体标记")
	}
	if !strings.Contains(string(postHTMLContent), "欢迎使用 HonePress") {
		t.Fatalf("文章 HTML 缺少正文内容")
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
			t.Fatalf("缺少生成文件 %s：%v", requiredGeneratedFile, err)
		}
	}
}

func TestRenderAllSkipsDraftPosts(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := option.Options{
		Title:       "honepress",
		Description: "测试博客",
		DataDir:     dataDirectoryPath,
		ContentDir:  filepath.Join(dataDirectoryPath, "content"),
		PostsDir:    filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:   filepath.Join(dataDirectoryPath, "public"),
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

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
			t.Fatalf("写入文章失败：%v", err)
		}
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

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
		filepath.Join(testOptions.PublicDir, "archive.html"),
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
		Title:       "honepress",
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
		},
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("创建文章目录失败：%v", err)
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
		t.Fatalf("写入文章失败：%v", err)
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("渲染失败：%v", err)
	}

	postHTMLContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "giscus.html"))
	if err != nil {
		t.Fatalf("读取文章 HTML 失败：%v", err)
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
			t.Fatalf("文章 HTML 缺少 giscus 片段 %q", requiredFragment)
		}
	}
	if strings.Contains(postHTML, "https://giscus.app/client.js") {
		t.Fatalf("文章 HTML 不应直接渲染 giscus 脚本")
	}
	if !strings.Contains(postHTML, `rel="icon" href="data:image/svg`) {
		t.Fatalf("文章 HTML 应将文章元信息 icon 渲染为 favicon")
	}
	if strings.Contains(postHTML, "post-icon") {
		t.Fatalf("文章 icon 不应渲染到标题中")
	}

	themeScriptContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "theme.js"))
	if err != nil {
		t.Fatalf("读取主题脚本失败：%v", err)
	}
	if !strings.Contains(string(themeScriptContent), "https://giscus.app/client.js") {
		t.Fatalf("主题脚本应加载 giscus 客户端")
	}
}

func TestRenderAllUsesPostSEOFields(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := option.Options{
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
		t.Fatalf("创建文章目录失败：%v", err)
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
		t.Fatalf("写入文章失败：%v", err)
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("渲染失败：%v", err)
	}

	postHTMLContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "seo-post.html"))
	if err != nil {
		t.Fatalf("读取文章 HTML 失败：%v", err)
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
			t.Fatalf("文章 HTML 缺少 SEO 片段 %q", requiredFragment)
		}
	}
}
