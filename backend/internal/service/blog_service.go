package service

import (
	"encoding/json"
	"fmt"
	htmlTemplate "html/template"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/filesystem"
	"github.com/honeok/honepress/internal/model"
	"github.com/honeok/honepress/internal/renderer"
	"github.com/honeok/honepress/internal/validation"
)

// 博客业务
type BlogService struct {
	options          config.Options
	markdownRenderer *renderer.MarkdownRenderer
	renderMutex      sync.Mutex
}

// 创建博客服务
func NewBlogService(options config.Options) *BlogService {
	options = normalizeRuntimeOptions(options)
	return &BlogService{
		options:          options,
		markdownRenderer: renderer.NewMarkdownRenderer(),
	}
}

func normalizeRuntimeOptions(options config.Options) config.Options {
	if strings.TrimSpace(options.DataDir) == "" {
		options.DataDir = "data"
	}
	if strings.TrimSpace(options.ContentDir) == "" {
		options.ContentDir = filepath.Join(options.DataDir, "content")
	}
	if strings.TrimSpace(options.PostsDir) == "" {
		options.PostsDir = filepath.Join(options.ContentDir, "posts")
	}
	if strings.TrimSpace(options.PublicDir) == "" {
		options.PublicDir = filepath.Join(options.DataDir, "public")
	}
	if strings.TrimSpace(options.AssetsDir) == "" {
		options.AssetsDir = filepath.Join(options.DataDir, "assets")
	}
	if strings.TrimSpace(options.AdminDistDir) == "" {
		options.AdminDistDir = filepath.Join("dist", "admin")
	}
	if strings.TrimSpace(options.ThemeDistDir) == "" {
		options.ThemeDistDir = filepath.Join("dist", "theme")
	}
	if strings.TrimSpace(options.TemplateDir) == "" {
		options.TemplateDir = filepath.Join("backend", "templates")
	}
	if strings.TrimSpace(options.ThemeDefault) == "" {
		options.ThemeDefault = "auto"
	}
	switch strings.ToLower(strings.TrimSpace(options.Font)) {
	case "douyin-sans":
		options.Font = "douyin-sans"
	default:
		options.Font = "default"
	}
	return options
}

// 初始化并渲染站点
func (blogService *BlogService) InitializeAndRender() error {
	if err := blogService.ensureDataDirectories(); err != nil {
		return err
	}
	if err := blogService.createExamplePostsIfEmpty(); err != nil {
		return err
	}
	return blogService.RenderAll()
}

// 渲染静态站点
func (blogService *BlogService) RenderAll() error {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	return blogService.renderAllWithoutLock()
}

// 渲染 Markdown 预览
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
		blogService.options.PublicDir,
		blogService.options.AssetsDir,
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

	exampleFileName := "世界你好.md"
	exampleFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, exampleFileName)
	if err != nil {
		return err
	}
	exampleMarkdownContent := defaultFirstPost(time.Now().Format("2006-01-02 15:04:05"))
	if err := filesystem.WriteFileCreatingDirectory(exampleFilePath, []byte(exampleMarkdownContent), 0644); err != nil {
		return err
	}

	log.Println("未发现文章，已生成默认文章 世界你好.md。")
	return nil
}

func (blogService *BlogService) renderAllWithoutLock() error {
	if err := blogService.ensureDataDirectories(); err != nil {
		return err
	}

	posts, err := blogService.scanPosts()
	if err != nil {
		return err
	}
	publishedPosts := filterPublishedPosts(posts)
	sortPostsByDateDescending(publishedPosts)
	if err := validatePermalinkConflicts(publishedPosts); err != nil {
		return err
	}

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
	if err := blogService.copyAssets(); err != nil {
		return err
	}
	if err := blogService.copyThemeDist(); err != nil {
		return err
	}

	if err := blogService.renderSite(templateRenderer, publishedPosts); err != nil {
		return err
	}

	log.Println("静态站点已更新。")
	return nil
}

func (blogService *BlogService) scanPosts() ([]model.Post, error) {
	directoryEntries, err := os.ReadDir(blogService.options.PostsDir)
	if err != nil {
		return nil, fmt.Errorf("读取文章目录失败：%w", err)
	}

	posts := make([]model.Post, 0, len(directoryEntries))
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

		var renderedPostHTML htmlTemplate.HTML
		if !parsedFrontMatter.Draft {
			renderedPostHTML, err = blogService.markdownRenderer.Render(bodyMarkdownContent)
			if err != nil {
				return nil, fmt.Errorf("文章 %s 渲染失败：%w", sourceFilePath, err)
			}
		}

		post := model.Post{
			SourceFileName: directoryEntry.Name(),
			SourceFilePath: sourceFilePath,
			Title:          parsedFrontMatter.Title,
			Icon:           parsedFrontMatter.Icon,
			DateText:       parsedFrontMatter.Date,
			PublishedAt:    publishedAt,
			Description:    parsedFrontMatter.Description,
			SEOTitle:       parsedFrontMatter.SEOTitle,
			SEODescription: parsedFrontMatter.SEODescription,
			Draft:          parsedFrontMatter.Draft,
			URL:            normalizedPermalink,
			Aliases:        normalizedAliases,
			Tags:           parsedFrontMatter.Tags,
			BodyMarkdown:   bodyMarkdownContent,
			BodyHTML:       renderedPostHTML,
		}
		posts = append(posts, post)
	}

	sortPostsByDateDescending(posts)
	return posts, nil
}

func (blogService *BlogService) renderSite(templateRenderer *renderer.TemplateRenderer, posts []model.Post) error {
	postSummaries := postsToSummaries(posts)
	labels := templateLabels(blogService.options.ThemeDefault)
	archivePath := "/archive.html"
	siteViewData := model.SiteViewData{
		SiteTitle:       blogService.options.Title,
		SiteDescription: blogService.options.Description,
		SiteIconURL:     blogService.options.SiteIconURL,
		FaviconHref:     faviconHref(blogService.options.SiteIconURL),
		ThemeDefault:    blogService.options.ThemeDefault,
		Font:            blogService.options.Font,
		CanonicalPath:   "/",
		CanonicalURL:    seoPublicURL(blogService.options, "/"),
		SEOTitle:        homeSEOTitle(blogService.options.Title, blogService.options.Description),
		SEODescription:  seoDescription(blogService.options.Title, blogService.options.Description),
		SEOType:         "website",
		SEOImage:        seoImageURL(blogService.options, blogService.options.SiteIconURL),
		StructuredData:  siteStructuredData(blogService.options),
		HomePath:        "/",
		BlogPath:        archivePath,
		RSSPath:         "/rss.xml",
		SitemapPath:     "/sitemap.xml",
		Labels:          labels,
		Posts:           postSummaries,
		PostCount:       len(posts),
		WordCount:       totalPostWords(posts),
	}

	if err := templateRenderer.RenderIndex(filepath.Join(blogService.options.PublicDir, "index.html"), siteViewData); err != nil {
		return err
	}
	siteViewData.CanonicalPath = archivePath
	siteViewData.CanonicalURL = seoPublicURL(blogService.options, archivePath)
	siteViewData.SEOTitle = pageSEOTitle(labels.AllPosts, blogService.options.Title)
	siteViewData.SEODescription = seoDescription(blogService.options.Title, blogService.options.Description)
	siteViewData.SEOType = "website"
	siteViewData.StructuredData = archiveStructuredData(blogService.options, posts)
	if err := templateRenderer.RenderBlog(filepath.Join(blogService.options.PublicDir, "archive.html"), siteViewData); err != nil {
		return err
	}
	if err := templateRenderer.RenderRedirect(filepath.Join(blogService.options.PublicDir, "blog.html"), archivePath); err != nil {
		return err
	}

	postRenderErrors := make(chan error, len(posts))
	var postRenderWaitGroup sync.WaitGroup
	for _, currentPost := range posts {
		currentPost := currentPost
		postRenderWaitGroup.Add(1)
		go func() {
			defer postRenderWaitGroup.Done()

			postViewData := model.PostViewData{
				SiteTitle:       blogService.options.Title,
				SiteDescription: blogService.options.Description,
				SiteIconURL:     blogService.options.SiteIconURL,
				FaviconHref:     postFaviconHref(currentPost.Icon, blogService.options.SiteIconURL),
				ThemeDefault:    blogService.options.ThemeDefault,
				Font:            blogService.options.Font,
				CanonicalPath:   "/" + currentPost.URL,
				CanonicalURL:    seoPublicURL(blogService.options, "/"+currentPost.URL),
				SEOTitle:        postSEOTitle(currentPost, blogService.options.Title),
				SEODescription:  postSEODescription(currentPost),
				SEOType:         "article",
				SEOImage:        seoImageURL(blogService.options, blogService.options.SiteIconURL),
				StructuredData:  postStructuredData(blogService.options, currentPost),
				HomePath:        "/",
				BlogPath:        archivePath,
				RSSPath:         "/rss.xml",
				SitemapPath:     "/sitemap.xml",
				Labels:          labels,
				Post:            currentPost,
				CommentHTML:     blogService.commentHTML(),
			}
			if err := templateRenderer.RenderPost(filepath.Join(blogService.options.PublicDir, currentPost.URL), postViewData); err != nil {
				postRenderErrors <- err
				return
			}
			for _, normalizedAlias := range currentPost.Aliases {
				if err := templateRenderer.RenderRedirect(filepath.Join(blogService.options.PublicDir, normalizedAlias), "/"+currentPost.URL); err != nil {
					postRenderErrors <- err
					return
				}
			}
		}()
	}
	postRenderWaitGroup.Wait()
	close(postRenderErrors)
	for renderErr := range postRenderErrors {
		if renderErr != nil {
			return renderErr
		}
	}
	if err := templateRenderer.RenderRSS(filepath.Join(blogService.options.PublicDir, "rss.xml"), blogService.options.Title, blogService.options.Description, "/", posts, ""); err != nil {
		return err
	}

	sitemapPaths := []string{"/", archivePath}
	for _, currentPost := range posts {
		sitemapPaths = append(sitemapPaths, "/"+currentPost.URL)
	}
	return templateRenderer.RenderSitemap(filepath.Join(blogService.options.PublicDir, "sitemap.xml"), sitemapPaths)
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

func (blogService *BlogService) copyThemeDist() error {
	if _, err := os.Stat(blogService.options.ThemeDistDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("前台主题构建目录不存在：%s，请先构建 frontend/theme", blogService.options.ThemeDistDir)
		}
		return fmt.Errorf("读取前台主题构建目录失败：%w", err)
	}

	return filepath.WalkDir(blogService.options.ThemeDistDir, func(sourcePath string, directoryEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if directoryEntry.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(blogService.options.ThemeDistDir, sourcePath)
		if err != nil {
			return fmt.Errorf("解析前台主题资源路径失败：%w", err)
		}
		targetPath := filepath.Join(blogService.options.PublicDir, relativePath)
		return filesystem.CopyFile(sourcePath, targetPath)
	})
}

func (blogService *BlogService) copyAssets() error {
	if _, err := os.Stat(blogService.options.AssetsDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("读取资源目录失败：%w", err)
	}

	return filepath.WalkDir(blogService.options.AssetsDir, func(sourcePath string, directoryEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if directoryEntry.IsDir() {
			return nil
		}
		relativePath, err := filepath.Rel(blogService.options.AssetsDir, sourcePath)
		if err != nil {
			return fmt.Errorf("解析资源路径失败：%w", err)
		}
		targetPath := filepath.Join(blogService.options.PublicDir, "assets", relativePath)
		return filesystem.CopyFile(sourcePath, targetPath)
	})
}

func (blogService *BlogService) commentHTML() htmlTemplate.HTML {
	if !blogService.options.Comment.Enabled {
		return ""
	}
	if !blogService.options.Comment.HasRequiredGiscusConfig() {
		log.Println("警告：giscus 配置不完整，已跳过评论容器。")
		return ""
	}

	commentAttributes := map[string]string{
		"data-repo":        blogService.options.Comment.GiscusRepo,
		"data-repo-id":     blogService.options.Comment.GiscusRepoID,
		"data-category":    blogService.options.Comment.GiscusCategory,
		"data-category-id": blogService.options.Comment.GiscusCategoryID,
	}
	attributeNames := []string{
		"data-repo",
		"data-repo-id",
		"data-category",
		"data-category-id",
	}

	var htmlBuilder strings.Builder
	htmlBuilder.WriteString(`<section id="comments" class="comments" data-giscus-comments`)
	for _, attributeName := range attributeNames {
		htmlBuilder.WriteString(` `)
		htmlBuilder.WriteString(attributeName)
		htmlBuilder.WriteString(`="`)
		htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(commentAttributes[attributeName]))
		htmlBuilder.WriteString(`"`)
	}
	htmlBuilder.WriteString(`></section>`)

	return htmlTemplate.HTML(htmlBuilder.String())
}

func siteName(title string) string {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return "HonePress"
	}
	return trimmedTitle
}

func homeSEOTitle(siteTitle string, siteDescription string) string {
	name := siteName(siteTitle)
	description := strings.TrimSpace(siteDescription)
	if description == "" {
		return name
	}
	return name + " - " + description
}

func pageSEOTitle(pageTitle string, siteTitle string) string {
	trimmedPageTitle := strings.TrimSpace(pageTitle)
	name := siteName(siteTitle)
	if trimmedPageTitle == "" || trimmedPageTitle == name {
		return name
	}
	return trimmedPageTitle + " - " + name
}

func seoDescription(primary string, fallback string) string {
	if trimmedFallback := strings.TrimSpace(fallback); trimmedFallback != "" {
		return trimmedFallback
	}
	return strings.TrimSpace(primary)
}

func seoPublicURL(options config.Options, publicPath string) htmlTemplate.URL {
	return htmlTemplate.URL(options.AbsoluteURL(publicPath))
}

func seoImageURL(options config.Options, imageURL string) htmlTemplate.URL {
	trimmedImageURL := strings.TrimSpace(imageURL)
	if trimmedImageURL == "" || strings.HasPrefix(trimmedImageURL, "data:") {
		return ""
	}
	return htmlTemplate.URL(options.AbsoluteURL(trimmedImageURL))
}

func postSEOTitle(post model.Post, siteTitle string) string {
	if trimmedSEOTitle := strings.TrimSpace(post.SEOTitle); trimmedSEOTitle != "" {
		return trimmedSEOTitle
	}
	return pageSEOTitle(post.Title, siteTitle)
}

func postSEODescription(post model.Post) string {
	if trimmedSEODescription := strings.TrimSpace(post.SEODescription); trimmedSEODescription != "" {
		return trimmedSEODescription
	}
	return seoDescription(post.Title, post.Description)
}

func structuredData(document map[string]interface{}) htmlTemplate.JS {
	encodedDocument, err := json.Marshal(document)
	if err != nil {
		return ""
	}
	return htmlTemplate.JS(encodedDocument)
}

func addPublisher(document map[string]interface{}, options config.Options) {
	publisher := map[string]interface{}{
		"@type": "Organization",
		"name":  siteName(options.Title),
	}
	if logoURL := seoImageURL(options, options.SiteIconURL); logoURL != "" {
		publisher["logo"] = map[string]interface{}{
			"@type": "ImageObject",
			"url":   string(logoURL),
		}
	}
	document["publisher"] = publisher
}

func siteStructuredData(options config.Options) htmlTemplate.JS {
	document := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "Blog",
		"name":        siteName(options.Title),
		"description": seoDescription(options.Title, options.Description),
		"url":         options.AbsoluteURL("/"),
	}
	addPublisher(document, options)
	return structuredData(document)
}

func archiveStructuredData(options config.Options, posts []model.Post) htmlTemplate.JS {
	items := make([]map[string]interface{}, 0, len(posts))
	for postIndex, currentPost := range posts {
		items = append(items, map[string]interface{}{
			"@type":    "ListItem",
			"position": postIndex + 1,
			"name":     currentPost.Title,
			"url":      options.AbsoluteURL("/" + currentPost.URL),
		})
	}
	document := map[string]interface{}{
		"@context":    "https://schema.org",
		"@type":       "CollectionPage",
		"name":        pageSEOTitle("Archive", options.Title),
		"description": seoDescription(options.Title, options.Description),
		"url":         options.AbsoluteURL("/archive.html"),
		"mainEntity": map[string]interface{}{
			"@type":           "ItemList",
			"itemListElement": items,
		},
	}
	return structuredData(document)
}

func postStructuredData(options config.Options, post model.Post) htmlTemplate.JS {
	postURL := options.AbsoluteURL("/" + post.URL)
	document := map[string]interface{}{
		"@context":         "https://schema.org",
		"@type":            "BlogPosting",
		"headline":         post.Title,
		"description":      postSEODescription(post),
		"datePublished":    post.PublishedAt.Format(time.RFC3339),
		"dateModified":     post.PublishedAt.Format(time.RFC3339),
		"url":              postURL,
		"mainEntityOfPage": map[string]interface{}{"@type": "WebPage", "@id": postURL},
		"author":           map[string]interface{}{"@type": "Person", "name": siteName(options.Title)},
	}
	if len(post.Tags) > 0 {
		document["keywords"] = strings.Join(post.Tags, ", ")
	}
	if imageURL := seoImageURL(options, options.SiteIconURL); imageURL != "" {
		document["image"] = string(imageURL)
	}
	addPublisher(document, options)
	return structuredData(document)
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

func postsToSummaries(posts []model.Post) []model.PostSummary {
	postSummaries := make([]model.PostSummary, 0, len(posts))
	for _, currentPost := range posts {
		publicURL := "/" + currentPost.URL
		postSummaries = append(postSummaries, model.PostSummary{
			ID:          currentPost.SourceFileName,
			Title:       currentPost.Title,
			Date:        currentPost.DateText,
			Description: currentPost.Description,
			Draft:       currentPost.Draft,
			URL:         currentPost.URL,
			PublicURL:   publicURL,
			Tags:        currentPost.Tags,
		})
	}
	return postSummaries
}

func totalPostWords(posts []model.Post) int {
	totalWords := 0
	for _, currentPost := range posts {
		totalWords += countVisibleRunes(currentPost.BodyMarkdown)
	}
	return totalWords
}

func countVisibleRunes(text string) int {
	visibleRuneCount := 0
	for _, currentRune := range text {
		if unicode.IsSpace(currentRune) {
			continue
		}
		visibleRuneCount++
	}
	return visibleRuneCount
}

func postFaviconHref(postIcon string, siteIconURL string) htmlTemplate.URL {
	if emojiHref := emojiFaviconHref(postIcon); emojiHref != "" {
		return emojiHref
	}
	return faviconHref(siteIconURL)
}

func faviconHref(iconURL string) htmlTemplate.URL {
	trimmedIconURL := strings.TrimSpace(iconURL)
	if trimmedIconURL == "" {
		return ""
	}
	return htmlTemplate.URL(trimmedIconURL)
}

func emojiFaviconHref(emoji string) htmlTemplate.URL {
	trimmedEmoji := strings.TrimSpace(emoji)
	if trimmedEmoji == "" {
		return ""
	}
	svgContent := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text x="50%" y="50%" style="dominant-baseline:central;text-anchor:middle;font-size:86px;">` + htmlTemplate.HTMLEscapeString(trimmedEmoji) + `</text></svg>`
	return htmlTemplate.URL("data:image/svg+xml," + url.PathEscape(svgContent))
}

func templateLabels(themeDefault string) model.TemplateLabels {
	return model.TemplateLabels{
		Home:             "首页",
		Blog:             "归档",
		RSS:              "RSS",
		Sitemap:          "站点地图",
		AllPosts:         "归档",
		ReadMore:         "阅读",
		PublishedAt:      "发布于",
		NoPosts:          "还没有文章。",
		BackToList:       "返回",
		Footer:           "Powered by HonePress",
		ThemeButtonLabel: themeButtonLabel(themeDefault),
	}
}

func themeButtonLabel(themeDefault string) string {
	switch themeDefault {
	case "light":
		return "主题：亮色"
	case "dark":
		return "主题：暗色"
	default:
		return "主题：自动"
	}
}

func defaultFirstPost(dateText string) string {
	return fmt.Sprintf(`---
title: "世界你好"
icon: "☘️"
date: "%s"
description: "欢迎使用 HonePress。"
draft: false
url: "hello.html"
aliases: []
tags:
  - HonePress
---

欢迎使用 HonePress 。这是您的第一篇文章，编辑或删除它，然后开始写作吧！
`, dateText)
}
