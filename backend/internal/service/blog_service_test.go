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
	if err := os.MkdirAll(filepath.Join(themeDistDir, "_next", "static"), 0755); err != nil {
		t.Fatalf("create test next assets directory failed: %v", err)
	}
	staticFiles := map[string]string{
		"index.html":                    "<!doctype html><html><body>next-home</body></html>",
		"login.html":                    "<!doctype html><html><body>next-login</body></html>",
		"posts.html":                    `<!doctype html><html><head><title>Post - HonePress</title><meta name="description" content="generic post shell" /><meta property="og:title" content="generic" /></head><body><div id="__next">next-posts</div></body></html>`,
		"_next/static/app.test.js":      "console.log('next static')",
		"_next/static/app.test.css":     "body{margin:0}",
		"honepress-black.svg":           `<svg xmlns="http://www.w3.org/2000/svg"/>`,
		"fonts/DouyinSansBold.ttf":      "font",
		"admin-placeholder.ignore.html": "ignored by public app",
	}
	for fileName, fileContent := range staticFiles {
		filePath := filepath.Join(themeDistDir, filepath.FromSlash(fileName))
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			t.Fatalf("create test static parent failed: %v", err)
		}
		if err := os.WriteFile(filePath, []byte(fileContent), 0644); err != nil {
			t.Fatalf("write test static file failed: %v", err)
		}
	}

	testOptions.ThemeDistDir = themeDistDir
	return testOptions
}

func TestPostFaviconHrefPrefersPostIcon(t *testing.T) {
	testCases := []struct {
		name        string
		postIcon    string
		siteIconURL string
		want        string
		contains    string
	}{
		{
			name:        "post icon URL overrides site icon",
			postIcon:    "https://cdn.example.com/post-icon.svg",
			siteIconURL: "/site-icon.svg",
			want:        "https://cdn.example.com/post-icon.svg",
		},
		{
			name:        "post absolute icon path overrides site icon",
			postIcon:    "/post-icon.svg",
			siteIconURL: "/site-icon.svg",
			want:        "/post-icon.svg",
		},
		{
			name:        "post emoji renders SVG favicon",
			postIcon:    "☘️",
			siteIconURL: "/site-icon.svg",
			contains:    "data:image/svg+xml,",
		},
		{
			name:        "empty post icon falls back to site icon",
			postIcon:    "",
			siteIconURL: "/site-icon.svg",
			want:        "/site-icon.svg",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := string(postFaviconHref(testCase.postIcon, testCase.siteIconURL))
			if testCase.want != "" && got != testCase.want {
				t.Fatalf("favicon href mismatch: got %q, want %q", got, testCase.want)
			}
			if testCase.contains != "" && !strings.Contains(got, testCase.contains) {
				t.Fatalf("favicon href should contain %q, got %q", testCase.contains, got)
			}
		})
	}
}

func TestRenderAllCopiesNextStaticFilesAndMetadata(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "honepress",
		Description: "test blog",
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

	requiredGeneratedFiles := []string{
		filepath.Join(testOptions.PublicDir, "index.html"),
		filepath.Join(testOptions.PublicDir, "login.html"),
		filepath.Join(testOptions.PublicDir, "posts.html"),
		filepath.Join(testOptions.PublicDir, "_next", "static", "app.test.js"),
		filepath.Join(testOptions.PublicDir, "_next", "static", "app.test.css"),
		filepath.Join(testOptions.PublicDir, "rss.xml"),
		filepath.Join(testOptions.PublicDir, "sitemap.xml"),
	}
	for _, requiredGeneratedFile := range requiredGeneratedFiles {
		if _, err := os.Stat(requiredGeneratedFile); err != nil {
			t.Fatalf("missing generated file %s: %v", requiredGeneratedFile, err)
		}
	}

	indexContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "index.html"))
	if err != nil {
		t.Fatalf("read copied index failed: %v", err)
	}
	if !strings.Contains(string(indexContent), "next-home") {
		t.Fatalf("public index should come from Next static export")
	}
}

func TestRenderAllMetadataSkipsDraftPosts(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "honepress",
		Description: "test blog",
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
title: "Published Post"
icon: "spark"
date: "2026-05-04 12:00:00"
description: "public content"
seoTitle: "Published SEO Title"
seoDescription: "Published SEO Description"
draft: false
url: "published.html"
aliases:
  - old-published.html
tags: []
---

Published body.`,
		"draft.md": `---
title: "Draft Post"
date: "2026-05-04 13:00:00"
description: "draft content"
draft: true
url: "draft.html"
aliases: []
tags: []
---

Draft body.`,
	}
	for fileName, fileContent := range postFiles {
		if err := os.WriteFile(filepath.Join(testOptions.PostsDir, fileName), []byte(fileContent), 0644); err != nil {
			t.Fatalf("write post failed: %v", err)
		}
	}

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	publishedPostID := stablePostID("published.html")
	expectedPublishedPublicURL := "/?p=" + publishedPostID
	generatedFiles := []string{
		filepath.Join(testOptions.PublicDir, "rss.xml"),
		filepath.Join(testOptions.PublicDir, "sitemap.xml"),
	}
	for _, generatedFile := range generatedFiles {
		fileContent, err := os.ReadFile(generatedFile)
		if err != nil {
			t.Fatalf("read generated file %s failed: %v", generatedFile, err)
		}
		generatedContent := string(fileContent)
		if strings.Contains(generatedContent, "Draft Post") || strings.Contains(generatedContent, "draft.html") {
			t.Fatalf("generated file %s must not contain draft content", generatedFile)
		}
		if generatedFile == filepath.Join(testOptions.PublicDir, "sitemap.xml") && !strings.Contains(generatedContent, expectedPublishedPublicURL) {
			t.Fatalf("sitemap should contain published post permalink: %s", generatedContent)
		}
		if generatedFile != filepath.Join(testOptions.PublicDir, "sitemap.xml") && !strings.Contains(generatedContent, "Published Post") {
			t.Fatalf("generated file %s should contain published post metadata", generatedFile)
		}
		if generatedFile == filepath.Join(testOptions.PublicDir, "sitemap.xml") && strings.Contains(generatedContent, "old-published.html") {
			t.Fatalf("sitemap should not contain redirect aliases")
		}
	}

	staticPostContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "p", publishedPostID+".html"))
	if err != nil {
		t.Fatalf("read static post page failed: %v", err)
	}
	staticPostHTML := string(staticPostContent)
	expectedStaticPostSnippets := []string{
		"<title>Published SEO Title</title>",
		`<meta name="description" content="Published SEO Description" />`,
		`<link rel="canonical" href="` + expectedPublishedPublicURL + `" />`,
		`<meta property="og:type" content="article" />`,
		`"@type":"BlogPosting"`,
		"next-posts",
	}
	for _, expectedSnippet := range expectedStaticPostSnippets {
		if !strings.Contains(staticPostHTML, expectedSnippet) {
			t.Fatalf("static post page missing %q in %s", expectedSnippet, staticPostHTML)
		}
	}
	if strings.Contains(staticPostHTML, "generic post shell") || strings.Contains(staticPostHTML, "Post - HonePress") {
		t.Fatalf("static post page should replace generic Next post SEO: %s", staticPostHTML)
	}

	aliasContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "old-published.html"))
	if err != nil {
		t.Fatalf("read static alias redirect failed: %v", err)
	}
	if !strings.Contains(string(aliasContent), `url=`+expectedPublishedPublicURL) || !strings.Contains(string(aliasContent), `href="`+expectedPublishedPublicURL+`"`) {
		t.Fatalf("alias should redirect canonically to published post: %s", string(aliasContent))
	}
	if _, err := os.Stat(filepath.Join(testOptions.PublicDir, "draft.html")); !os.IsNotExist(err) {
		t.Fatalf("draft post page must not be generated")
	}

	blogRedirectContent, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "blog.html"))
	if err != nil {
		t.Fatalf("read blog redirect failed: %v", err)
	}
	if !strings.Contains(string(blogRedirectContent), `url=/archive.html`) {
		t.Fatalf("blog.html should redirect to archive.html")
	}
}

func TestPublicPostAPIsUsePublishedPostsOnly(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:       "honepress",
		Description: "test blog",
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
title: "Published Post"
icon: "☘️"
date: "2026-05-04 12:00:00"
description: "public content"
draft: false
url: "published.html"
aliases:
  - old-published.html
tags:
  - Go
---

**Published** body.`,
		"draft.md": `---
title: "Draft Post"
date: "2026-05-04 13:00:00"
description: "draft content"
draft: true
url: "draft.html"
aliases: []
tags: []
---

Draft body.`,
	}
	for fileName, fileContent := range postFiles {
		if err := os.WriteFile(filepath.Join(testOptions.PostsDir, fileName), []byte(fileContent), 0644); err != nil {
			t.Fatalf("write post failed: %v", err)
		}
	}

	blogService := NewBlogService(testOptions)
	publicPosts, err := blogService.ListPublicPosts()
	if err != nil {
		t.Fatalf("list public posts failed: %v", err)
	}
	if len(publicPosts) != 1 || publicPosts[0].ID != "published.md" {
		t.Fatalf("public posts mismatch: %#v", publicPosts)
	}

	publicPost, err := blogService.GetPublicPost("published.html")
	if err != nil {
		t.Fatalf("get public post failed: %v", err)
	}
	if _, err := blogService.GetPublicPost("old-published.html"); err != nil {
		t.Fatalf("get public post by alias failed: %v", err)
	}
	if !strings.Contains(publicPost.HTML, "<strong>Published</strong>") {
		t.Fatalf("public post HTML did not render markdown: %s", publicPost.HTML)
	}
	if publicPost.Icon != "☘️" {
		t.Fatalf("public post should expose article icon: %q", publicPost.Icon)
	}
	if strings.Contains(publicPost.HTML, "title:") {
		t.Fatalf("public post HTML must not contain front matter")
	}

	if _, err := blogService.GetPublicPost("draft.html"); err == nil {
		t.Fatalf("draft post must not be available from public API")
	}
}

func TestRenderAllUsesGlobalPermalinkStructure(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:              "honepress",
		Description:        "test blog",
		DataDir:            dataDirectoryPath,
		ContentDir:         filepath.Join(dataDirectoryPath, "content"),
		PostsDir:           filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:          filepath.Join(dataDirectoryPath, "public"),
		PermalinkStructure: "/%year%/%monthnum%/%postname%/",
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("create posts directory failed: %v", err)
	}
	postContent := `---
title: "Permalink Post"
date: "2026-05-04 12:00:00"
description: "permalink content"
draft: false
url: "permalink-post.html"
aliases: []
tags: []
---

Permalink body.`
	if err := os.WriteFile(filepath.Join(testOptions.PostsDir, "permalink.md"), []byte(postContent), 0644); err != nil {
		t.Fatalf("write post failed: %v", err)
	}

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	expectedPostPath := filepath.Join(testOptions.PublicDir, "2026", "05", "permalink-post", "index.html")
	if _, err := os.Stat(expectedPostPath); err != nil {
		t.Fatalf("global permalink page was not generated: %v", err)
	}
	postContentBytes, err := os.ReadFile(expectedPostPath)
	if err != nil {
		t.Fatalf("read generated permalink post failed: %v", err)
	}
	if !strings.Contains(string(postContentBytes), `href="/2026/05/permalink-post/"`) {
		t.Fatalf("post canonical should use global permalink: %s", string(postContentBytes))
	}

	oldPermalinkRedirect, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "permalink-post.html"))
	if err != nil {
		t.Fatalf("read old permalink redirect failed: %v", err)
	}
	if !strings.Contains(string(oldPermalinkRedirect), `url=/2026/05/permalink-post/`) {
		t.Fatalf("old permalink should redirect to global permalink: %s", string(oldPermalinkRedirect))
	}

	publicPosts, err := blogService.ListPublicPosts()
	if err != nil {
		t.Fatalf("list public posts failed: %v", err)
	}
	if len(publicPosts) != 1 || publicPosts[0].PublicURL != "/2026/05/permalink-post/" {
		t.Fatalf("public post URL should use global permalink: %#v", publicPosts)
	}
	if _, err := blogService.GetPublicPost("2026/05/permalink-post"); err != nil {
		t.Fatalf("get public post by global permalink failed: %v", err)
	}
}

func TestRenderAllUsesPlainPermalinkPostID(t *testing.T) {
	dataDirectoryPath := t.TempDir()

	testOptions := config.Options{
		Title:              "honepress",
		Description:        "test blog",
		DataDir:            dataDirectoryPath,
		ContentDir:         filepath.Join(dataDirectoryPath, "content"),
		PostsDir:           filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:          filepath.Join(dataDirectoryPath, "public"),
		PermalinkStructure: "/?p=%post_id%",
	}
	testOptions = withTestRuntimeFiles(t, dataDirectoryPath, testOptions)

	if err := os.MkdirAll(testOptions.PostsDir, 0755); err != nil {
		t.Fatalf("create posts directory failed: %v", err)
	}
	postContent := `---
title: "Plain Permalink Post"
date: "2026-05-04 12:00:00"
description: "plain permalink content"
draft: false
url: "plain-post.html"
aliases: []
tags: []
---

Plain permalink body.`
	if err := os.WriteFile(filepath.Join(testOptions.PostsDir, "plain.md"), []byte(postContent), 0644); err != nil {
		t.Fatalf("write post failed: %v", err)
	}

	blogService := NewBlogService(testOptions)
	if err := blogService.RenderAll(); err != nil {
		t.Fatalf("render failed: %v", err)
	}

	postID := stablePostID("plain-post.html")
	expectedPublicURL := "/?p=" + postID
	expectedPostPath := filepath.Join(testOptions.PublicDir, "p", postID+".html")
	if _, err := os.Stat(expectedPostPath); err != nil {
		t.Fatalf("plain permalink page was not generated by post id: %v", err)
	}
	postContentBytes, err := os.ReadFile(expectedPostPath)
	if err != nil {
		t.Fatalf("read generated plain permalink post failed: %v", err)
	}
	if !strings.Contains(string(postContentBytes), `href="`+expectedPublicURL+`"`) {
		t.Fatalf("post canonical should use plain permalink: %s", string(postContentBytes))
	}

	oldPermalinkRedirect, err := os.ReadFile(filepath.Join(testOptions.PublicDir, "plain-post.html"))
	if err != nil {
		t.Fatalf("read old permalink redirect failed: %v", err)
	}
	if !strings.Contains(string(oldPermalinkRedirect), `url=`+expectedPublicURL) {
		t.Fatalf("old permalink should redirect to plain permalink: %s", string(oldPermalinkRedirect))
	}

	publicPosts, err := blogService.ListPublicPosts()
	if err != nil {
		t.Fatalf("list public posts failed: %v", err)
	}
	if len(publicPosts) != 1 || publicPosts[0].PublicURL != expectedPublicURL {
		t.Fatalf("public post URL should use plain permalink: %#v", publicPosts)
	}
	if _, err := blogService.GetPublicPost(postID); err != nil {
		t.Fatalf("get public post by numeric post id failed: %v", err)
	}
}
