package main

import (
	"log"
	"os"

	"github.com/honeok/blog/adapter/httpserver"
	"github.com/honeok/blog/option"
	"github.com/honeok/blog/service"
)

func main() {
	configPath, err := option.ResolveConfigPath(os.Args[1:])
	if err != nil {
		log.Fatalf("解析配置路径失败：%v", err)
	}

	loadedOptions, err := option.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败：%v", err)
	}
	blogService := service.NewBlogService(loadedOptions)

	if err := blogService.InitializeAndRender(); err != nil {
		log.Fatalf("启动渲染失败：%v", err)
	}

	httpServer := httpserver.New(loadedOptions, blogService)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("HTTP 服务停止：%v", err)
	}
}
