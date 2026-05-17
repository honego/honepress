package service

import (
	"encoding/json"
	"fmt"
	"hash/crc32"
	htmlTemplate "html/template"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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
		options.TemplateDir = filepath.Join("frontend", "theme", "templates")
	}
	if strings.TrimSpace(options.ThemeDefault) == "" {
		options.ThemeDefault = "auto"
	}
	options.PermalinkStructure = validation.NormalizePermalinkStructure(options.PermalinkStructure)
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

	if err := blogService.copyAssets(); err != nil {
		return err
	}
	if err := blogService.copyThemeDist(); err != nil {
		return err
	}

	if err := blogService.renderStaticPostPages(publishedPosts); err != nil {
		return err
	}

	if err := blogService.renderStaticMetadata(publishedPosts); err != nil {
		return err
	}

	log.Println("static site updated")
	return nil
}

func (blogService *BlogService) renderStaticMetadata(posts []model.Post) error {
	metadataRenderer := renderer.NewMetadataRenderer(blogService.options)
	if err := metadataRenderer.RenderRSS(filepath.Join(blogService.options.PublicDir, "rss.xml"), blogService.options.Title, blogService.options.Description, "/", posts, ""); err != nil {
		return err
	}

	sitemapPaths := []string{"/", "/archive.html"}
	for _, currentPost := range posts {
		sitemapPaths = append(sitemapPaths, publicPath(currentPost.URL))
	}
	return metadataRenderer.RenderSitemap(filepath.Join(blogService.options.PublicDir, "sitemap.xml"), sitemapPaths)
}

func (blogService *BlogService) renderStaticPostPages(posts []model.Post) error {
	metadataRenderer := renderer.NewMetadataRenderer(blogService.options)
	if err := metadataRenderer.RenderRedirect(filepath.Join(blogService.options.PublicDir, "blog.html"), "/archive.html"); err != nil {
		return err
	}
	if len(posts) == 0 {
		return nil
	}

	postShellFilePath := filepath.Join(blogService.options.PublicDir, "posts.html")
	postShellContent, err := os.ReadFile(postShellFilePath)
	if err != nil {
		return fmt.Errorf("read Next post shell at %s: %w", postShellFilePath, err)
	}

	for _, currentPost := range posts {
		postPageHTML := blogService.postShellWithSEO(string(postShellContent), currentPost)
		if err := filesystem.WriteFileCreatingDirectory(filepath.Join(blogService.options.PublicDir, currentPost.OutputPath), []byte(postPageHTML), 0644); err != nil {
			return fmt.Errorf("write static post page %s: %w", currentPost.URL, err)
		}
		if currentPost.SourceURL != "" && currentPost.SourceURL != currentPost.URL {
			if err := metadataRenderer.RenderRedirect(filepath.Join(blogService.options.PublicDir, currentPost.SourceURL), publicPath(currentPost.URL)); err != nil {
				return err
			}
		}
		for _, normalizedAlias := range currentPost.Aliases {
			if err := metadataRenderer.RenderRedirect(filepath.Join(blogService.options.PublicDir, normalizedAlias), publicPath(currentPost.URL)); err != nil {
				return err
			}
		}
	}
	return nil
}

func (blogService *BlogService) postShellWithSEO(postShellHTML string, post model.Post) string {
	htmlWithAttributes := blogService.withDocumentAttributes(postShellHTML)
	return injectSEOHead(htmlWithAttributes, blogService.postSEOHeadHTML(post))
}

func (blogService *BlogService) withDocumentAttributes(documentHTML string) string {
	lowerDocumentHTML := strings.ToLower(documentHTML)
	htmlTagStart := strings.Index(lowerDocumentHTML, "<html")
	if htmlTagStart < 0 {
		return documentHTML
	}
	htmlTagEndRelative := strings.Index(documentHTML[htmlTagStart:], ">")
	if htmlTagEndRelative < 0 {
		return documentHTML
	}
	htmlTagEnd := htmlTagStart + htmlTagEndRelative
	htmlTag := documentHTML[htmlTagStart : htmlTagEnd+1]
	htmlTag = setHTMLAttribute(htmlTag, "lang", "zh-CN")
	htmlTag = setHTMLAttribute(htmlTag, "data-theme", blogService.options.ThemeDefault)
	htmlTag = setHTMLAttribute(htmlTag, "data-font", blogService.options.Font)
	return documentHTML[:htmlTagStart] + htmlTag + documentHTML[htmlTagEnd+1:]
}

func injectSEOHead(documentHTML string, seoHeadHTML string) string {
	lowerDocumentHTML := strings.ToLower(documentHTML)
	headTagStart := strings.Index(lowerDocumentHTML, "<head")
	if headTagStart < 0 {
		return seoHeadHTML + documentHTML
	}
	headTagEndRelative := strings.Index(documentHTML[headTagStart:], ">")
	if headTagEndRelative < 0 {
		return seoHeadHTML + documentHTML
	}
	headContentStart := headTagStart + headTagEndRelative + 1
	headTagEnd := headContentStart - 1
	headCloseRelative := strings.Index(strings.ToLower(documentHTML[headContentStart:]), "</head>")
	if headCloseRelative < 0 {
		return documentHTML[:headContentStart] + seoHeadHTML + documentHTML[headContentStart:]
	}
	headClose := headContentStart + headCloseRelative
	headContent := stripSEOHeadElements(documentHTML[headContentStart:headClose])
	return documentHTML[:headTagEnd+1] + headContent + seoHeadHTML + documentHTML[headClose:]
}

func stripSEOHeadElements(headHTML string) string {
	for _, pattern := range seoHeadElementPatterns {
		headHTML = pattern.ReplaceAllString(headHTML, "")
	}
	return headHTML
}

var seoHeadElementPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?is)\s*<title\b[^>]*>.*?</title>`),
	regexp.MustCompile(`(?is)\s*<meta\b[^>]*(?:name|property)=["'](?:description|og:[^"']+|twitter:[^"']+)["'][^>]*>`),
	regexp.MustCompile(`(?is)\s*<link\b[^>]*rel=["'][^"']*\bcanonical\b[^"']*["'][^>]*>`),
	regexp.MustCompile(`(?is)\s*<link\b[^>]*rel=["'][^"']*(?:icon|apple-touch-icon)[^"']*["'][^>]*>`),
	regexp.MustCompile(`(?is)\s*<script\b[^>]*type=["']application/ld\+json["'][^>]*>.*?</script>`),
}

func setHTMLAttribute(htmlTag string, attributeName string, attributeValue string) string {
	attributePattern := regexp.MustCompile(`(?i)\s+` + regexp.QuoteMeta(attributeName) + `=(?:"[^"]*"|'[^']*'|[^\s>]+)`)
	attributeHTML := ` ` + attributeName + `="` + htmlTemplate.HTMLEscapeString(attributeValue) + `"`
	if attributePattern.MatchString(htmlTag) {
		return attributePattern.ReplaceAllString(htmlTag, attributeHTML)
	}
	return strings.TrimSuffix(htmlTag, ">") + attributeHTML + ">"
}

func (blogService *BlogService) postSEOHeadHTML(post model.Post) string {
	seoTitle := postSEOTitle(post, blogService.options.Title)
	seoDescription := postSEODescription(post)
	canonicalURL := string(seoPublicURL(blogService.options, publicPath(post.URL)))
	siteTitle := siteName(blogService.options.Title)
	seoImage := string(seoImageURL(blogService.options, blogService.options.SiteIconURL))
	structuredData := string(postStructuredData(blogService.options, post))

	var htmlBuilder strings.Builder
	htmlBuilder.WriteString("\n")
	htmlBuilder.WriteString(`<title>`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(seoTitle))
	htmlBuilder.WriteString("</title>\n")
	if strings.TrimSpace(seoDescription) != "" {
		writeMetaName(&htmlBuilder, "description", seoDescription)
	}
	writeLink(&htmlBuilder, "canonical", canonicalURL)
	writeMetaProperty(&htmlBuilder, "og:type", "article")
	writeMetaProperty(&htmlBuilder, "og:title", seoTitle)
	if strings.TrimSpace(seoDescription) != "" {
		writeMetaProperty(&htmlBuilder, "og:description", seoDescription)
	}
	writeMetaProperty(&htmlBuilder, "og:url", canonicalURL)
	writeMetaProperty(&htmlBuilder, "og:site_name", siteTitle)
	if seoImage != "" {
		writeMetaProperty(&htmlBuilder, "og:image", seoImage)
	}
	if seoImage != "" {
		writeMetaName(&htmlBuilder, "twitter:card", "summary_large_image")
	} else {
		writeMetaName(&htmlBuilder, "twitter:card", "summary")
	}
	writeMetaName(&htmlBuilder, "twitter:title", seoTitle)
	if strings.TrimSpace(seoDescription) != "" {
		writeMetaName(&htmlBuilder, "twitter:description", seoDescription)
	}
	if seoImage != "" {
		writeMetaName(&htmlBuilder, "twitter:image", seoImage)
	}
	if structuredData != "" {
		htmlBuilder.WriteString(`<script type="application/ld+json">`)
		htmlBuilder.WriteString(structuredData)
		htmlBuilder.WriteString("</script>\n")
	}
	faviconHref := string(postFaviconHref(post.Icon, blogService.options.SiteIconURL))
	if faviconHref != "" {
		writeLink(&htmlBuilder, "icon", faviconHref)
	} else {
		htmlBuilder.WriteString(`<link rel="icon" href="/honepress-black.svg" type="image/svg+xml" />` + "\n")
	}
	if blogService.options.Font == "douyin-sans" {
		htmlBuilder.WriteString(`<link rel="preload" href="/fonts/DouyinSansBold.ttf" as="font" type="font/ttf" crossorigin />` + "\n")
	}
	return htmlBuilder.String()
}

func writeMetaName(htmlBuilder *strings.Builder, name string, content string) {
	htmlBuilder.WriteString(`<meta name="`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(name))
	htmlBuilder.WriteString(`" content="`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(content))
	htmlBuilder.WriteString(`" />` + "\n")
}

func writeMetaProperty(htmlBuilder *strings.Builder, property string, content string) {
	htmlBuilder.WriteString(`<meta property="`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(property))
	htmlBuilder.WriteString(`" content="`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(content))
	htmlBuilder.WriteString(`" />` + "\n")
}

func writeLink(htmlBuilder *strings.Builder, rel string, href string) {
	htmlBuilder.WriteString(`<link rel="`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(rel))
	htmlBuilder.WriteString(`" href="`)
	htmlBuilder.WriteString(htmlTemplate.HTMLEscapeString(href))
	htmlBuilder.WriteString(`" />` + "\n")
}

func (blogService *BlogService) postPublicURL(frontMatter model.PostFrontMatter, postSlug string, postID string, publishedAt time.Time) string {
	structure := validation.NormalizePermalinkStructure(blogService.options.PermalinkStructure)
	replacements := map[string]string{
		"%year%":     publishedAt.Format("2006"),
		"%monthnum%": publishedAt.Format("01"),
		"%day%":      publishedAt.Format("02"),
		"%hour%":     publishedAt.Format("15"),
		"%minute%":   publishedAt.Format("04"),
		"%second%":   publishedAt.Format("05"),
		"%post_id%":  postID,
		"%postname%": postSlug,
		"%category%": firstCategorySlug(frontMatter.Tags),
	}
	publicURL := structure
	for tagName, tagValue := range replacements {
		publicURL = strings.ReplaceAll(publicURL, tagName, tagValue)
	}
	publicURL = strings.TrimPrefix(publicURL, "/")
	if strings.HasPrefix(publicURL, "?") {
		return publicURL
	}
	return strings.TrimLeft(publicURL, "/")
}

func stablePostID(sourceFileName string) string {
	checksum := crc32.ChecksumIEEE([]byte(sourceFileName))
	if checksum == 0 {
		return "1"
	}
	return fmt.Sprintf("%d", checksum)
}

func (blogService *BlogService) postOutputPath(publicURL string, postSlug string) string {
	if strings.HasPrefix(publicURL, "?") {
		queryValues, err := url.ParseQuery(strings.TrimPrefix(publicURL, "?"))
		if err == nil {
			if postID, err := validation.NormalizePostSlug(queryValues.Get("p")); err == nil {
				return filepath.ToSlash(filepath.Join("p", postID+".html"))
			}
		}
		return filepath.ToSlash(filepath.Join("p", postSlug+".html"))
	}
	trimmedPublicURL := strings.Trim(publicURL, "/")
	if trimmedPublicURL == "" {
		return filepath.ToSlash(filepath.Join(postSlug, "index.html"))
	}
	if strings.HasSuffix(publicURL, "/") || filepath.Ext(trimmedPublicURL) == "" {
		return filepath.ToSlash(filepath.Join(trimmedPublicURL, "index.html"))
	}
	return filepath.ToSlash(trimmedPublicURL)
}

func publicPath(postURL string) string {
	if strings.HasPrefix(postURL, "?") {
		return "/" + postURL
	}
	return "/" + strings.TrimPrefix(postURL, "/")
}

func firstCategorySlug(tags []string) string {
	for _, tag := range tags {
		if slug := slugifyPermalinkPart(tag); slug != "" {
			return slug
		}
	}
	return "uncategorized"
}

func slugifyPermalinkPart(value string) string {
	trimmedValue := strings.ToLower(strings.TrimSpace(value))
	if trimmedValue == "" {
		return ""
	}
	var slugBuilder strings.Builder
	previousDash := false
	for _, currentRune := range trimmedValue {
		if currentRune >= 'a' && currentRune <= 'z' || currentRune >= '0' && currentRune <= '9' {
			slugBuilder.WriteRune(currentRune)
			previousDash = false
			continue
		}
		if currentRune == '-' || currentRune == '_' {
			if !previousDash {
				slugBuilder.WriteRune('-')
				previousDash = true
			}
			continue
		}
		if !previousDash {
			slugBuilder.WriteRune('-')
			previousDash = true
		}
	}
	return strings.Trim(slugBuilder.String(), "-")
}

func (blogService *BlogService) scanPosts() ([]model.Post, error) {
	directoryEntries, err := os.ReadDir(blogService.options.PostsDir)
	if err != nil {
		return nil, fmt.Errorf("read posts directory: %w", err)
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
			return nil, fmt.Errorf("read post at %s: %w", sourceFilePath, err)
		}

		parsedFrontMatter, bodyMarkdownContent, err := renderer.ParsePostDocument(directoryEntry.Name(), sourceMarkdownContent)
		if err != nil {
			return nil, err
		}
		if err := validation.ValidateRequiredPostFields(parsedFrontMatter.Title, parsedFrontMatter.Date); err != nil {
			return nil, fmt.Errorf("validate post at %s: %w", sourceFilePath, err)
		}

		normalizedPermalink, err := validation.NormalizePermalinkWithFallback(parsedFrontMatter.URL, directoryEntry.Name())
		if err != nil {
			return nil, fmt.Errorf("normalize permalink for post at %s: %w", sourceFilePath, err)
		}
		postSlug, err := validation.NormalizePostSlugWithFallback(parsedFrontMatter.URL, directoryEntry.Name())
		if err != nil {
			return nil, fmt.Errorf("normalize post slug for post at %s: %w", sourceFilePath, err)
		}

		normalizedAliases := make([]string, 0, len(parsedFrontMatter.Aliases))
		for _, rawAlias := range parsedFrontMatter.Aliases {
			normalizedAlias, err := validation.NormalizePermalink(rawAlias)
			if err != nil {
				return nil, fmt.Errorf("normalize alias for post at %s: %w", sourceFilePath, err)
			}
			normalizedAliases = append(normalizedAliases, normalizedAlias)
		}

		publishedAt, err := validation.ParsePostDate(parsedFrontMatter.Date)
		if err != nil {
			return nil, fmt.Errorf("parse date for post at %s: %w", sourceFilePath, err)
		}
		postID := stablePostID(normalizedPermalink)
		publicPostURL := blogService.postPublicURL(parsedFrontMatter, postSlug, postID, publishedAt)

		var renderedPostHTML htmlTemplate.HTML
		if !parsedFrontMatter.Draft {
			renderedPostHTML, err = blogService.markdownRenderer.Render(bodyMarkdownContent)
			if err != nil {
				return nil, fmt.Errorf("render post at %s: %w", sourceFilePath, err)
			}
		}

		post := model.Post{
			SourceFileName: directoryEntry.Name(),
			SourceFilePath: sourceFilePath,
			Title:          parsedFrontMatter.Title,
			Icon:           parsedFrontMatter.Icon,
			Thumbnail:      parsedFrontMatter.Thumbnail,
			DateText:       parsedFrontMatter.Date,
			PublishedAt:    publishedAt,
			Description:    parsedFrontMatter.Description,
			SEOTitle:       parsedFrontMatter.SEOTitle,
			SEODescription: parsedFrontMatter.SEODescription,
			Draft:          parsedFrontMatter.Draft,
			SourceURL:      normalizedPermalink,
			Slug:           postSlug,
			PostID:         postID,
			URL:            publicPostURL,
			OutputPath:     blogService.postOutputPath(publicPostURL, postSlug),
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

func (blogService *BlogService) resetPublicDirectory() error {
	absoluteDataDirectoryPath, err := filepath.Abs(blogService.options.DataDir)
	if err != nil {
		return fmt.Errorf("resolve data directory: %w", err)
	}
	absolutePublicDirectoryPath, err := filepath.Abs(blogService.options.PublicDir)
	if err != nil {
		return fmt.Errorf("resolve public directory: %w", err)
	}
	relativePublicDirectoryPath, err := filepath.Rel(absoluteDataDirectoryPath, absolutePublicDirectoryPath)
	if err != nil {
		return fmt.Errorf("validate public directory: %w", err)
	}
	if relativePublicDirectoryPath == "." || strings.HasPrefix(relativePublicDirectoryPath, "..") {
		return fmt.Errorf("public directory must be inside data directory: %s", blogService.options.PublicDir)
	}

	if err := os.RemoveAll(absolutePublicDirectoryPath); err != nil {
		return fmt.Errorf("clean public directory: %w", err)
	}
	return filesystem.EnsureDirectory(absolutePublicDirectoryPath)
}

func (blogService *BlogService) copyThemeDist() error {
	if _, err := os.Stat(blogService.options.ThemeDistDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("theme dist directory does not exist at %s; build frontend/theme first", blogService.options.ThemeDistDir)
		}
		return fmt.Errorf("read theme dist directory: %w", err)
	}

	return filepath.WalkDir(blogService.options.ThemeDistDir, func(sourcePath string, directoryEntry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relativePath, err := filepath.Rel(blogService.options.ThemeDistDir, sourcePath)
		if err != nil {
			return fmt.Errorf("resolve theme asset path: %w", err)
		}
		if directoryEntry.IsDir() {
			return nil
		}
		targetPath := filepath.Join(blogService.options.PublicDir, relativePath)
		return filesystem.CopyFile(sourcePath, targetPath)
	})
}

func (blogService *BlogService) copyAssets() error {
	if _, err := os.Stat(blogService.options.AssetsDir); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("read assets directory: %w", err)
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
			return fmt.Errorf("resolve asset path: %w", err)
		}
		targetPath := filepath.Join(blogService.options.PublicDir, "assets", relativePath)
		return filesystem.CopyFile(sourcePath, targetPath)
	})
}

func siteName(title string) string {
	trimmedTitle := strings.TrimSpace(title)
	if trimmedTitle == "" {
		return "HonePress"
	}
	return trimmedTitle
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

func postStructuredData(options config.Options, post model.Post) htmlTemplate.JS {
	postURL := options.AbsoluteURL(publicPath(post.URL))
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

func postFaviconHref(postIcon string, siteIconURL string) htmlTemplate.URL {
	if isSupportedIconURL(postIcon) {
		return faviconHref(postIcon)
	}
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
		publicURL := publicPath(currentPost.URL)
		postSummaries = append(postSummaries, model.PostSummary{
			ID:          currentPost.SourceFileName,
			Title:       currentPost.Title,
			Thumbnail:   currentPost.Thumbnail,
			Date:        currentPost.DateText,
			Description: currentPost.Description,
			Draft:       currentPost.Draft,
			URL:         currentPost.URL,
			PublicURL:   publicURL,
			Tags:        currentPost.Tags,
			WordCount:   countVisibleRunes(currentPost.BodyMarkdown),
		})
	}
	return postSummaries
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
