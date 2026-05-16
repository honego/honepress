package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/honeok/honepress/internal/core"
)

var (
	publicHTMLFileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*\.html$`)
	publicSlugPattern         = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*$`)
)

const DefaultPermalinkStructure = "/?p=%post_id%"

var allowedPermalinkTags = map[string]struct{}{
	"%year%":     {},
	"%monthnum%": {},
	"%day%":      {},
	"%hour%":     {},
	"%minute%":   {},
	"%second%":   {},
	"%post_id%":  {},
	"%postname%": {},
	"%category%": {},
}

// 把后台输入的固定链接归一成最终输出文件名
func NormalizePermalink(rawPermalink string) (string, error) {
	trimmedPermalink := strings.TrimSpace(rawPermalink)
	if trimmedPermalink == "" {
		return "", fmt.Errorf("permalink is empty")
	}

	trimmedPermalink = strings.TrimPrefix(trimmedPermalink, "/")
	if strings.Contains(trimmedPermalink, "/") || strings.Contains(trimmedPermalink, `\`) {
		return "", fmt.Errorf("permalink must not contain path separators: %s", rawPermalink)
	}
	if strings.Contains(trimmedPermalink, "..") {
		return "", fmt.Errorf("permalink must not contain path traversal: %s", rawPermalink)
	}
	if strings.ContainsAny(trimmedPermalink, " \t\r\n") {
		return "", fmt.Errorf("permalink must not contain whitespace: %s", rawPermalink)
	}
	if !strings.HasSuffix(trimmedPermalink, ".html") {
		trimmedPermalink += ".html"
	}
	if !publicHTMLFileNamePattern.MatchString(trimmedPermalink) {
		return "", fmt.Errorf("permalink must use ASCII letters, digits, hyphen, or underscore: %s", rawPermalink)
	}
	if _, isReservedFileName := core.ReservedPublicFileNames[trimmedPermalink]; isReservedFileName {
		return "", fmt.Errorf("permalink uses reserved public file name: %s", trimmedPermalink)
	}

	return trimmedPermalink, nil
}

// 使用文件名兜底归一固定链接
func NormalizePermalinkWithFallback(rawPermalink string, sourceFileName string) (string, error) {
	if strings.TrimSpace(rawPermalink) != "" {
		return NormalizePermalink(rawPermalink)
	}

	fileExtensionName := filepath.Ext(sourceFileName)
	fallbackPermalink := strings.TrimSuffix(sourceFileName, fileExtensionName) + ".html"
	return NormalizePermalink(fallbackPermalink)
}

func NormalizePostSlug(rawSlug string) (string, error) {
	trimmedSlug := strings.TrimSpace(rawSlug)
	trimmedSlug = strings.TrimPrefix(trimmedSlug, "/")
	trimmedSlug = strings.TrimSuffix(trimmedSlug, ".html")
	if trimmedSlug == "" {
		return "", fmt.Errorf("post slug is empty")
	}
	if strings.Contains(trimmedSlug, "/") || strings.Contains(trimmedSlug, `\`) {
		return "", fmt.Errorf("post slug must not contain path separators: %s", rawSlug)
	}
	if strings.Contains(trimmedSlug, "..") {
		return "", fmt.Errorf("post slug must not contain path traversal: %s", rawSlug)
	}
	if strings.ContainsAny(trimmedSlug, " \t\r\n") {
		return "", fmt.Errorf("post slug must not contain whitespace: %s", rawSlug)
	}
	if !publicSlugPattern.MatchString(trimmedSlug) {
		return "", fmt.Errorf("post slug must use ASCII letters, digits, hyphen, or underscore: %s", rawSlug)
	}
	if _, isReservedFileName := core.ReservedPublicFileNames[trimmedSlug+".html"]; isReservedFileName {
		return "", fmt.Errorf("post slug uses reserved public file name: %s", trimmedSlug)
	}
	return trimmedSlug, nil
}

func NormalizePostSlugWithFallback(rawPermalink string, sourceFileName string) (string, error) {
	if strings.TrimSpace(rawPermalink) != "" {
		return NormalizePostSlug(rawPermalink)
	}

	fileExtensionName := filepath.Ext(sourceFileName)
	return NormalizePostSlug(strings.TrimSuffix(sourceFileName, fileExtensionName))
}

func NormalizePermalinkStructure(rawStructure string) string {
	trimmedStructure := strings.TrimSpace(rawStructure)
	if trimmedStructure == "" {
		return DefaultPermalinkStructure
	}
	switch trimmedStructure {
	case "plain":
		return "/?p=%post_id%"
	case "date":
		return "/%year%/%monthnum%/%day%/%postname%/"
	case "month":
		return "/%year%/%monthnum%/%postname%/"
	case "numeric":
		return "/archives/%post_id%"
	case "postname":
		return "/%postname%/"
	}
	if !strings.HasPrefix(trimmedStructure, "/") {
		trimmedStructure = "/" + trimmedStructure
	}
	return trimmedStructure
}

func ValidatePermalinkStructure(rawStructure string) error {
	structure := NormalizePermalinkStructure(rawStructure)
	if strings.Contains(structure, `\`) || strings.Contains(structure, "..") {
		return fmt.Errorf("permalink structure must not contain path traversal")
	}
	if strings.ContainsAny(structure, " \t\r\n") {
		return fmt.Errorf("permalink structure must not contain whitespace")
	}
	if !strings.Contains(structure, "%post_id%") && !strings.Contains(structure, "%postname%") {
		return fmt.Errorf("permalink structure must contain %%post_id%% or %%postname%%")
	}
	if strings.Contains(structure, "?") && structure != "/?p=%post_id%" {
		return fmt.Errorf("query permalink structure must be /?p=%%post_id%%")
	}

	for _, match := range regexp.MustCompile(`%[A-Za-z0-9_]+%`).FindAllString(structure, -1) {
		if _, exists := allowedPermalinkTags[match]; !exists {
			return fmt.Errorf("unsupported permalink tag: %s", match)
		}
	}
	return nil
}

// 校验 Markdown 文件名
func ValidateMarkdownFileName(markdownFileName string) error {
	trimmedFileName := strings.TrimSpace(markdownFileName)
	if trimmedFileName == "" {
		return fmt.Errorf("markdown file name is empty")
	}
	if filepath.Base(trimmedFileName) != trimmedFileName {
		return fmt.Errorf("markdown file name must not contain a path: %s", markdownFileName)
	}
	if !strings.EqualFold(filepath.Ext(trimmedFileName), ".md") {
		return fmt.Errorf("markdown file name must end with .md: %s", markdownFileName)
	}
	fileNameStem := strings.TrimSuffix(trimmedFileName, filepath.Ext(trimmedFileName))
	if strings.Trim(fileNameStem, " .") == "" {
		return fmt.Errorf("markdown file name is empty: %s", markdownFileName)
	}
	if strings.Contains(fileNameStem, "..") {
		return fmt.Errorf("markdown file name must not contain path traversal: %s", markdownFileName)
	}
	for _, currentRune := range trimmedFileName {
		if isInvalidMarkdownFileRune(currentRune) {
			return fmt.Errorf("markdown file name contains invalid characters: %s", markdownFileName)
		}
	}
	return nil
}

// 根据固定链接生成 Markdown 文件名
func MarkdownFileNameFromPermalink(normalizedPermalink string) (string, error) {
	if _, err := NormalizePermalink(normalizedPermalink); err != nil {
		return "", err
	}

	markdownFileName := strings.TrimSuffix(normalizedPermalink, ".html") + ".md"
	if err := ValidateMarkdownFileName(markdownFileName); err != nil {
		return "", err
	}
	return markdownFileName, nil
}

// 根据标题生成 Markdown 文件名
func MarkdownFileNameFromTitle(title string) (string, error) {
	fileNameStem := strings.Join(strings.Fields(strings.TrimSpace(title)), " ")
	if fileNameStem == "" {
		return "", fmt.Errorf("title is empty")
	}

	var safeFileNameBuilder strings.Builder
	for _, currentRune := range fileNameStem {
		if isInvalidMarkdownFileRune(currentRune) {
			continue
		}
		safeFileNameBuilder.WriteRune(currentRune)
	}

	fileNameStem = strings.Trim(safeFileNameBuilder.String(), " .")
	if fileNameStem == "" {
		return "", fmt.Errorf("title contains only invalid file name characters")
	}

	markdownFileName := fileNameStem + ".md"
	if err := ValidateMarkdownFileName(markdownFileName); err != nil {
		return "", err
	}
	return markdownFileName, nil
}

func isInvalidMarkdownFileRune(currentRune rune) bool {
	return unicode.IsControl(currentRune) || strings.ContainsRune(`<>:"/\|?*`, currentRune)
}
