package constant

// ProjectName 是项目和二进制的固定名称，集中定义可以避免部署脚本与代码出现分叉。
const ProjectName = "blog"

// DefaultAddress 是 config.yaml 没有设置监听地址时使用的默认值。
const DefaultAddress = ":8080"

// ReservedPublicFileNames 保存前台静态输出中不能被文章固定链接占用的文件名。
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
