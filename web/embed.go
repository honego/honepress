package web

import (
	"embed"
	"fmt"
	"io/fs"
)

// 后台和主题构建产物
//
//go:embed admin/dist/* theme/dist/theme.js
var FS embed.FS

// 后台静态文件系统
func AdminDistFS() (fs.FS, error) {
	adminDistFS, err := fs.Sub(FS, "admin/dist")
	if err != nil {
		return nil, fmt.Errorf("读取内置后台文件失败：%w", err)
	}
	return adminDistFS, nil
}

// 前台主题脚本构建产物
func ThemeScript() ([]byte, error) {
	themeScriptContent, err := FS.ReadFile("theme/dist/theme.js")
	if err != nil {
		return nil, fmt.Errorf("读取内置主题脚本失败：%w", err)
	}
	return themeScriptContent, nil
}
