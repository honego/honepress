package renderer

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/honeok/honepress/backend/internal/model"
)

type postFrontMatterYAML struct {
	Title          string   `yaml:"title"`
	Icon           string   `yaml:"icon,omitempty"`
	Date           string   `yaml:"date"`
	Description    string   `yaml:"description"`
	SEOTitle       string   `yaml:"seoTitle,omitempty"`
	SEODescription string   `yaml:"seoDescription,omitempty"`
	Draft          bool     `yaml:"draft"`
	URL            string   `yaml:"url"`
	Aliases        []string `yaml:"aliases"`
	Tags           []string `yaml:"tags"`
}

func ParsePostDocument(sourceFileName string, markdownContent []byte) (model.PostFrontMatter, string, error) {
	frontMatterContent, bodyMarkdownContent, hasFrontMatter := splitFrontMatter(markdownContent)
	if !hasFrontMatter {
		return model.PostFrontMatter{}, "", fmt.Errorf("缺少文章元信息：%s", sourceFileName)
	}

	var decodedFrontMatter postFrontMatterYAML
	if err := yaml.Unmarshal([]byte(frontMatterContent), &decodedFrontMatter); err != nil {
		return model.PostFrontMatter{}, "", fmt.Errorf("解析文章元信息失败：%s：%w", sourceFileName, err)
	}

	parsedFrontMatter := model.PostFrontMatter{
		Title:          strings.TrimSpace(decodedFrontMatter.Title),
		Icon:           strings.TrimSpace(decodedFrontMatter.Icon),
		Date:           strings.TrimSpace(decodedFrontMatter.Date),
		Description:    strings.TrimSpace(decodedFrontMatter.Description),
		SEOTitle:       strings.TrimSpace(decodedFrontMatter.SEOTitle),
		SEODescription: strings.TrimSpace(decodedFrontMatter.SEODescription),
		Draft:          decodedFrontMatter.Draft,
		URL:            strings.TrimSpace(decodedFrontMatter.URL),
		Aliases:        normalizeStringList(decodedFrontMatter.Aliases),
		Tags:           normalizeStringList(decodedFrontMatter.Tags),
	}

	return parsedFrontMatter, bodyMarkdownContent, nil
}

func BuildPostDocument(frontMatter model.PostFrontMatter, bodyMarkdownContent string) ([]byte, error) {
	encodedFrontMatter, err := yaml.Marshal(postFrontMatterYAML{
		Title:          frontMatter.Title,
		Icon:           frontMatter.Icon,
		Date:           frontMatter.Date,
		Description:    frontMatter.Description,
		SEOTitle:       frontMatter.SEOTitle,
		SEODescription: frontMatter.SEODescription,
		Draft:          frontMatter.Draft,
		URL:            frontMatter.URL,
		Aliases:        frontMatter.Aliases,
		Tags:           frontMatter.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("生成文章元信息失败：%w", err)
	}

	normalizedBodyMarkdownContent := strings.TrimLeft(bodyMarkdownContent, "\r\n")
	return []byte("---\n" + string(encodedFrontMatter) + "---\n\n" + normalizedBodyMarkdownContent), nil
}

func splitFrontMatter(markdownContent []byte) (string, string, bool) {
	normalizedMarkdownContent := strings.ReplaceAll(string(markdownContent), "\r\n", "\n")
	if !strings.HasPrefix(normalizedMarkdownContent, "---\n") {
		return "", normalizedMarkdownContent, false
	}

	frontMatterEndIndex := strings.Index(normalizedMarkdownContent[4:], "\n---\n")
	if frontMatterEndIndex == -1 {
		return "", normalizedMarkdownContent, false
	}

	frontMatterContent := normalizedMarkdownContent[4 : 4+frontMatterEndIndex]
	bodyMarkdownContent := strings.TrimLeft(normalizedMarkdownContent[4+frontMatterEndIndex+5:], "\n")
	return frontMatterContent, bodyMarkdownContent, true
}

func normalizeStringList(rawValues []string) []string {
	normalizedValues := make([]string, 0, len(rawValues))
	seenValues := make(map[string]struct{})
	for _, rawValue := range rawValues {
		trimmedValue := strings.TrimSpace(rawValue)
		if trimmedValue == "" {
			continue
		}
		if _, exists := seenValues[trimmedValue]; !exists {
			normalizedValues = append(normalizedValues, trimmedValue)
			seenValues[trimmedValue] = struct{}{}
		}
	}
	return normalizedValues
}
