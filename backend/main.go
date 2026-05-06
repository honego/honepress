package main

import (
	"log"
	"os"

	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/server"
	"github.com/honeok/honepress/internal/service"
)

func main() {
	configPath, err := config.ResolveConfigPath(os.Args[1:])
	if err != nil {
		log.Fatalf("解析配置路径失败：%v", err)
	}

	loadedOptions, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("加载配置失败：%v", err)
	}
	if err := loadedOptions.ValidateRuntimeFiles(); err != nil {
		log.Fatalf("检查运行时文件失败：%v", err)
	}
	blogService := service.NewBlogService(loadedOptions)

	if err := blogService.InitializeAndRender(); err != nil {
		log.Fatalf("启动渲染失败：%v", err)
	}

	httpServer := server.New(loadedOptions, blogService)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatalf("HTTP 服务停止：%v", err)
	}
}
