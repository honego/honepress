package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/honeok/blog/common/filesystem"
	"github.com/honeok/blog/model"
	"github.com/honeok/blog/option"
)

var allowedIconMIMETypes = map[string]string{
	"image/vnd.microsoft.icon": ".ico",
	"image/x-icon":             ".ico",
	"image/png":                ".png",
	"image/jpeg":               ".jpg",
	"image/webp":               ".webp",
	"image/svg+xml":            ".svg",
}

// 后台站点设置
func (blogService *BlogService) GetSiteSettings() model.SiteSettings {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	return option.SiteSettingsFromOptions(blogService.options)
}

// 更新后台站点设置
func (blogService *BlogService) UpdateSiteSettings(siteSettings model.SiteSettings) error {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	if err := validateSiteSettings(siteSettings); err != nil {
		return err
	}

	updatedConfig := option.ApplySiteSettings(blogService.options.Config, siteSettings)
	if err := option.WriteConfig(blogService.options.ConfigPath, updatedConfig); err != nil {
		return err
	}

	blogService.options = option.OptionsFromConfig(blogService.options.ConfigPath, updatedConfig)

	if err := blogService.renderAllWithoutLock(); err != nil {
		return err
	}

	return nil
}

// 上传站点图标并写入站点设置
func (blogService *BlogService) UploadSiteIcon(fileHeader *multipart.FileHeader) (model.SiteSettings, error) {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	if fileHeader == nil {
		return model.SiteSettings{}, fmt.Errorf("请选择要上传的图标文件")
	}
	if fileHeader.Size > 2*1024*1024 {
		return model.SiteSettings{}, fmt.Errorf("图标文件不能超过 2MB")
	}
	uploadedFile, err := fileHeader.Open()
	if err != nil {
		return model.SiteSettings{}, fmt.Errorf("打开上传文件失败：%w", err)
	}
	defer uploadedFile.Close()

	iconContent, err := io.ReadAll(io.LimitReader(uploadedFile, 2*1024*1024+1))
	if err != nil {
		return model.SiteSettings{}, fmt.Errorf("读取上传文件失败：%w", err)
	}
	if len(iconContent) > 2*1024*1024 {
		return model.SiteSettings{}, fmt.Errorf("图标文件不能超过 2MB")
	}
	iconExtension, err := detectIconExtension(iconContent)
	if err != nil {
		return model.SiteSettings{}, err
	}

	iconFileName := "site-icon" + iconExtension
	iconFilePath := filepath.Join(blogService.options.AssetsDir, iconFileName)
	if err := filesystem.WriteFileCreatingDirectory(iconFilePath, iconContent, 0644); err != nil {
		return model.SiteSettings{}, err
	}

	updatedConfig := blogService.options.Config
	updatedConfig.Site.IconURL = "/assets/" + iconFileName
	if err := option.WriteConfig(blogService.options.ConfigPath, updatedConfig); err != nil {
		return model.SiteSettings{}, err
	}
	blogService.options = option.OptionsFromConfig(blogService.options.ConfigPath, updatedConfig)
	if err := blogService.renderAllWithoutLock(); err != nil {
		return model.SiteSettings{}, err
	}

	return option.SiteSettingsFromOptions(blogService.options), nil
}

func validateSiteSettings(siteSettings model.SiteSettings) error {
	if strings.TrimSpace(siteSettings.Title) == "" {
		return fmt.Errorf("站点标题不能为空")
	}
	if strings.TrimSpace(siteSettings.Description) == "" {
		return fmt.Errorf("站点描述不能为空")
	}
	if strings.TrimSpace(siteSettings.Language) == "" {
		return fmt.Errorf("站点语言不能为空")
	}
	switch strings.ToLower(strings.TrimSpace(siteSettings.ThemeDefault)) {
	case "", "auto", "light", "dark":
	default:
		return fmt.Errorf("默认主题只能是 auto、light 或 dark")
	}
	if strings.TrimSpace(siteSettings.IconURL) != "" && !isSupportedIconURL(siteSettings.IconURL) {
		return fmt.Errorf("网站 icon 只支持 http(s) 链接或 / 开头的站内路径")
	}
	if strings.TrimSpace(siteSettings.CommentProvider) != "" && strings.TrimSpace(siteSettings.CommentProvider) != "giscus" {
		return fmt.Errorf("评论服务暂只支持 giscus")
	}
	return nil
}

func detectIconExtension(iconContent []byte) (string, error) {
	detectedMIME := mimetype.Detect(iconContent)
	for currentMIME := detectedMIME; currentMIME != nil; currentMIME = currentMIME.Parent() {
		if iconExtension, supported := allowedIconMIMETypes[currentMIME.String()]; supported {
			return iconExtension, nil
		}
	}
	return "", fmt.Errorf("网站 icon 只支持 ico、png、jpg、webp 或 svg，当前文件类型是 %s", detectedMIME.String())
}

func isSupportedIconURL(iconURL string) bool {
	trimmedIconURL := strings.TrimSpace(iconURL)
	return strings.HasPrefix(trimmedIconURL, "http://") ||
		strings.HasPrefix(trimmedIconURL, "https://") ||
		strings.HasPrefix(trimmedIconURL, "/")
}
