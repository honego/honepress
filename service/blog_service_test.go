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
	themeScriptPath := filepath.Join(dataDirectoryPath, "theme.js")
	if err := os.WriteFile(themeScriptPath, []byte(""), 0644); err != nil {
		t.Fatalf("写入主题脚本失败：%v", err)
	}

	testOptions := option.Options{
		Address:             ":0",
		BaseURL:             "https://example.com",
		Title:               "blog",
		Description:         "测试博客",
		DataDir:             dataDirectoryPath,
		ContentDir:          filepath.Join(dataDirectoryPath, "content"),
		PostsDir:            filepath.Join(dataDirectoryPath, "content", "posts"),
		PublicDir:           filepath.Join(dataDirectoryPath, "public"),
		TemplateDir:         filepath.Join("..", "template"),
		AdminDistDir:        filepath.Join("..", "web", "admin", "dist"),
		ThemeDistPath:       themeScriptPath,
		TranslationCacheDir: filepath.Join(dataDirectoryPath, "content", "translations", "en"),
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
