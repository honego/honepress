package renderer

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/honeok/blog/model"
)

type postFrontMatterYAML struct {
	Title       string   `yaml:"title"`
	Date        string   `yaml:"date"`
	Description string   `yaml:"description"`
	Draft       bool     `yaml:"draft"`
	URL         string   `yaml:"url"`
	Comments    *bool    `yaml:"comments"`
	Translation *bool    `yaml:"translation"`
	Aliases     []string `yaml:"aliases"`
}

// ParsePostDocument 解析 Markdown 文件并剥离 Front Matter，避免元数据泄露到正文 HTML。
func ParsePostDocument(sourceFileName string, markdownContent []byte) (model.PostFrontMatter, string, error) {
	frontMatterContent, bodyMarkdownContent, hasFrontMatter := splitFrontMatter(markdownContent)
	if !hasFrontMatter {
		return model.PostFrontMatter{}, "", fmt.Errorf("Front Matter 缺失：%s", sourceFileName)
	}

	decodedFrontMatter := postFrontMatterYAML{
		Comments:    boolPointer(true),
		Translation: boolPointer(true),
	}
	if err := yaml.Unmarshal([]byte(frontMatterContent), &decodedFrontMatter); err != nil {
		return model.PostFrontMatter{}, "", fmt.Errorf("解析 Front Matter 失败：%s：%w", sourceFileName, err)
	}

	parsedFrontMatter := model.PostFrontMatter{
		Title:       strings.TrimSpace(decodedFrontMatter.Title),
		Date:        strings.TrimSpace(decodedFrontMatter.Date),
		Description: strings.TrimSpace(decodedFrontMatter.Description),
		Draft:       decodedFrontMatter.Draft,
		URL:         strings.TrimSpace(decodedFrontMatter.URL),
		Comments:    true,
		Translation: true,
		Aliases:     normalizeAliases(decodedFrontMatter.Aliases),
	}
	if decodedFrontMatter.Comments != nil {
		parsedFrontMatter.Comments = *decodedFrontMatter.Comments
	}
	if decodedFrontMatter.Translation != nil {
		parsedFrontMatter.Translation = *decodedFrontMatter.Translation
	}

	return parsedFrontMatter, bodyMarkdownContent, nil
}

// ParseTranslationDocument 解析英文缓存文件，manual 字段会在翻译刷新时保护人工内容。
func ParseTranslationDocument(sourceFileName string, markdownContent []byte) (model.TranslationFrontMatter, string, error) {
	frontMatterContent, bodyMarkdownContent, hasFrontMatter := splitFrontMatter(markdownContent)
	if !hasFrontMatter {
		return model.TranslationFrontMatter{}, "", fmt.Errorf("英文缓存 Front Matter 缺失：%s", sourceFileName)
	}

	var parsedFrontMatter model.TranslationFrontMatter
	if err := yaml.Unmarshal([]byte(frontMatterContent), &parsedFrontMatter); err != nil {
		return model.TranslationFrontMatter{}, "", fmt.Errorf("解析英文缓存 Front Matter 失败：%s：%w", sourceFileName, err)
	}

	parsedFrontMatter.Title = strings.TrimSpace(parsedFrontMatter.Title)
	parsedFrontMatter.Date = strings.TrimSpace(parsedFrontMatter.Date)
	parsedFrontMatter.Description = strings.TrimSpace(parsedFrontMatter.Description)
	parsedFrontMatter.URL = strings.TrimSpace(parsedFrontMatter.URL)
	parsedFrontMatter.Source = strings.TrimSpace(parsedFrontMatter.Source)
	parsedFrontMatter.SourceHash = strings.TrimSpace(parsedFrontMatter.SourceHash)
	parsedFrontMatter.GeneratedAt = strings.TrimSpace(parsedFrontMatter.GeneratedAt)

	return parsedFrontMatter, bodyMarkdownContent, nil
}

// BuildPostDocument 根据后台表单重新生成 Markdown 文件，统一 Front Matter 的字段顺序。
func BuildPostDocument(frontMatter model.PostFrontMatter, bodyMarkdownContent string) ([]byte, error) {
	encodedFrontMatter, err := yaml.Marshal(postFrontMatterYAML{
		Title:       frontMatter.Title,
		Date:        frontMatter.Date,
		Description: frontMatter.Description,
		Draft:       frontMatter.Draft,
		URL:         frontMatter.URL,
		Comments:    boolPointer(frontMatter.Comments),
		Translation: boolPointer(frontMatter.Translation),
		Aliases:     frontMatter.Aliases,
	})
	if err != nil {
		return nil, fmt.Errorf("生成 Front Matter 失败：%w", err)
	}

	normalizedBodyMarkdownContent := strings.TrimLeft(bodyMarkdownContent, "\r\n")
	return []byte("---\n" + string(encodedFrontMatter) + "---\n\n" + normalizedBodyMarkdownContent), nil
}

// BuildTranslationDocument 写入英文缓存时保留来源哈希，避免没有变化的文章重复请求翻译接口。
func BuildTranslationDocument(frontMatter model.TranslationFrontMatter, bodyMarkdownContent string) ([]byte, error) {
	encodedFrontMatter, err := yaml.Marshal(frontMatter)
	if err != nil {
		return nil, fmt.Errorf("生成英文缓存 Front Matter 失败：%w", err)
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

func normalizeAliases(rawAliases []string) []string {
	normalizedAliases := make([]string, 0, len(rawAliases))
	for _, rawAlias := range rawAliases {
		trimmedAlias := strings.TrimSpace(rawAlias)
		if trimmedAlias != "" {
			normalizedAliases = append(normalizedAliases, trimmedAlias)
		}
	}
	return normalizedAliases
}
