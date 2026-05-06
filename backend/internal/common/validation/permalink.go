package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/honeok/honepress/backend/internal/constant"
)

var publicHTMLFileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*\.html$`)

// 把后台输入的固定链接归一成最终输出文件名
func NormalizePermalink(rawPermalink string) (string, error) {
	trimmedPermalink := strings.TrimSpace(rawPermalink)
	if trimmedPermalink == "" {
		return "", fmt.Errorf("固定链接不能为空")
	}

	trimmedPermalink = strings.TrimPrefix(trimmedPermalink, "/")
	if strings.Contains(trimmedPermalink, "/") || strings.Contains(trimmedPermalink, `\`) {
		return "", fmt.Errorf("固定链接不能包含斜杠：%s", rawPermalink)
	}
	if strings.Contains(trimmedPermalink, "..") {
		return "", fmt.Errorf("固定链接不能包含路径穿越：%s", rawPermalink)
	}
	if strings.ContainsAny(trimmedPermalink, " \t\r\n") {
		return "", fmt.Errorf("固定链接不能包含空格：%s", rawPermalink)
	}
	if !strings.HasSuffix(trimmedPermalink, ".html") {
		trimmedPermalink += ".html"
	}
	if !publicHTMLFileNamePattern.MatchString(trimmedPermalink) {
		return "", fmt.Errorf("固定链接只能使用 ASCII 字母、数字、短横线和下划线：%s", rawPermalink)
	}
	if _, isReservedFileName := constant.ReservedPublicFileNames[trimmedPermalink]; isReservedFileName {
		return "", fmt.Errorf("固定链接不能使用保留文件名：%s", trimmedPermalink)
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

// 校验 Markdown 文件名
func ValidateMarkdownFileName(markdownFileName string) error {
	trimmedFileName := strings.TrimSpace(markdownFileName)
	if trimmedFileName == "" {
		return fmt.Errorf("文章文件名不能为空")
	}
	if filepath.Base(trimmedFileName) != trimmedFileName {
		return fmt.Errorf("文章文件名不能包含路径：%s", markdownFileName)
	}
	if !strings.EqualFold(filepath.Ext(trimmedFileName), ".md") {
		return fmt.Errorf("文章文件名必须以 .md 结尾：%s", markdownFileName)
	}
	fileNameStem := strings.TrimSuffix(trimmedFileName, filepath.Ext(trimmedFileName))
	if strings.Trim(fileNameStem, " .") == "" {
		return fmt.Errorf("文章文件名不能为空：%s", markdownFileName)
	}
	if strings.Contains(fileNameStem, "..") {
		return fmt.Errorf("文章文件名不能包含路径穿越：%s", markdownFileName)
	}
	for _, currentRune := range trimmedFileName {
		if isInvalidMarkdownFileRune(currentRune) {
			return fmt.Errorf("文章文件名包含非法字符：%s", markdownFileName)
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
		return "", fmt.Errorf("标题不能为空")
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
		return "", fmt.Errorf("标题不能只包含文件名非法字符")
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
