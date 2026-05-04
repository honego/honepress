package renderer

import (
	"bytes"
	"encoding/xml"
	"fmt"
	htmlTemplate "html/template"
	"os"
	"path/filepath"
	"time"

	"github.com/honeok/blog/common/filesystem"
	"github.com/honeok/blog/model"
	"github.com/honeok/blog/option"
	templatefiles "github.com/honeok/blog/template"
)

// TemplateRenderer 负责把文章模型写成最终静态 HTML、RSS 和 sitemap 文件。
type TemplateRenderer struct {
	templates *htmlTemplate.Template
	options   option.Options
}

// NewTemplateRenderer 从二进制内置模板解析页面，运行期只需要 app 自身。
func NewTemplateRenderer(options option.Options) (*TemplateRenderer, error) {
	parsedTemplates, err := htmlTemplate.ParseFS(templatefiles.FS, "index.html", "blog.html", "post.html")
	if err != nil {
		return nil, fmt.Errorf("解析模板失败：%w", err)
	}

	return &TemplateRenderer{
		templates: parsedTemplates,
		options:   options,
	}, nil
}

// RenderIndex 渲染首页。
func (templateRenderer *TemplateRenderer) RenderIndex(targetFilePath string, siteViewData model.SiteViewData) error {
	return templateRenderer.executeHTMLTemplate("index.html", targetFilePath, siteViewData)
}

// RenderBlog 渲染文章列表页。
func (templateRenderer *TemplateRenderer) RenderBlog(targetFilePath string, siteViewData model.SiteViewData) error {
	return templateRenderer.executeHTMLTemplate("blog.html", targetFilePath, siteViewData)
}

// RenderPost 渲染文章详情页，评论脚本只会通过这个模板输出。
func (templateRenderer *TemplateRenderer) RenderPost(targetFilePath string, postViewData model.PostViewData) error {
	return templateRenderer.executeHTMLTemplate("post.html", targetFilePath, postViewData)
}

// RenderRedirect 写入别名跳转页，旧链接保留对搜索引擎和外部引用更友好。
func (templateRenderer *TemplateRenderer) RenderRedirect(targetFilePath string, targetPublicPath string) error {
	absoluteTargetURL := templateRenderer.options.AbsoluteURL(targetPublicPath)
	redirectHTML := "<!doctype html>\n<html lang=\"zh-CN\" data-theme=\"auto\">\n<head>\n<meta charset=\"utf-8\">\n<meta http-equiv=\"refresh\" content=\"0; url=" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">\n<link rel=\"canonical\" href=\"" + htmlTemplate.HTMLEscapeString(absoluteTargetURL) + "\">\n<title>页面已移动</title>\n</head>\n<body>\n<p>页面已移动：<a href=\"" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">继续访问</a></p>\n</body>\n</html>\n"
	return filesystem.WriteFileCreatingDirectory(targetFilePath, []byte(redirectHTML), 0644)
}

// RenderRSS 生成 RSS 2.0，guid 使用完整固定链接，保证订阅器不会因标题变化误判新文章。
func (templateRenderer *TemplateRenderer) RenderRSS(targetFilePath string, channelTitle string, channelDescription string, channelPath string, posts []model.Post, pathPrefix string) error {
	rssItems := make([]rssItem, 0, len(posts))
	for _, currentPost := range posts {
		publicPath := pathPrefix + "/" + currentPost.URL
		if pathPrefix == "" {
			publicPath = "/" + currentPost.URL
		}
		absolutePostURL := templateRenderer.options.AbsoluteURL(publicPath)
		rssItems = append(rssItems, rssItem{
			Title:       currentPost.Title,
			Link:        absolutePostURL,
			GUID:        rssGUID{Value: absolutePostURL, IsPermaLink: "true"},
			PubDate:     currentPost.PublishedAt.Format(time.RFC1123Z),
			Description: currentPost.Description,
		})
	}

	rssDocument := rssDocument{
		Version: "2.0",
		Channel: rssChannel{
			Title:       channelTitle,
			Link:        templateRenderer.options.AbsoluteURL(channelPath),
			Description: channelDescription,
			Items:       rssItems,
		},
	}

	xmlContent, err := xml.MarshalIndent(rssDocument, "", "  ")
	if err != nil {
		return fmt.Errorf("生成 RSS 失败：%w", err)
	}
	return filesystem.WriteFileCreatingDirectory(targetFilePath, append([]byte(xml.Header), xmlContent...), 0644)
}

// RenderSitemap 生成 sitemap，后台和 API 路径永远不会进入站点地图。
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

// CopyStyle 把内置 CSS 复制到 public，静态页面不依赖外部模板目录。
func (templateRenderer *TemplateRenderer) CopyStyle() error {
	styleContent, err := templatefiles.FS.ReadFile("style.css")
	if err != nil {
		return fmt.Errorf("读取内置样式失败：%w", err)
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

func removeFileIfExists(filePath string) error {
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除文件失败：%s：%w", filePath, err)
	}
	return nil
}

type rssDocument struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	Items       []rssItem `xml:"item"`
}

type rssItem struct {
	Title       string  `xml:"title"`
	Link        string  `xml:"link"`
	GUID        rssGUID `xml:"guid"`
	PubDate     string  `xml:"pubDate"`
	Description string  `xml:"description"`
}

type rssGUID struct {
	Value       string `xml:",chardata"`
	IsPermaLink string `xml:"isPermaLink,attr"`
}

type sitemapURLSet struct {
	XMLName   xml.Name     `xml:"urlset"`
	Namespace string       `xml:"xmlns,attr"`
	URLs      []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}
