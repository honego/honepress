package web

import (
	"embed"
	"fmt"
	"io/fs"
)

// FS 嵌入后台构建产物和前台主题脚本，让容器运行层只需要单个 app 二进制。
//
// Docker 构建会在 Go 编译前把真实 Vite dist 复制进来，本地没有 npm 时使用占位 dist 保持可编译。
//
//go:embed admin/dist/* theme/dist/theme.js
var FS embed.FS

// AdminDistFS 返回后台静态文件系统。
func AdminDistFS() (fs.FS, error) {
	adminDistFS, err := fs.Sub(FS, "admin/dist")
	if err != nil {
		return nil, fmt.Errorf("读取内置后台文件失败：%w", err)
	}
	return adminDistFS, nil
}

// ThemeScript 返回前台主题脚本构建产物。
func ThemeScript() ([]byte, error) {
	themeScriptContent, err := FS.ReadFile("theme/dist/theme.js")
	if err != nil {
		return nil, fmt.Errorf("读取内置主题脚本失败：%w", err)
	}
	return themeScriptContent, nil
}
