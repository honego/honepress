package renderer

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	htmlTemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/filesystem"
	"github.com/honeok/honepress/internal/model"
)

// 静态站点文件渲染器
type TemplateRenderer struct {
	templates *htmlTemplate.Template
	options   config.Options
}

// 创建模板渲染器
func NewTemplateRenderer(options config.Options) (*TemplateRenderer, error) {
	parsedTemplates, err := htmlTemplate.ParseFiles(
		filepath.Join(options.TemplateDir, "index.html"),
		filepath.Join(options.TemplateDir, "blog.html"),
		filepath.Join(options.TemplateDir, "post.html"),
	)
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}

	return &TemplateRenderer{
		templates: parsedTemplates,
		options:   options,
	}, nil
}

// 前台主题构建产物
type ThemeAssets struct {
	ScriptPath htmlTemplate.URL
	StylePaths []htmlTemplate.URL
}

type viteManifestEntry struct {
	File    string   `json:"file"`
	CSS     []string `json:"css"`
	IsEntry bool     `json:"isEntry"`
	Name    string   `json:"name"`
	Src     string   `json:"src"`
}

func ResolveThemeAssets(themeDistDir string) (ThemeAssets, error) {
	manifestContent, manifestPath, err := readViteManifest(themeDistDir)
	if err != nil {
		return ThemeAssets{}, err
	}

	var manifestEntries map[string]viteManifestEntry
	if err := json.Unmarshal(manifestContent, &manifestEntries); err != nil {
		return ThemeAssets{}, fmt.Errorf("decode Vite manifest at %s: %w", manifestPath, err)
	}

	themeEntry, err := findThemeManifestEntry(manifestEntries)
	if err != nil {
		return ThemeAssets{}, fmt.Errorf("resolve theme assets from %s: %w", manifestPath, err)
	}

	themeAssets := ThemeAssets{
		ScriptPath: publicAssetPath(themeEntry.File),
		StylePaths: make([]htmlTemplate.URL, 0, len(themeEntry.CSS)),
	}
	for _, stylesheetPath := range themeEntry.CSS {
		themeAssets.StylePaths = append(themeAssets.StylePaths, publicAssetPath(stylesheetPath))
	}
	return themeAssets, nil
}

func readViteManifest(themeDistDir string) ([]byte, string, error) {
	manifestPaths := []string{
		filepath.Join(themeDistDir, ".vite", "manifest.json"),
		filepath.Join(themeDistDir, "manifest.json"),
	}
	for _, manifestPath := range manifestPaths {
		manifestContent, err := os.ReadFile(manifestPath)
		if err == nil {
			return manifestContent, manifestPath, nil
		}
		if !os.IsNotExist(err) {
			return nil, "", fmt.Errorf("read Vite manifest at %s: %w", manifestPath, err)
		}
	}
	return nil, "", fmt.Errorf("Vite manifest does not exist in %s; build frontend/theme first", themeDistDir)
}

func findThemeManifestEntry(manifestEntries map[string]viteManifestEntry) (viteManifestEntry, error) {
	if themeEntry, exists := manifestEntries["src/theme.ts"]; exists && strings.TrimSpace(themeEntry.File) != "" {
		return themeEntry, nil
	}
	for manifestKey, manifestEntry := range manifestEntries {
		normalizedKey := filepath.ToSlash(manifestKey)
		normalizedSource := filepath.ToSlash(manifestEntry.Src)
		if strings.TrimSpace(manifestEntry.File) == "" {
			continue
		}
		if manifestEntry.Name == "theme" || strings.HasSuffix(normalizedKey, "src/theme.ts") || strings.HasSuffix(normalizedSource, "src/theme.ts") {
			return manifestEntry, nil
		}
	}
	for _, manifestEntry := range manifestEntries {
		if manifestEntry.IsEntry && strings.TrimSpace(manifestEntry.File) != "" {
			return manifestEntry, nil
		}
	}
	return viteManifestEntry{}, fmt.Errorf("theme entry not found")
}

func publicAssetPath(assetPath string) htmlTemplate.URL {
	trimmedAssetPath := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(assetPath)), "/")
	if trimmedAssetPath == "" {
		return ""
	}
	return htmlTemplate.URL("/" + trimmedAssetPath)
}

// 渲染首页
func (templateRenderer *TemplateRenderer) RenderIndex(targetFilePath string, siteViewData model.SiteViewData) error {
	return templateRenderer.executeHTMLTemplate("index.html", targetFilePath, siteViewData)
}

// 渲染文章列表页
func (templateRenderer *TemplateRenderer) RenderBlog(targetFilePath string, siteViewData model.SiteViewData) error {
	return templateRenderer.executeHTMLTemplate("blog.html", targetFilePath, siteViewData)
}

// 渲染文章详情页
func (templateRenderer *TemplateRenderer) RenderPost(targetFilePath string, postViewData model.PostViewData) error {
	return templateRenderer.executeHTMLTemplate("post.html", targetFilePath, postViewData)
}

// 写入别名跳转页
func (templateRenderer *TemplateRenderer) RenderRedirect(targetFilePath string, targetPublicPath string) error {
	absoluteTargetURL := templateRenderer.options.AbsoluteURL(targetPublicPath)
	redirectHTML := "<!doctype html>\n<html lang=\"zh-CN\" data-theme=\"auto\">\n<head>\n<meta charset=\"utf-8\">\n<meta http-equiv=\"refresh\" content=\"0; url=" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">\n<link rel=\"canonical\" href=\"" + htmlTemplate.HTMLEscapeString(absoluteTargetURL) + "\">\n<title>Page moved</title>\n</head>\n<body>\n<p>Page moved: <a href=\"" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">continue</a></p>\n</body>\n</html>\n"
	return filesystem.WriteFileCreatingDirectory(targetFilePath, []byte(redirectHTML), 0644)
}

// 生成 RSS
func (templateRenderer *TemplateRenderer) RenderRSS(targetFilePath string, channelTitle string, channelDescription string, channelPath string, posts []model.Post, pathPrefix string) error {
	rssFeed := &feeds.RssFeed{
		Title:       channelTitle,
		Link:        templateRenderer.options.AbsoluteURL(channelPath),
		Description: channelDescription,
	}
	for _, currentPost := range posts {
		publicPath := pathPrefix + "/" + currentPost.URL
		if pathPrefix == "" {
			publicPath = "/" + currentPost.URL
		}
		absolutePostURL := templateRenderer.options.AbsoluteURL(publicPath)
		rssFeed.Items = append(rssFeed.Items, &feeds.RssItem{
			Title:       currentPost.Title,
			Link:        absolutePostURL,
			Guid:        &feeds.RssGuid{Id: absolutePostURL, IsPermaLink: "true"},
			PubDate:     currentPost.PublishedAt.Format(time.RFC1123Z),
			Description: currentPost.Description,
			Category:    strings.Join(currentPost.Tags, ","),
		})
	}

	xmlContent, err := feeds.ToXML(rssFeed)
	if err != nil {
		return fmt.Errorf("generate RSS: %w", err)
	}
	return filesystem.WriteFileCreatingDirectory(targetFilePath, []byte(xmlContent), 0644)
}

// 生成 sitemap
func (templateRenderer *TemplateRenderer) RenderSitemap(targetFilePath string, publicPaths []string) error {
	sitemapURLs := make([]sitemapURL, 0, len(publicPaths))
	for _, publicPath := range publicPaths {
		sitemapURLs = append(sitemapURLs, sitemapURL{
			Location: templateRenderer.options.AbsoluteURL(publicPath),
		})
	}

	sitemapDocument := sitemapURLSet{
		Namespace: "http://www.sitemaps.org/schemas/sitemap/0.9",
		URLs:      sitemapURLs,
	}

	xmlContent, err := xml.MarshalIndent(sitemapDocument, "", "  ")
	if err != nil {
		return fmt.Errorf("generate sitemap: %w", err)
	}
	return filesystem.WriteFileCreatingDirectory(targetFilePath, append([]byte(xml.Header), xmlContent...), 0644)
}

func (templateRenderer *TemplateRenderer) executeHTMLTemplate(templateName string, targetFilePath string, templateData interface{}) error {
	var renderedHTMLBuffer bytes.Buffer
	if err := templateRenderer.templates.ExecuteTemplate(&renderedHTMLBuffer, templateName, templateData); err != nil {
		return fmt.Errorf("execute template %s: %w", templateName, err)
	}
	return filesystem.WriteFileCreatingDirectory(targetFilePath, renderedHTMLBuffer.Bytes(), 0644)
}

type sitemapURLSet struct {
	XMLName   xml.Name     `xml:"urlset"`
	Namespace string       `xml:"xmlns,attr"`
	URLs      []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}
