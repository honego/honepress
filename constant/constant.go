package constant

const ProjectName = "honepress"

// 默认监听地址
const DefaultAddress = ":8080"

// 保留的公开文件名
var ReservedPublicFileNames = map[string]struct{}{
	"index.html":  {},
	"blog.html":   {},
	"rss.xml":     {},
	"sitemap.xml": {},
	"style.css":   {},
	"theme.js":    {},
	"admin.html":  {},
	"api.html":    {},
}
