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

func (blogService *BlogService) ListPublicPosts() ([]model.PostSummary, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	posts, err := blogService.scanPosts()
	if err != nil {
		return nil, err
	}
	return postsToSummaries(filterPublishedPosts(posts)), nil
}

func (blogService *BlogService) GetPublicPost(postID string) (model.PublicPostDetail, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	normalizedPostID := strings.Trim(strings.TrimSpace(postID), "/")
	posts, err := blogService.scanPosts()
	if err != nil {
		return model.PublicPostDetail{}, err
	}
	for _, currentPost := range posts {
		if currentPost.Draft {
			continue
		}
		if currentPost.SourceFileName != normalizedPostID &&
			currentPost.SourceURL != normalizedPostID &&
			currentPost.Slug != normalizedPostID &&
			currentPost.PostID != normalizedPostID &&
			strings.Trim(currentPost.URL, "/") != normalizedPostID {
			continue
		}
		renderedPostHTML, err := blogService.markdownRenderer.Render(currentPost.BodyMarkdown)
		if err != nil {
			return model.PublicPostDetail{}, fmt.Errorf("render post at %s: %w", currentPost.SourceFilePath, err)
		}
		return model.PublicPostDetail{
			ID:             currentPost.SourceFileName,
			Title:          currentPost.Title,
			Icon:           currentPost.Icon,
			Thumbnail:      currentPost.Thumbnail,
			Date:           currentPost.DateText,
			Description:    currentPost.Description,
			SEOTitle:       currentPost.SEOTitle,
			SEODescription: currentPost.SEODescription,
			URL:            currentPost.URL,
			PublicURL:      publicPath(currentPost.URL),
			Tags:           currentPost.Tags,
			HTML:           string(renderedPostHTML),
		}, nil
	}
	return model.PublicPostDetail{}, fmt.Errorf("post not found: %s", normalizedPostID)
}

// 读取单篇文章
func (blogService *BlogService) GetPost(sourceFileName string) (model.PostDetail, error) {
	if err := validation.ValidateMarkdownFileName(sourceFileName); err != nil {
		return model.PostDetail{}, err
	}

	posts, err := blogService.scanPosts()
	if err != nil {
		return model.PostDetail{}, err
	}
	for _, currentPost := range posts {
		if currentPost.SourceFileName != sourceFileName {
			continue
		}
		return model.PostDetail{
			ID:             currentPost.SourceFileName,
			Title:          currentPost.Title,
			Icon:           currentPost.Icon,
			Thumbnail:      currentPost.Thumbnail,
			Date:           currentPost.DateText,
			Description:    currentPost.Description,
			SEOTitle:       currentPost.SEOTitle,
			SEODescription: currentPost.SEODescription,
			Draft:          currentPost.Draft,
			URL:            currentPost.URL,
			Tags:           currentPost.Tags,
			Body:           currentPost.BodyMarkdown,
		}, nil
	}
	return model.PostDetail{}, fmt.Errorf("post not found: %s", sourceFileName)
}

// 新建文章
func (blogService *BlogService) CreatePost(savePostRequest model.SavePostRequest) (model.PostDetail, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	frontMatter, bodyMarkdownContent, err := normalizeSavePostRequest(savePostRequest)
	if err != nil {
		return model.PostDetail{}, err
	}
	nextPostURL, err := blogService.nextAvailablePostURL()
	if err != nil {
		return model.PostDetail{}, err
	}
	frontMatter.URL = nextPostURL

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
		return model.PostDetail{}, fmt.Errorf("read existing post: %w", err)
	}
	if strings.TrimSpace(frontMatter.URL) == "" {
		currentPostURL, err := blogService.postURLForSource(sourceFileName)
		if err != nil {
			return model.PostDetail{}, err
		}
		frontMatter.URL = currentPostURL
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
		return fmt.Errorf("read existing post: %w", err)
	}
	if err := os.Remove(sourceFilePath); err != nil {
		return fmt.Errorf("delete post: %w", err)
	}

	if err := blogService.renderAllWithoutLock(); err != nil {
		if restoreErr := filesystem.WriteFileCreatingDirectory(sourceFilePath, previousFileContent, 0644); restoreErr != nil {
			return fmt.Errorf("render failed and restore post failed: %v; restore error: %w", err, restoreErr)
		}
		_ = blogService.renderAllWithoutLock()
		return err
	}
	return nil
}

func (blogService *BlogService) nextAvailablePostURL() (string, error) {
	posts, err := blogService.scanPosts()
	if err != nil {
		return "", err
	}
	usedPostIDs := make(map[int]string)
	for _, currentPost := range posts {
		if postID, hasPostID := postIDFromPermalink(currentPost.URL); hasPostID {
			usedPostIDs[postID] = currentPost.SourceFilePath
		}
	}
	return postPublicURL(nextSequentialPostID(usedPostIDs)), nil
}

func (blogService *BlogService) postURLForSource(sourceFileName string) (string, error) {
	posts, err := blogService.scanPosts()
	if err != nil {
		return "", err
	}
	for _, currentPost := range posts {
		if currentPost.SourceFileName == sourceFileName {
			return currentPost.URL, nil
		}
	}
	return "", fmt.Errorf("post not found: %s", sourceFileName)
}

func normalizeSavePostRequest(savePostRequest model.SavePostRequest) (model.PostFrontMatter, string, error) {
	if err := validation.ValidateRequiredPostFields(savePostRequest.Title, savePostRequest.Date); err != nil {
		return model.PostFrontMatter{}, "", err
	}
	normalizedPermalink := strings.TrimSpace(savePostRequest.URL)
	if normalizedPermalink != "" {
		var err error
		normalizedPermalink, err = validation.NormalizePermalink(normalizedPermalink)
		if err != nil {
			return model.PostFrontMatter{}, "", err
		}
	}

	frontMatter := model.PostFrontMatter{
		Title:          strings.TrimSpace(savePostRequest.Title),
		Icon:           strings.TrimSpace(savePostRequest.Icon),
		Thumbnail:      strings.TrimSpace(savePostRequest.Thumbnail),
		Date:           strings.TrimSpace(savePostRequest.Date),
		Description:    strings.TrimSpace(savePostRequest.Description),
		SEOTitle:       strings.TrimSpace(savePostRequest.SEOTitle),
		SEODescription: strings.TrimSpace(savePostRequest.SEODescription),
		Draft:          savePostRequest.Draft,
		URL:            normalizedPermalink,
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
			return "", fmt.Errorf("check post file name: %w", err)
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
				return fmt.Errorf("render failed and restore post failed: %v; restore error: %w", err, restoreErr)
			}
		} else if removeErr := os.Remove(sourceFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("render failed and remove new post failed: %v; remove error: %w", err, removeErr)
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
		return fmt.Errorf("rename post file: %w", err)
	}

	if err := blogService.renderAllWithoutLock(); err != nil {
		if restoreErr := filesystem.WriteFileCreatingDirectory(sourceFilePath, previousFileContent, 0644); restoreErr != nil {
			return fmt.Errorf("render failed and restore original post failed: %v; restore error: %w", err, restoreErr)
		}
		if removeErr := os.Remove(targetFilePath); removeErr != nil && !os.IsNotExist(removeErr) {
			return fmt.Errorf("render failed and remove new post failed: %v; remove error: %w", err, removeErr)
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
