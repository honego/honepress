package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/honeok/blog/common/filesystem"
	"github.com/honeok/blog/common/validation"
	"github.com/honeok/blog/model"
	"github.com/honeok/blog/renderer"
)

// ListPosts 返回后台文章列表，包含草稿和翻译状态。
func (blogService *BlogService) ListPosts() ([]model.PostSummary, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	chinesePosts, err := blogService.scanChinesePosts()
	if err != nil {
		return nil, err
	}
	return postsToSummaries(chinesePosts, "", map[string]string{}), nil
}

// GetPost 读取单篇 Markdown 源文件，返回后台编辑器需要的字段。
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
		Comments:    frontMatter.Comments,
		Translation: frontMatter.Translation,
		Body:        bodyMarkdownContent,
	}, nil
}

// CreatePost 新建 Markdown 源文件，随后立即重新生成静态站点。
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
	} else if err != nil && !os.IsNotExist(err) {
		return model.PostDetail{}, fmt.Errorf("检查文章是否存在失败：%w", err)
	}

	if err := blogService.writePostAndRenderWithRollback(sourceFilePath, nil, false, frontMatter, bodyMarkdownContent); err != nil {
		return model.PostDetail{}, err
	}
	return blogService.GetPost(sourceFileName)
}

// UpdatePost 更新已有 Markdown 源文件，失败时会恢复原文件内容。
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

// DeletePost 删除 Markdown 源文件，随后重新生成静态站点。
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

// RegenerateEnglishPost 手动刷新英文缓存，manual: true 的缓存不会被覆盖。
func (blogService *BlogService) RegenerateEnglishPost(sourceFileName string) error {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	if !blogService.options.Translation.Enabled {
		return fmt.Errorf("英文翻译未开启")
	}
	if err := validation.ValidateMarkdownFileName(sourceFileName); err != nil {
		return err
	}

	chinesePosts, err := blogService.scanChinesePosts()
	if err != nil {
		return err
	}
	for _, chinesePost := range chinesePosts {
		if chinesePost.SourceFileName != sourceFileName {
			continue
		}
		if chinesePost.Draft {
			return fmt.Errorf("草稿文章不生成英文")
		}
		if !chinesePost.Translation {
			return fmt.Errorf("当前文章已关闭英文生成")
		}

		cacheFilePath := filepath.Join(blogService.options.TranslationCacheDir, sourceFileName)
		cacheFileContent, readErr := os.ReadFile(cacheFilePath)
		if readErr == nil {
			cacheFrontMatter, _, err := renderer.ParseTranslationDocument(sourceFileName, cacheFileContent)
			if err != nil {
				return err
			}
			if cacheFrontMatter.Manual {
				return fmt.Errorf("英文缓存 manual: true，不能自动覆盖")
			}
		} else if readErr != nil && !os.IsNotExist(readErr) {
			return fmt.Errorf("读取英文缓存失败：%w", readErr)
		}

		if _, err := blogService.generateAndWriteEnglishCache(cacheFilePath, chinesePost); err != nil {
			return err
		}
		return blogService.renderAllWithoutLock()
	}

	return fmt.Errorf("文章不存在：%s", sourceFileName)
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
		Translation: savePostRequest.Translation,
		Aliases:     normalizedAliases,
	}

	return frontMatter, savePostRequest.Body, nil
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
