package renderer

import (
	"bytes"
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
		return nil, fmt.Errorf("解析模板失败：%w", err)
	}

	return &TemplateRenderer{
		templates: parsedTemplates,
		options:   options,
	}, nil
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
	redirectHTML := "<!doctype html>\n<html lang=\"zh-CN\" data-theme=\"auto\">\n<head>\n<meta charset=\"utf-8\">\n<meta http-equiv=\"refresh\" content=\"0; url=" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">\n<link rel=\"canonical\" href=\"" + htmlTemplate.HTMLEscapeString(absoluteTargetURL) + "\">\n<title>页面已移动</title>\n</head>\n<body>\n<p>页面已移动：<a href=\"" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">继续访问</a></p>\n</body>\n</html>\n"
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
		return fmt.Errorf("生成 RSS 失败：%w", err)
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
		return fmt.Errorf("生成 sitemap 失败：%w", err)
	}
	return filesystem.WriteFileCreatingDirectory(targetFilePath, append([]byte(xml.Header), xmlContent...), 0644)
}

// 复制前台样式
func (templateRenderer *TemplateRenderer) CopyStyle() error {
	styleContent, err := os.ReadFile(filepath.Join(templateRenderer.options.TemplateDir, "style.css"))
	if err != nil {
		return fmt.Errorf("读取前台样式失败：%w", err)
	}
	targetStylePath := filepath.Join(templateRenderer.options.PublicDir, "style.css")
	return filesystem.WriteFileCreatingDirectory(targetStylePath, styleContent, 0644)
}

func (templateRenderer *TemplateRenderer) executeHTMLTemplate(templateName string, targetFilePath string, templateData interface{}) error {
	var renderedHTMLBuffer bytes.Buffer
	if err := templateRenderer.templates.ExecuteTemplate(&renderedHTMLBuffer, templateName, templateData); err != nil {
		return fmt.Errorf("执行模板失败：%s：%w", templateName, err)
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
