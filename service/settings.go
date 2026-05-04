package service

import (
	"fmt"
	"strings"

	"github.com/honeok/blog/model"
	"github.com/honeok/blog/option"
)

// GetSiteSettings 返回后台站点设置页允许编辑的配置。
func (blogService *BlogService) GetSiteSettings() model.SiteSettings {
	blogService.renderMutex.Lock()
	defer blogService.renderMutex.Unlock()

	return option.SiteSettingsFromOptions(blogService.options)
}

// UpdateSiteSettings 写入 config.yaml，并用新配置重新渲染静态页面。
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
		return nil
	default:
		return fmt.Errorf("默认主题只能是 auto、light 或 dark")
	}
}
