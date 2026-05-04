package service

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	htmlTemplate "html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/honeok/blog/common/filesystem"
	"github.com/honeok/blog/common/validation"
	"github.com/honeok/blog/model"
	"github.com/honeok/blog/option"
	"github.com/honeok/blog/renderer"
	"github.com/honeok/blog/web"
)

// BlogService 串联文章扫描、Markdown 渲染、静态文件生成和后台写入操作。
type BlogService struct {
	options           option.Options
	markdownRenderer  *renderer.MarkdownRenderer
	translationClient *TranslationClient
	renderMutex       sync.Mutex
}

// NewBlogService 创建博客服务，翻译客户端即使配置不完整也允许启动。
func NewBlogService(options option.Options) *BlogService {
	return &BlogService{
		options:           options,
		markdownRenderer:  renderer.NewMarkdownRenderer(),
		translationClient: NewTranslationClient(options.Translation),
	}
}

// InitializeAndRender 准备数据目录和示例文章，然后生成完整静态站点。
func (blogService *BlogService) InitializeAndRender() error {
	if err := blogService.ensureDataDirectories(); err != nil {
		return err
	}
	if err := blogService.createExamplePostsIfEmpty(); err != nil {
		return err
	}
	return blogService.RenderAll()
}

// RenderAll 重新扫描全部文章并生成 public 目录里的静态产物。
func (blogService *BlogService) RenderAll() error {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	return blogService.renderAllWithoutLock()
}

// PreviewMarkdown 使用与正式渲染相同的 goldmark 配置，避免后台预览和前台页面不一致。
func (blogService *BlogService) PreviewMarkdown(markdownContent string) (string, error) {
	renderedHTML, err := blogService.markdownRenderer.Render(markdownContent)
	if err != nil {
		return "", err
	}
	return string(renderedHTML), nil
}

func (blogService *BlogService) ensureDataDirectories() error {
	requiredDirectoryPaths := []string{
		blogService.options.PostsDir,
		blogService.options.TranslationCacheDir,
		blogService.options.PublicDir,
	}
	for _, requiredDirectoryPath := range requiredDirectoryPaths {
		if err := filesystem.EnsureDirectory(requiredDirectoryPath); err != nil {
			return err
		}
	}
	return nil
}

func (blogService *BlogService) createExamplePostsIfEmpty() error {
	directoryEntries, err := os.ReadDir(blogService.options.PostsDir)
	if err != nil {
		return fmt.Errorf("读取文章目录失败：%w", err)
	}

	hasMarkdownPost := false
	for _, directoryEntry := range directoryEntries {
		if !directoryEntry.IsDir() && strings.EqualFold(filepath.Ext(directoryEntry.Name()), ".md") {
			hasMarkdownPost = true
			break
		}
	}
	if hasMarkdownPost {
		return nil
	}

	examplePosts := map[string]string{
		"1.md": examplePostOne,
		"2.md": examplePostTwo,
	}
	for exampleFileName, exampleMarkdownContent := range examplePosts {
		exampleFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, exampleFileName)
		if err != nil {
			return err
		}
		if err := filesystem.WriteFileCreatingDirectory(exampleFilePath, []byte(exampleMarkdownContent), 0644); err != nil {
			return err
		}
	}

	log.Println("未发现文章，已生成示例文章 1.md 和 2.md。")
	return nil
}

func (blogService *BlogService) renderAllWithoutLock() error {
	if err := blogService.ensureDataDirectories(); err != nil {
		return err
	}

	chinesePosts, err := blogService.scanChinesePosts()
	if err != nil {
		return err
	}
	if err := validatePermalinkConflicts(chinesePosts); err != nil {
		return err
	}

	publishedChinesePosts := filterPublishedPosts(chinesePosts)
	sortPostsByDateDescending(publishedChinesePosts)

	englishPosts := blogService.prepareEnglishPosts(publishedChinesePosts)
	if err := validatePermalinkConflicts(englishPosts); err != nil {
		return err
	}
	sortPostsByDateDescending(englishPosts)

	if err := blogService.resetPublicDirectory(); err != nil {
		return err
	}

	templateRenderer, err := renderer.NewTemplateRenderer(blogService.options)
	if err != nil {
		return err
	}

	if err := templateRenderer.CopyStyle(); err != nil {
		return err
	}
	blogService.copyThemeScript()

	if err := blogService.renderChineseSite(templateRenderer, publishedChinesePosts, englishPosts); err != nil {
		return err
	}
	if blogService.options.Translation.Enabled {
		if err := blogService.renderEnglishSite(templateRenderer, englishPosts); err != nil {
			return err
		}
	}

	log.Println("静态文件已重新生成。")
	return nil
}

func (blogService *BlogService) scanChinesePosts() ([]model.Post, error) {
	directoryEntries, err := os.ReadDir(blogService.options.PostsDir)
	if err != nil {
		return nil, fmt.Errorf("读取文章目录失败：%w", err)
	}

	chinesePosts := make([]model.Post, 0, len(directoryEntries))
	for _, directoryEntry := range directoryEntries {
		if directoryEntry.IsDir() || !strings.EqualFold(filepath.Ext(directoryEntry.Name()), ".md") {
			continue
		}
		if err := validation.ValidateMarkdownFileName(directoryEntry.Name()); err != nil {
			return nil, err
		}

		sourceFilePath := filepath.Join(blogService.options.PostsDir, directoryEntry.Name())
		sourceMarkdownContent, err := os.ReadFile(sourceFilePath)
		if err != nil {
			return nil, fmt.Errorf("读取文章失败：%s：%w", sourceFilePath, err)
		}

		parsedFrontMatter, bodyMarkdownContent, err := renderer.ParsePostDocument(directoryEntry.Name(), sourceMarkdownContent)
		if err != nil {
			return nil, err
		}
		if err := validation.ValidateRequiredPostFields(parsedFrontMatter.Title, parsedFrontMatter.Date); err != nil {
			return nil, fmt.Errorf("文章 %s 校验失败：%w", sourceFilePath, err)
		}

		normalizedPermalink, err := validation.NormalizePermalinkWithFallback(parsedFrontMatter.URL, directoryEntry.Name())
		if err != nil {
			return nil, fmt.Errorf("文章 %s 的固定链接无效：%w", sourceFilePath, err)
		}

		normalizedAliases := make([]string, 0, len(parsedFrontMatter.Aliases))
		for _, rawAlias := range parsedFrontMatter.Aliases {
			normalizedAlias, err := validation.NormalizePermalink(rawAlias)
			if err != nil {
				return nil, fmt.Errorf("文章 %s 的别名无效：%w", sourceFilePath, err)
			}
			normalizedAliases = append(normalizedAliases, normalizedAlias)
		}

		publishedAt, err := validation.ParsePostDate(parsedFrontMatter.Date)
		if err != nil {
			return nil, fmt.Errorf("文章 %s 的发布时间无效：%w", sourceFilePath, err)
		}

		renderedPostHTML, err := blogService.markdownRenderer.Render(bodyMarkdownContent)
		if err != nil {
			return nil, fmt.Errorf("文章 %s 渲染失败：%w", sourceFilePath, err)
		}

		sourceContentHash := sha256.Sum256(sourceMarkdownContent)
		chinesePost := model.Post{
			SourceFileName:    directoryEntry.Name(),
			SourceFilePath:    sourceFilePath,
			Title:             parsedFrontMatter.Title,
			DateText:          parsedFrontMatter.Date,
			PublishedAt:       publishedAt,
			Description:       parsedFrontMatter.Description,
			Draft:             parsedFrontMatter.Draft,
			URL:               normalizedPermalink,
			Aliases:           normalizedAliases,
			Comments:          parsedFrontMatter.Comments,
			Translation:       parsedFrontMatter.Translation,
			BodyMarkdown:      bodyMarkdownContent,
			BodyHTML:          renderedPostHTML,
			Language:          "zh-CN",
			SourceHash:        hex.EncodeToString(sourceContentHash[:]),
			TranslationStatus: blogService.detectTranslationStatus(directoryEntry.Name(), parsedFrontMatter, parsedFrontMatter.Draft, hex.EncodeToString(sourceContentHash[:])),
		}
		chinesePosts = append(chinesePosts, chinesePost)
	}

	sortPostsByDateDescending(chinesePosts)
	return chinesePosts, nil
}

func (blogService *BlogService) renderChineseSite(templateRenderer *renderer.TemplateRenderer, chinesePosts []model.Post, englishPosts []model.Post) error {
	englishURLBySourceFileName := make(map[string]string, len(englishPosts))
	for _, englishPost := range englishPosts {
		englishURLBySourceFileName[englishPost.SourceFileName] = "/en/" + englishPost.URL
	}

	chineseSummaries := postsToSummaries(chinesePosts, "", englishURLBySourceFileName)
	chineseLabels := chineseTemplateLabels(blogService.options.ThemeDefault)
	chineseSiteViewData := model.SiteViewData{
		SiteTitle:       blogService.options.Title,
		SiteDescription: blogService.options.Description,
		BaseURL:         blogService.options.BaseURL,
		GitHubURL:       blogService.options.GitHubURL,
		TelegramURL:     blogService.options.TelegramURL,
		ThemeDefault:    blogService.options.ThemeDefault,
		Language:        blogService.options.Language,
		CanonicalPath:   "/",
		HomePath:        "/",
		BlogPath:        "/blog.html",
		RSSPath:         "/rss.xml",
		AlternatePath:   "/en/",
		Labels:          chineseLabels,
		Posts:           chineseSummaries,
	}

	if err := templateRenderer.RenderIndex(filepath.Join(blogService.options.PublicDir, "index.html"), chineseSiteViewData); err != nil {
		return err
	}
	chineseSiteViewData.CanonicalPath = "/blog.html"
	if err := templateRenderer.RenderBlog(filepath.Join(blogService.options.PublicDir, "blog.html"), chineseSiteViewData); err != nil {
		return err
	}

	for _, currentPost := range chinesePosts {
		postViewData := model.PostViewData{
			SiteTitle:       blogService.options.Title,
			SiteDescription: blogService.options.Description,
			BaseURL:         blogService.options.BaseURL,
			GitHubURL:       blogService.options.GitHubURL,
			TelegramURL:     blogService.options.TelegramURL,
			ThemeDefault:    blogService.options.ThemeDefault,
			Language:        blogService.options.Language,
			CanonicalPath:   "/" + currentPost.URL,
			HomePath:        "/",
			BlogPath:        "/blog.html",
			RSSPath:         "/rss.xml",
			AlternatePath:   englishURLBySourceFileName[currentPost.SourceFileName],
			Labels:          chineseLabels,
			Post:            currentPost,
			CommentHTML:     blogService.commentHTMLForPost(currentPost),
		}
		if err := templateRenderer.RenderPost(filepath.Join(blogService.options.PublicDir, currentPost.URL), postViewData); err != nil {
			return err
		}
		for _, normalizedAlias := range currentPost.Aliases {
			if err := templateRenderer.RenderRedirect(filepath.Join(blogService.options.PublicDir, normalizedAlias), "/"+currentPost.URL); err != nil {
				return err
			}
		}
	}

	if err := templateRenderer.RenderRSS(filepath.Join(blogService.options.PublicDir, "rss.xml"), blogService.options.Title, blogService.options.Description, "/", chinesePosts, ""); err != nil {
		return err
	}

	chineseSitemapPaths := []string{"/", "/blog.html"}
	for _, currentPost := range chinesePosts {
		chineseSitemapPaths = append(chineseSitemapPaths, "/"+currentPost.URL)
	}
	return templateRenderer.RenderSitemap(filepath.Join(blogService.options.PublicDir, "sitemap.xml"), chineseSitemapPaths)
}

func (blogService *BlogService) renderEnglishSite(templateRenderer *renderer.TemplateRenderer, englishPosts []model.Post) error {
	englishDirectoryPath := filepath.Join(blogService.options.PublicDir, "en")
	if err := filesystem.EnsureDirectory(englishDirectoryPath); err != nil {
		return err
	}

	englishSummaries := postsToSummaries(englishPosts, "/en", map[string]string{})
	englishLabels := englishTemplateLabels(blogService.options.ThemeDefault)
	englishSiteViewData := model.SiteViewData{
		SiteTitle:       blogService.options.Title,
		SiteDescription: blogService.options.Description,
		BaseURL:         blogService.options.BaseURL,
		GitHubURL:       blogService.options.GitHubURL,
		TelegramURL:     blogService.options.TelegramURL,
		ThemeDefault:    blogService.options.ThemeDefault,
		Language:        "en-US",
		CanonicalPath:   "/en/",
		HomePath:        "/en/",
		BlogPath:        "/en/blog.html",
		RSSPath:         "/en/rss.xml",
		AlternatePath:   "/",
		Labels:          englishLabels,
		Posts:           englishSummaries,
	}

	if err := templateRenderer.RenderIndex(filepath.Join(englishDirectoryPath, "index.html"), englishSiteViewData); err != nil {
		return err
	}
	englishSiteViewData.CanonicalPath = "/en/blog.html"
	if err := templateRenderer.RenderBlog(filepath.Join(englishDirectoryPath, "blog.html"), englishSiteViewData); err != nil {
		return err
	}

	for _, currentPost := range englishPosts {
		postViewData := model.PostViewData{
			SiteTitle:       blogService.options.Title,
			SiteDescription: blogService.options.Description,
			BaseURL:         blogService.options.BaseURL,
			GitHubURL:       blogService.options.GitHubURL,
			TelegramURL:     blogService.options.TelegramURL,
			ThemeDefault:    blogService.options.ThemeDefault,
			Language:        "en-US",
			CanonicalPath:   "/en/" + currentPost.URL,
			HomePath:        "/en/",
			BlogPath:        "/en/blog.html",
			RSSPath:         "/en/rss.xml",
			AlternatePath:   "/" + currentPost.URL,
			Labels:          englishLabels,
			Post:            currentPost,
		}
		if err := templateRenderer.RenderPost(filepath.Join(englishDirectoryPath, currentPost.URL), postViewData); err != nil {
			return err
		}
	}

	if err := templateRenderer.RenderRSS(filepath.Join(englishDirectoryPath, "rss.xml"), blogService.options.Title+" - English", blogService.options.Description, "/en/", englishPosts, "/en"); err != nil {
		return err
	}

	englishSitemapPaths := []string{"/en/", "/en/blog.html"}
	for _, currentPost := range englishPosts {
		englishSitemapPaths = append(englishSitemapPaths, "/en/"+currentPost.URL)
	}
	return templateRenderer.RenderSitemap(filepath.Join(englishDirectoryPath, "sitemap.xml"), englishSitemapPaths)
}

func (blogService *BlogService) resetPublicDirectory() error {
	absoluteDataDirectoryPath, err := filepath.Abs(blogService.options.DataDir)
	if err != nil {
		return fmt.Errorf("解析 data 目录失败：%w", err)
	}
	absolutePublicDirectoryPath, err := filepath.Abs(blogService.options.PublicDir)
	if err != nil {
		return fmt.Errorf("解析 public 目录失败：%w", err)
	}
	relativePublicDirectoryPath, err := filepath.Rel(absoluteDataDirectoryPath, absolutePublicDirectoryPath)
	if err != nil {
		return fmt.Errorf("校验 public 目录失败：%w", err)
	}
	if relativePublicDirectoryPath == "." || strings.HasPrefix(relativePublicDirectoryPath, "..") {
		return fmt.Errorf("public 目录必须位于 data 目录内部：%s", blogService.options.PublicDir)
	}

	if err := os.RemoveAll(absolutePublicDirectoryPath); err != nil {
		return fmt.Errorf("清理 public 目录失败：%w", err)
	}
	return filesystem.EnsureDirectory(absolutePublicDirectoryPath)
}

func (blogService *BlogService) copyThemeScript() {
	targetThemeScriptPath := filepath.Join(blogService.options.PublicDir, "theme.js")
	themeScriptContent, err := web.ThemeScript()
	if err != nil {
		log.Printf("警告：读取内置主题脚本失败，请先构建 web/theme：%v", err)
		return
	}
	if err := filesystem.WriteFileCreatingDirectory(targetThemeScriptPath, themeScriptContent, 0644); err != nil {
		log.Printf("警告：写入前台主题脚本失败：%v", err)
	}
}

func (blogService *BlogService) commentHTMLForPost(post model.Post) htmlTemplate.HTML {
	if !blogService.options.Comment.Enabled || !post.Comments {
		return ""
	}
	if blogService.options.Comment.Provider != "giscus" {
		log.Printf("警告：不支持的评论服务：%s", blogService.options.Comment.Provider)
		return ""
	}
	if !blogService.options.Comment.HasRequiredGiscusConfig() {
		log.Println("警告：giscus 配置不完整，已跳过评论脚本。")
		return ""
	}

	escapedRepo := htmlTemplate.HTMLEscapeString(blogService.options.Comment.GiscusRepo)
	escapedRepoID := htmlTemplate.HTMLEscapeString(blogService.options.Comment.GiscusRepoID)
	escapedCategory := htmlTemplate.HTMLEscapeString(blogService.options.Comment.GiscusCategory)
	escapedCategoryID := htmlTemplate.HTMLEscapeString(blogService.options.Comment.GiscusCategoryID)
	escapedMapping := htmlTemplate.HTMLEscapeString(blogService.options.Comment.GiscusMapping)
	escapedStrict := htmlTemplate.HTMLEscapeString(blogService.options.Comment.GiscusStrict)
	escapedReactions := htmlTemplate.HTMLEscapeString(blogService.options.Comment.ReactionsEnabled)
	escapedEmitMetadata := htmlTemplate.HTMLEscapeString(blogService.options.Comment.EmitMetadata)
	escapedInputPosition := htmlTemplate.HTMLEscapeString(blogService.options.Comment.InputPosition)
	escapedTheme := htmlTemplate.HTMLEscapeString(blogService.options.Comment.Theme)
	escapedLanguage := htmlTemplate.HTMLEscapeString(blogService.options.Comment.Language)

	commentScriptHTML := `<section class="comments"><script src="https://giscus.app/client.js" data-repo="` + escapedRepo + `" data-repo-id="` + escapedRepoID + `" data-category="` + escapedCategory + `" data-category-id="` + escapedCategoryID + `" data-mapping="` + escapedMapping + `" data-strict="` + escapedStrict + `" data-reactions-enabled="` + escapedReactions + `" data-emit-metadata="` + escapedEmitMetadata + `" data-input-position="` + escapedInputPosition + `" data-theme="` + escapedTheme + `" data-lang="` + escapedLanguage + `" crossorigin="anonymous" async></script></section>`
	return htmlTemplate.HTML(commentScriptHTML)
}

func filterPublishedPosts(posts []model.Post) []model.Post {
	publishedPosts := make([]model.Post, 0, len(posts))
	for _, currentPost := range posts {
		if !currentPost.Draft {
			publishedPosts = append(publishedPosts, currentPost)
		}
	}
	return publishedPosts
}

func sortPostsByDateDescending(posts []model.Post) {
	sort.SliceStable(posts, func(firstIndex int, secondIndex int) bool {
		return posts[firstIndex].PublishedAt.After(posts[secondIndex].PublishedAt)
	})
}

func postsToSummaries(posts []model.Post, pathPrefix string, englishURLBySourceFileName map[string]string) []model.PostSummary {
	postSummaries := make([]model.PostSummary, 0, len(posts))
	for _, currentPost := range posts {
		publicURL := "/" + currentPost.URL
		if pathPrefix != "" {
			publicURL = pathPrefix + "/" + currentPost.URL
		}
		postSummaries = append(postSummaries, model.PostSummary{
			ID:                currentPost.SourceFileName,
			Title:             currentPost.Title,
			Date:              currentPost.DateText,
			Description:       currentPost.Description,
			Draft:             currentPost.Draft,
			URL:               currentPost.URL,
			PublicURL:         publicURL,
			EnglishPublicURL:  englishURLBySourceFileName[currentPost.SourceFileName],
			Comments:          currentPost.Comments,
			Translation:       currentPost.Translation,
			TranslationStatus: currentPost.TranslationStatus,
		})
	}
	return postSummaries
}

func chineseTemplateLabels(themeDefault string) model.TemplateLabels {
	return model.TemplateLabels{
		Home:             "首页",
		Blog:             "文章",
		RSS:              "RSS",
		LatestPosts:      "最新文章",
		AllPosts:         "全部文章",
		ReadMore:         "阅读",
		PublishedAt:      "发布于",
		NoPosts:          "还没有文章。",
		LanguageSwitch:   "English",
		BackToList:       "返回文章列表",
		Footer:           "由 Go 静态渲染生成",
		ThemeButtonLabel: chineseThemeButtonLabel(themeDefault),
	}
}

func englishTemplateLabels(themeDefault string) model.TemplateLabels {
	return model.TemplateLabels{
		Home:             "Home",
		Blog:             "Blog",
		RSS:              "RSS",
		LatestPosts:      "Latest Posts",
		AllPosts:         "All Posts",
		ReadMore:         "Read",
		PublishedAt:      "Published",
		NoPosts:          "No posts yet.",
		LanguageSwitch:   "中文",
		BackToList:       "Back to posts",
		Footer:           "Generated by Go static rendering",
		ThemeButtonLabel: englishThemeButtonLabel(themeDefault),
	}
}

func chineseThemeButtonLabel(themeDefault string) string {
	switch themeDefault {
	case "light":
		return "主题：亮色"
	case "dark":
		return "主题：暗色"
	default:
		return "主题：自动"
	}
}

func englishThemeButtonLabel(themeDefault string) string {
	switch themeDefault {
	case "light":
		return "Theme: Light"
	case "dark":
		return "Theme: Dark"
	default:
		return "Theme: Auto"
	}
}

const examplePostOne = `---
title: "Docker 搭建第一篇博客"
date: "2026-05-04 12:00:00"
description: "这是一篇 Docker 部署笔记。"
draft: false
url: "1.html"
comments: true
translation: true
aliases:
  - "docker-old.html"
---

这里是第一篇示例文章的正文。

## 部署记录

使用 Docker 部署博客时，最重要的是把 ` + "`data/content/posts`" + ` 目录作为长期保存的内容目录。
`

const examplePostTwo = `---
title: "记录生活"
date: "2026-05-04 13:00:00"
description: "这是一篇用于验证排序和固定链接的示例。"
draft: false
url: "2.html"
comments: true
translation: true
aliases: []
---

这篇文章的标题和日期可以修改，但 ` + "`url`" + ` 字段决定的固定链接不会跟着标题变化。
`
