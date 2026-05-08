package core

const ProjectName = "honepress"

// 默认监听地址
const DefaultAddress = ":8080"

// 保留的公开文件名
var ReservedPublicFileNames = map[string]struct{}{
	"index.html":   {},
	"blog.html":    {},
	"archive.html": {},
	"posts.html":   {},
	"rss.xml":      {},
	"sitemap.xml":  {},

	"admin.html": {},
	"api.html":   {},
}
