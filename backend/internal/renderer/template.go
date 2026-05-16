package renderer

import (
	"encoding/xml"
	"fmt"
	htmlTemplate "html/template"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/filesystem"
	"github.com/honeok/honepress/internal/model"
)

type TemplateRenderer struct {
	options config.Options
}

func NewMetadataRenderer(options config.Options) *TemplateRenderer {
	return &TemplateRenderer{
		options: options,
	}
}

func (templateRenderer *TemplateRenderer) RenderRSS(targetFilePath string, channelTitle string, channelDescription string, channelPath string, posts []model.Post, pathPrefix string) error {
	rssFeed := &feeds.RssFeed{
		Title:       channelTitle,
		Link:        templateRenderer.options.AbsoluteURL(channelPath),
		Description: channelDescription,
	}
	for _, currentPost := range posts {
		publicPath := pathPrefix + publicPostPath(currentPost.URL)
		if pathPrefix == "" {
			publicPath = publicPostPath(currentPost.URL)
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

func (templateRenderer *TemplateRenderer) RenderRedirect(targetFilePath string, targetPublicPath string) error {
	absoluteTargetURL := templateRenderer.options.AbsoluteURL(targetPublicPath)
	redirectHTML := "<!doctype html>\n<html lang=\"zh-CN\" data-theme=\"auto\">\n<head>\n<meta charset=\"utf-8\">\n<meta http-equiv=\"refresh\" content=\"0; url=" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">\n<link rel=\"canonical\" href=\"" + htmlTemplate.HTMLEscapeString(absoluteTargetURL) + "\">\n<title>Page moved</title>\n</head>\n<body>\n<p>Page moved: <a href=\"" + htmlTemplate.HTMLEscapeString(targetPublicPath) + "\">continue</a></p>\n</body>\n</html>\n"
	return filesystem.WriteFileCreatingDirectory(targetFilePath, []byte(redirectHTML), 0644)
}

func publicPostPath(postURL string) string {
	if strings.HasPrefix(postURL, "?") {
		return "/" + postURL
	}
	return "/" + strings.TrimPrefix(postURL, "/")
}

type sitemapURLSet struct {
	XMLName   xml.Name     `xml:"urlset"`
	Namespace string       `xml:"xmlns,attr"`
	URLs      []sitemapURL `xml:"url"`
}

type sitemapURL struct {
	Location string `xml:"loc"`
}
