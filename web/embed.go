package web

import "embed"

// AdminFallbackFS 嵌入后台入口源码，构建产物缺失时仍能让 Go 包保持完整。
//
// 实际部署时 Docker 会先执行 Vite 构建，HTTP 服务优先读取 web/admin/dist。
//
//go:embed admin/index.html
var AdminFallbackFS embed.FS
