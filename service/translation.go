package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/honeok/blog/common/filesystem"
	"github.com/honeok/blog/common/validation"
	"github.com/honeok/blog/model"
	"github.com/honeok/blog/option"
	"github.com/honeok/blog/renderer"
)

// TranslationClient 调用 OpenAI-compatible 接口生成英文 Markdown 缓存。
type TranslationClient struct {
	options    option.TranslationOptions
	httpClient *http.Client
}

// TranslatedContent 是翻译接口返回后端需要写入缓存的结构化内容。
type TranslatedContent struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

type translationChatRequest struct {
	Model       string                   `json:"model"`
	Messages    []translationChatMessage `json:"messages"`
	Temperature float64                  `json:"temperature"`
}

type translationChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type translationChatResponse struct {
	Choices []translationChatChoice `json:"choices"`
	Error   *translationChatError   `json:"error"`
}

type translationChatChoice struct {
	Message translationChatMessage `json:"message"`
}

type translationChatError struct {
	Message string `json:"message"`
}

// NewTranslationClient 创建翻译客户端，超时可以防止渲染流程被外部接口长期阻塞。
func NewTranslationClient(options option.TranslationOptions) *TranslationClient {
	return &TranslationClient{
		options: options,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// TranslatePost 请求翻译接口，并要求模型只返回 JSON，便于后续写入缓存。
func (translationClient *TranslationClient) TranslatePost(chinesePost model.Post) (TranslatedContent, error) {
	if strings.TrimSpace(translationClient.options.APIURL) == "" ||
		strings.TrimSpace(translationClient.options.APIKey) == "" ||
		strings.TrimSpace(translationClient.options.Model) == "" {
		return TranslatedContent{}, fmt.Errorf("翻译接口配置不完整")
	}

	requestPayload := translationChatRequest{
		Model: translationClient.options.Model,
		Messages: []translationChatMessage{
			{
				Role:    "system",
				Content: "你是专业技术博客译者。请把中文博客翻译成目标语言，只返回 JSON，不要使用 Markdown 代码块。",
			},
			{
				Role: "user",
				Content: "目标语言：" + translationClient.options.TargetLanguage + "\n\n请返回 JSON：{\"title\":\"...\",\"description\":\"...\",\"body\":\"...\"}。\n\n标题：\n" + chinesePost.Title +
					"\n\n摘要：\n" + chinesePost.Description +
					"\n\nMarkdown 正文：\n" + chinesePost.BodyMarkdown,
			},
		},
		Temperature: 0.2,
	}

	encodedRequestPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return TranslatedContent{}, fmt.Errorf("生成翻译请求失败：%w", err)
	}

	httpRequest, err := http.NewRequest(http.MethodPost, normalizeTranslationAPIURL(translationClient.options.APIURL), bytes.NewReader(encodedRequestPayload))
	if err != nil {
		return TranslatedContent{}, fmt.Errorf("创建翻译请求失败：%w", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+translationClient.options.APIKey)
	httpRequest.Header.Set("Content-Type", "application/json")

	httpResponse, err := translationClient.httpClient.Do(httpRequest)
	if err != nil {
		return TranslatedContent{}, fmt.Errorf("调用翻译接口失败：%w", err)
	}
	defer httpResponse.Body.Close()

	var chatResponse translationChatResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&chatResponse); err != nil {
		return TranslatedContent{}, fmt.Errorf("解析翻译响应失败：%w", err)
	}
	if chatResponse.Error != nil {
		return TranslatedContent{}, fmt.Errorf("翻译接口返回错误：%s", chatResponse.Error.Message)
	}
	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return TranslatedContent{}, fmt.Errorf("翻译接口状态异常：%s", httpResponse.Status)
	}
	if len(chatResponse.Choices) == 0 {
		return TranslatedContent{}, fmt.Errorf("翻译接口没有返回内容")
	}

	translatedJSONString := stripJSONCodeFence(chatResponse.Choices[0].Message.Content)
	var translatedContent TranslatedContent
	if err := json.Unmarshal([]byte(translatedJSONString), &translatedContent); err != nil {
		return TranslatedContent{}, fmt.Errorf("翻译结果不是合法 JSON：%w", err)
	}
	translatedContent.Title = strings.TrimSpace(translatedContent.Title)
	translatedContent.Description = strings.TrimSpace(translatedContent.Description)
	translatedContent.Body = strings.TrimSpace(translatedContent.Body)
	if translatedContent.Title == "" || translatedContent.Body == "" {
		return TranslatedContent{}, fmt.Errorf("翻译结果缺少标题或正文")
	}

	return translatedContent, nil
}

func (blogService *BlogService) prepareEnglishPosts(chinesePosts []model.Post) []model.Post {
	if !blogService.options.Translation.Enabled {
		return []model.Post{}
	}

	englishPosts := make([]model.Post, 0, len(chinesePosts))
	for _, chinesePost := range chinesePosts {
		if chinesePost.Draft || !chinesePost.Translation {
			continue
		}

		englishPost, err := blogService.loadOrGenerateEnglishPost(chinesePost)
		if err != nil {
			log.Printf("警告：英文生成失败，已保留中文渲染：%s：%v", chinesePost.SourceFileName, err)
			continue
		}
		englishPosts = append(englishPosts, englishPost)
	}
	return englishPosts
}

func (blogService *BlogService) loadOrGenerateEnglishPost(chinesePost model.Post) (model.Post, error) {
	cacheFileName := chinesePost.SourceFileName
	cacheFilePath, err := filesystem.SafeJoin(blogService.options.TranslationCacheDir, cacheFileName)
	if err != nil {
		return model.Post{}, err
	}

	cacheFileContent, readErr := os.ReadFile(cacheFilePath)
	if readErr == nil {
		cacheFrontMatter, bodyMarkdownContent, err := renderer.ParseTranslationDocument(cacheFileName, cacheFileContent)
		if err != nil {
			return model.Post{}, err
		}
		if cacheFrontMatter.Manual || cacheFrontMatter.SourceHash == chinesePost.SourceHash {
			return blogService.buildEnglishPostFromCache(cacheFilePath, cacheFrontMatter, bodyMarkdownContent, chinesePost)
		}

		translatedPost, err := blogService.generateAndWriteEnglishCache(cacheFilePath, chinesePost)
		if err != nil {
			log.Printf("警告：英文缓存已过期但刷新失败，将继续使用旧缓存：%s：%v", cacheFileName, err)
			return blogService.buildEnglishPostFromCache(cacheFilePath, cacheFrontMatter, bodyMarkdownContent, chinesePost)
		}
		return translatedPost, nil
	}
	if !os.IsNotExist(readErr) {
		return model.Post{}, fmt.Errorf("读取英文缓存失败：%w", readErr)
	}

	return blogService.generateAndWriteEnglishCache(cacheFilePath, chinesePost)
}

func (blogService *BlogService) generateAndWriteEnglishCache(cacheFilePath string, chinesePost model.Post) (model.Post, error) {
	translatedContent, err := blogService.translationClient.TranslatePost(chinesePost)
	if err != nil {
		return model.Post{}, err
	}

	cacheFrontMatter := model.TranslationFrontMatter{
		Title:       translatedContent.Title,
		Date:        chinesePost.DateText,
		Description: translatedContent.Description,
		Draft:       false,
		URL:         chinesePost.URL,
		Source:      chinesePost.SourceFileName,
		SourceHash:  chinesePost.SourceHash,
		GeneratedAt: time.Now().Format(validation.DateLayout),
		Manual:      false,
	}
	cacheFileContent, err := renderer.BuildTranslationDocument(cacheFrontMatter, translatedContent.Body)
	if err != nil {
		return model.Post{}, err
	}
	if err := filesystem.WriteFileCreatingDirectory(cacheFilePath, cacheFileContent, 0644); err != nil {
		return model.Post{}, err
	}

	return blogService.buildEnglishPostFromCache(cacheFilePath, cacheFrontMatter, translatedContent.Body, chinesePost)
}

func (blogService *BlogService) buildEnglishPostFromCache(cacheFilePath string, cacheFrontMatter model.TranslationFrontMatter, bodyMarkdownContent string, chinesePost model.Post) (model.Post, error) {
	if err := validation.ValidateRequiredPostFields(cacheFrontMatter.Title, cacheFrontMatter.Date); err != nil {
		return model.Post{}, fmt.Errorf("英文缓存 %s 校验失败：%w", cacheFilePath, err)
	}

	normalizedPermalink, err := validation.NormalizePermalinkWithFallback(cacheFrontMatter.URL, chinesePost.SourceFileName)
	if err != nil {
		return model.Post{}, fmt.Errorf("英文缓存 %s 的固定链接无效：%w", cacheFilePath, err)
	}
	publishedAt, err := validation.ParsePostDate(cacheFrontMatter.Date)
	if err != nil {
		return model.Post{}, fmt.Errorf("英文缓存 %s 的发布时间无效：%w", cacheFilePath, err)
	}
	renderedPostHTML, err := blogService.markdownRenderer.Render(bodyMarkdownContent)
	if err != nil {
		return model.Post{}, fmt.Errorf("英文缓存 %s 渲染失败：%w", cacheFilePath, err)
	}

	return model.Post{
		SourceFileName:    chinesePost.SourceFileName,
		SourceFilePath:    cacheFilePath,
		Title:             cacheFrontMatter.Title,
		DateText:          cacheFrontMatter.Date,
		PublishedAt:       publishedAt,
		Description:       cacheFrontMatter.Description,
		Draft:             cacheFrontMatter.Draft,
		URL:               normalizedPermalink,
		Comments:          false,
		Translation:       false,
		BodyMarkdown:      bodyMarkdownContent,
		BodyHTML:          renderedPostHTML,
		Language:          "en-US",
		SourceHash:        cacheFrontMatter.SourceHash,
		ManualTranslation: cacheFrontMatter.Manual,
		TranslationStatus: "已生成英文",
	}, nil
}

func (blogService *BlogService) detectTranslationStatus(sourceFileName string, frontMatter model.PostFrontMatter, draft bool, sourceHash string) string {
	if !blogService.options.Translation.Enabled {
		return "翻译关闭"
	}
	if draft {
		return "草稿不生成"
	}
	if !frontMatter.Translation {
		return "不生成英文"
	}

	cacheFilePath := filepath.Join(blogService.options.TranslationCacheDir, sourceFileName)
	cacheFileContent, err := os.ReadFile(cacheFilePath)
	if os.IsNotExist(err) {
		return "未生成"
	}
	if err != nil {
		return "读取失败"
	}

	cacheFrontMatter, _, err := renderer.ParseTranslationDocument(sourceFileName, cacheFileContent)
	if err != nil {
		return "缓存无效"
	}
	if cacheFrontMatter.Manual {
		return "手动维护"
	}
	if cacheFrontMatter.SourceHash != sourceHash {
		return "待更新"
	}
	return "已生成"
}

func normalizeTranslationAPIURL(apiURL string) string {
	trimmedAPIURL := strings.TrimRight(strings.TrimSpace(apiURL), "/")
	if strings.HasSuffix(trimmedAPIURL, "/chat/completions") {
		return trimmedAPIURL
	}
	return trimmedAPIURL + "/chat/completions"
}

func stripJSONCodeFence(content string) string {
	trimmedContent := strings.TrimSpace(content)
	trimmedContent = strings.TrimPrefix(trimmedContent, "```json")
	trimmedContent = strings.TrimPrefix(trimmedContent, "```")
	trimmedContent = strings.TrimSuffix(trimmedContent, "```")
	return strings.TrimSpace(trimmedContent)
}
