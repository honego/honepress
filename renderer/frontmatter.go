package renderer

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/honeok/honepress/model"
)

type postFrontMatterYAML struct {
	Title       string   `yaml:"title"`
	Icon        string   `yaml:"icon"`
	Date        string   `yaml:"date"`
	Description string   `yaml:"description"`
	Draft       bool     `yaml:"draft"`
	URL         string   `yaml:"url"`
	Comments    *bool    `yaml:"comments"`
	Aliases     []string `yaml:"aliases"`
	Tags        []string `yaml:"tags"`
}

// 解析 Markdown 文件并剥离 Front Matter
func ParsePostDocument(sourceFileName string, markdownContent []byte) (model.PostFrontMatter, string, error) {
	frontMatterContent, bodyMarkdownContent, hasFrontMatter := splitFrontMatter(markdownContent)
	if !hasFrontMatter {
		return model.PostFrontMatter{}, "", fmt.Errorf("Front Matter 缺失：%s", sourceFileName)
	}

	decodedFrontMatter := postFrontMatterYAML{
		Comments: boolPointer(true),
	}
	if err := yaml.Unmarshal([]byte(frontMatterContent), &decodedFrontMatter); err != nil {
		return model.PostFrontMatter{}, "", fmt.Errorf("解析 Front Matter 失败：%s：%w", sourceFileName, err)
	}

	parsedFrontMatter := model.PostFrontMatter{
		Title:       strings.TrimSpace(decodedFrontMatter.Title),
		Icon:        strings.TrimSpace(decodedFrontMatter.Icon),
		Date:        strings.TrimSpace(decodedFrontMatter.Date),
		Description: strings.TrimSpace(decodedFrontMatter.Description),
		Draft:       decodedFrontMatter.Draft,
		URL:         strings.TrimSpace(decodedFrontMatter.URL),
		Comments:    true,
		Aliases:     normalizeStringList(decodedFrontMatter.Aliases),
		Tags:        normalizeStringList(decodedFrontMatter.Tags),
	}
	if decodedFrontMatter.Comments != nil {
		parsedFrontMatter.Comments = *decodedFrontMatter.Comments
	}

	return parsedFrontMatter, bodyMarkdownContent, nil
}

// 生成 Markdown 文件内容
func BuildPostDocument(frontMatter model.PostFrontMatter, bodyMarkdownContent string) ([]byte, error) {
	encodedFrontMatter, err := yaml.Marshal(postFrontMatterYAML{
		Title:       frontMatter.Title,
		Icon:        frontMatter.Icon,
		Date:        frontMatter.Date,
		Description: frontMatter.Description,
		Draft:       frontMatter.Draft,
		URL:         frontMatter.URL,
		Comments:    boolPointer(frontMatter.Comments),
		Aliases:     frontMatter.Aliases,
		Tags:        frontMatter.Tags,
	})
	if err != nil {
		return nil, fmt.Errorf("生成 Front Matter 失败：%w", err)
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

func boolPointer(booleanValue bool) *bool {
	return &booleanValue
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
