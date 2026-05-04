package validation

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/honeok/blog/constant"
)

var (
	publicHTMLFileNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*\.html$`)
	markdownFileNamePattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]*\.md$`)
)

// NormalizePermalink 把后台输入的固定链接归一成最终输出文件名。
//
// 固定链接只接受单层英文文件名，是为了避免标题变化、路径穿越和多语言目录互相影响。
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
		return "", fmt.Errorf("固定链接只能使用英文、数字、短横线和下划线：%s", rawPermalink)
	}
	if _, isReservedFileName := constant.ReservedPublicFileNames[trimmedPermalink]; isReservedFileName {
		return "", fmt.Errorf("固定链接不能使用保留文件名：%s", trimmedPermalink)
	}

	return trimmedPermalink, nil
}

// NormalizePermalinkWithFallback 在 Front Matter 没有 url 字段时使用 Markdown 文件名兜底。
func NormalizePermalinkWithFallback(rawPermalink string, sourceFileName string) (string, error) {
	if strings.TrimSpace(rawPermalink) != "" {
		return NormalizePermalink(rawPermalink)
	}

	fileExtensionName := filepath.Ext(sourceFileName)
	fallbackPermalink := strings.TrimSuffix(sourceFileName, fileExtensionName) + ".html"
	return NormalizePermalink(fallbackPermalink)
}

// ValidateMarkdownFileName 校验后台文章 ID，文章 ID 始终等同于 Markdown 文件名。
func ValidateMarkdownFileName(markdownFileName string) error {
	if strings.TrimSpace(markdownFileName) == "" {
		return fmt.Errorf("文章文件名不能为空")
	}
	if filepath.Base(markdownFileName) != markdownFileName {
		return fmt.Errorf("文章文件名不能包含路径：%s", markdownFileName)
	}
	if !markdownFileNamePattern.MatchString(markdownFileName) {
		return fmt.Errorf("文章文件名只能使用英文、数字、短横线和下划线：%s", markdownFileName)
	}
	return nil
}

// MarkdownFileNameFromPermalink 默认用固定链接派生 Markdown 文件名，保证后台新建文章时链接稳定。
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
