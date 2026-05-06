package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/honeok/honepress/internal/filesystem"
	"github.com/honeok/honepress/internal/model"
	"github.com/honeok/honepress/internal/renderer"
	"github.com/honeok/honepress/internal/validation"
)

// 后台文章列表
func (blogService *BlogService) ListPosts() ([]model.PostSummary, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	posts, err := blogService.scanPosts()
	if err != nil {
		return nil, err
	}
	return postsToSummaries(posts), nil
}

// 读取单篇文章
func (blogService *BlogService) GetPost(sourceFileName string) (model.PostDetail, error) {
	if err := validation.ValidateMarkdownFileName(sourceFileName); err != nil {
		return model.PostDetail{}, err
	}

	sourceFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, sourceFileName)
	if err != nil {
		return model.PostDetail{}, err
	}
	sourceMarkdownContent, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return model.PostDetail{}, fmt.Errorf("读取文章失败：%w", err)
	}

	frontMatter, bodyMarkdownContent, err := renderer.ParsePostDocument(sourceFileName, sourceMarkdownContent)
	if err != nil {
		return model.PostDetail{}, err
	}

	normalizedPermalink, err := validation.NormalizePermalinkWithFallback(frontMatter.URL, sourceFileName)
	if err != nil {
		return model.PostDetail{}, err
	}

	return model.PostDetail{
		ID:             sourceFileName,
		Title:          frontMatter.Title,
		Icon:           frontMatter.Icon,
		Date:           frontMatter.Date,
		Description:    frontMatter.Description,
		SEOTitle:       frontMatter.SEOTitle,
		SEODescription: frontMatter.SEODescription,
		Draft:          frontMatter.Draft,
		URL:            normalizedPermalink,
		Aliases:        frontMatter.Aliases,
		Tags:           frontMatter.Tags,
		Body:           bodyMarkdownContent,
	}, nil
}

// 新建文章
func (blogService *BlogService) CreatePost(savePostRequest model.SavePostRequest) (model.PostDetail, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	frontMatter, bodyMarkdownContent, err := normalizeSavePostRequest(savePostRequest)
	if err != nil {
		return model.PostDetail{}, err
	}

	sourceFileName, err := blogService.availableMarkdownFileNameForTitle(frontMatter.Title, "")
	if err != nil {
		return model.PostDetail{}, err
	}
	sourceFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, sourceFileName)
	if err != nil {
		return model.PostDetail{}, err
	}

	if err := blogService.writePostAndRenderWithRollback(sourceFilePath, nil, false, frontMatter, bodyMarkdownContent); err != nil {
		return model.PostDetail{}, err
	}
	return blogService.GetPost(sourceFileName)
}

// 更新文章
func (blogService *BlogService) UpdatePost(sourceFileName string, savePostRequest model.SavePostRequest) (model.PostDetail, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	if err := validation.ValidateMarkdownFileName(sourceFileName); err != nil {
		return model.PostDetail{}, err
	}
	frontMatter, bodyMarkdownContent, err := normalizeSavePostRequest(savePostRequest)
	if err != nil {
		return model.PostDetail{}, err
	}

	sourceFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, sourceFileName)
	if err != nil {
		return model.PostDetail{}, err
	}
	previousFileContent, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return model.PostDetail{}, fmt.Errorf("读取原文章失败：%w", err)
	}

	targetFileName, err := blogService.availableMarkdownFileNameForTitle(frontMatter.Title, sourceFileName)
	if err != nil {
		return model.PostDetail{}, err
	}
	targetFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, targetFileName)
	if err != nil {
		return model.PostDetail{}, err
	}

	if samePath(sourceFilePath, targetFilePath) {
		if err := blogService.writePostAndRenderWithRollback(sourceFilePath, previousFileContent, true, frontMatter, bodyMarkdownContent); err != nil {
			return model.PostDetail{}, err
		}
		return blogService.GetPost(sourceFileName)
	}

	if err := blogService.writeMovedPostAndRenderWithRollback(sourceFilePath, targetFilePath, previousFileContent, frontMatter, bodyMarkdownContent); err != nil {
		return model.PostDetail{}, err
	}
	return blogService.GetPost(targetFileName)
}

// 删除文章
func (blogService *BlogService) DeletePost(sourceFileName string) error {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	if err := validation.ValidateMarkdownFileName(sourceFileName); err != nil {
		return err
	}
	sourceFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, sourceFileName)
	if err != nil {
		return err
	}
	previousFileContent, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return fmt.Errorf("读取原文章失败：%w", err)
	}
	if err := os.Remove(sourceFilePath); err != nil {
		return fmt.Errorf("删除文章失败：%w", err)
	}

	if err := blogService.renderAllWithoutLock(); err != nil {
		if restoreErr := filesystem.WriteFileCreatingDirectory(sourceFilePath, previousFileContent, 0644); restoreErr != nil {
			return fmt.Errorf("重新渲染失败且恢复文章失败：%v；恢复错误：%w", err, restoreErr)
		}
		_ = blogService.renderAllWithoutLock()
		return err
	}
	return nil
}

func normalizeSavePostRequest(savePostRequest model.SavePostRequest) (model.PostFrontMatter, string, error) {
	if err := validation.ValidateRequiredPostFields(savePostRequest.Title, savePostRequest.Date); err != nil {
		return model.PostFrontMatter{}, "", err
	}
	normalizedPermalink, err := validation.NormalizePermalink(savePostRequest.URL)
	if err != nil {
		return model.PostFrontMatter{}, "", err
	}

	normalizedAliases := make([]string, 0, len(savePostRequest.Aliases))
	aliasOwners := make(map[string]struct{})
	for _, rawAlias := range savePostRequest.Aliases {
		if strings.TrimSpace(rawAlias) == "" {
			continue
		}
		normalizedAlias, err := validation.NormalizePermalink(rawAlias)
		if err != nil {
			return model.PostFrontMatter{}, "", err
		}
		if normalizedAlias == normalizedPermalink {
			return model.PostFrontMatter{}, "", fmt.Errorf("别名链接不能与固定链接相同：%s", normalizedAlias)
		}
		if _, exists := aliasOwners[normalizedAlias]; exists {
			return model.PostFrontMatter{}, "", fmt.Errorf("别名链接重复：%s", normalizedAlias)
		}
		aliasOwners[normalizedAlias] = struct{}{}
		normalizedAliases = append(normalizedAliases, normalizedAlias)
	}

	frontMatter := model.PostFrontMatter{
		Title:          strings.TrimSpace(savePostRequest.Title),
		Icon:           strings.TrimSpace(savePostRequest.Icon),
		Date:           strings.TrimSpace(savePostRequest.Date),
		Description:    strings.TrimSpace(savePostRequest.Description),
		SEOTitle:       strings.TrimSpace(savePostRequest.SEOTitle),
		SEODescription: strings.TrimSpace(savePostRequest.SEODescription),
		Draft:          savePostRequest.Draft,
		URL:            normalizedPermalink,
		Aliases:        normalizedAliases,
		Tags:           normalizeTags(savePostRequest.Tags),
	}

	return frontMatter, savePostRequest.Body, nil
}

func normalizeTags(rawTags []string) []string {
	normalizedTags := make([]string, 0, len(rawTags))
	seenTags := make(map[string]struct{})
	for _, rawTag := range rawTags {
		trimmedTag := strings.TrimSpace(rawTag)
		if trimmedTag == "" {
			continue
		}
		if _, exists := seenTags[trimmedTag]; exists {
			continue
		}
		seenTags[trimmedTag] = struct{}{}
		normalizedTags = append(normalizedTags, trimmedTag)
	}
	return normalizedTags
}

func (blogService *BlogService) availableMarkdownFileNameForTitle(title string, currentFileName string) (string, error) {
	preferredFileName, err := validation.MarkdownFileNameFromTitle(title)
	if err != nil {
		return "", err
	}
	if currentFileName != "" && preferredFileName == currentFileName {
		return preferredFileName, nil
	}

	extensionName := filepath.Ext(preferredFileName)
	fileNameStem := strings.TrimSuffix(preferredFileName, extensionName)
	candidateFileName := preferredFileName
	for suffix := 2; ; suffix++ {
		candidateFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, candidateFileName)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(candidateFilePath); os.IsNotExist(err) {
			return candidateFileName, nil
		} else if err != nil {
			return "", fmt.Errorf("检查文章文件名失败：%w", err)
		}
		if currentFileName != "" && candidateFileName == currentFileName {
			return candidateFileName, nil
		}
		candidateFileName = fmt.Sprintf("%s-%d%s", fileNameStem, suffix, extensionName)
	}
}

func (blogService *BlogService) writePostAndRenderWithRollback(sourceFilePath string, previousFileContent []byte, targetExisted bool, frontMatter model.PostFrontMatter, bodyMarkdownContent string) error {
	newFileContent, err := renderer.BuildPostDocument(frontMatter, bodyMarkdownContent)
	if err != nil {
		return err
	}
	if err := filesystem.WriteFileCreatingDirectory(sourceFilePath, newFileContent, 0644); err != nil {
		return err
	}

	if err := blogService.renderAllWithoutLock(); err != nil {
		if targetExisted {
			if restoreErr := filesystem.WriteFileCreatingDirectory(sourceFilePath, previousFileContent, 0644); restoreErr != nil {
				return fmt.Errorf("重新渲染失败且恢复文章失败：%v；恢复错误：%w", err, restoreErr)
			}
		} else if removeErr := os.Remove(sourceFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("重新渲染失败且删除新文章失败：%v；删除错误：%w", err, removeErr)
		}
		_ = blogService.renderAllWithoutLock()
		return err
	}

	return nil
}

func (blogService *BlogService) writeMovedPostAndRenderWithRollback(sourceFilePath string, targetFilePath string, previousFileContent []byte, frontMatter model.PostFrontMatter, bodyMarkdownContent string) error {
	newFileContent, err := renderer.BuildPostDocument(frontMatter, bodyMarkdownContent)
	if err != nil {
		return err
	}
	if err := filesystem.WriteFileCreatingDirectory(targetFilePath, newFileContent, 0644); err != nil {
		return err
	}
	if err := os.Remove(sourceFilePath); err != nil {
		_ = os.Remove(targetFilePath)
		return fmt.Errorf("重命名文章文件失败：%w", err)
	}

	if err := blogService.renderAllWithoutLock(); err != nil {
		if restoreErr := filesystem.WriteFileCreatingDirectory(sourceFilePath, previousFileContent, 0644); restoreErr != nil {
			return fmt.Errorf("重新渲染失败且恢复原文章失败：%v；恢复错误：%w", err, restoreErr)
		}
		if removeErr := os.Remove(targetFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("重新渲染失败且删除新文章失败：%v；删除错误：%w", err, removeErr)
		}
		_ = blogService.renderAllWithoutLock()
		return err
	}
	return nil
}

func samePath(firstPath string, secondPath string) bool {
	firstAbsPath, firstErr := filepath.Abs(firstPath)
	secondAbsPath, secondErr := filepath.Abs(secondPath)
	if firstErr == nil && secondErr == nil {
		return strings.EqualFold(firstAbsPath, secondAbsPath)
	}
	return strings.EqualFold(filepath.Clean(firstPath), filepath.Clean(secondPath))
}
