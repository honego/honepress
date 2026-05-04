package service

import (
	"fmt"
	"os"
	"strings"

	"github.com/honeok/blog/common/filesystem"
	"github.com/honeok/blog/common/validation"
	"github.com/honeok/blog/model"
	"github.com/honeok/blog/renderer"
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
		ID:          sourceFileName,
		Title:       frontMatter.Title,
		Date:        frontMatter.Date,
		Description: frontMatter.Description,
		Draft:       frontMatter.Draft,
		URL:         normalizedPermalink,
		Aliases:     frontMatter.Aliases,
		Tags:        frontMatter.Tags,
		Comments:    frontMatter.Comments,
		Body:        bodyMarkdownContent,
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

	sourceFileName := strings.TrimSpace(savePostRequest.ID)
	if sourceFileName == "" {
		sourceFileName, err = validation.MarkdownFileNameFromPermalink(frontMatter.URL)
		if err != nil {
			return model.PostDetail{}, err
		}
	}
	if err := validation.ValidateMarkdownFileName(sourceFileName); err != nil {
		return model.PostDetail{}, err
	}

	sourceFilePath, err := filesystem.SafeJoin(blogService.options.PostsDir, sourceFileName)
	if err != nil {
		return model.PostDetail{}, err
	}
	if _, err := os.Stat(sourceFilePath); err == nil {
		return model.PostDetail{}, fmt.Errorf("文章已存在：%s", sourceFileName)
	} else if !os.IsNotExist(err) {
		return model.PostDetail{}, fmt.Errorf("检查文章是否存在失败：%w", err)
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

	if err := blogService.writePostAndRenderWithRollback(sourceFilePath, previousFileContent, true, frontMatter, bodyMarkdownContent); err != nil {
		return model.PostDetail{}, err
	}
	return blogService.GetPost(sourceFileName)
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
		Title:       strings.TrimSpace(savePostRequest.Title),
		Date:        strings.TrimSpace(savePostRequest.Date),
		Description: strings.TrimSpace(savePostRequest.Description),
		Draft:       savePostRequest.Draft,
		URL:         normalizedPermalink,
		Comments:    savePostRequest.Comments,
		Aliases:     normalizedAliases,
		Tags:        normalizeTags(savePostRequest.Tags),
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
