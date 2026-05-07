package service

import (
	"fmt"
	"strings"

	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/model"
)

func (blogService *BlogService) GetSiteSettings() model.SiteSettings {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	return config.SiteSettingsFromOptions(blogService.options)
}

// 更新后台站点设置
func (blogService *BlogService) UpdateSiteSettings(siteSettings model.SiteSettings) error {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	if err := validateSiteSettings(siteSettings); err != nil {
		return err
	}

	updatedConfig := config.ApplySiteSettings(blogService.options.Config, siteSettings)
	if err := config.WriteConfig(blogService.options.ConfigPath, updatedConfig); err != nil {
		return err
	}

	blogService.options = config.OptionsFromConfig(blogService.options.ConfigPath, updatedConfig)

	if err := blogService.renderAllWithoutLock(); err != nil {
		return err
	}

	return nil
}

func validateSiteSettings(siteSettings model.SiteSettings) error {
	switch strings.ToLower(strings.TrimSpace(siteSettings.ThemeDefault)) {
	case "", "auto", "light", "dark":
	default:
		return fmt.Errorf("default theme must be auto, light, or dark")
	}
	switch strings.ToLower(strings.TrimSpace(siteSettings.Font)) {
	case "", "default", "douyin-sans":
	default:
		return fmt.Errorf("site font must be default or douyin-sans")
	}
	if strings.TrimSpace(siteSettings.IconURL) != "" && !isSupportedIconURL(siteSettings.IconURL) {
		return fmt.Errorf("site icon must be an http(s) URL or an absolute site path")
	}
	return nil
}

func isSupportedIconURL(iconURL string) bool {
	trimmedIconURL := strings.TrimSpace(iconURL)
	return strings.HasPrefix(trimmedIconURL, "http://") ||
		strings.HasPrefix(trimmedIconURL, "https://") ||
		strings.HasPrefix(trimmedIconURL, "/")
}
