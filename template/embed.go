package templatefiles

import "embed"

// 前台模板和样式
//
//go:embed index.html blog.html post.html style.css
var FS embed.FS
