package templatefiles

import "embed"

// FS 嵌入前台模板和样式，让运行期不再依赖外部 template 目录。
//
//go:embed index.html blog.html post.html style.css
var FS embed.FS
