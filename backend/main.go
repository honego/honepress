package main

import (
	"fmt"
	"log"
	"os"

	"github.com/honeok/honepress/internal/bootstrap"
	"github.com/honeok/honepress/internal/config"
	"github.com/honeok/honepress/internal/server"
	"github.com/honeok/honepress/internal/service"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	configPath, err := config.ResolveConfigPath(os.Args[1:])
	if err != nil {
		return fmt.Errorf("parse config path: %w", err)
	}

	loadedOptions, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if err := loadedOptions.ValidateRuntimeFiles(); err != nil {
		return fmt.Errorf("validate runtime files: %w", err)
	}
	if err := bootstrap.GenerateDefaultPostIfEmpty(loadedOptions.PostsDir); err != nil {
		return fmt.Errorf("generate default post: %w", err)
	}

	blogService := service.NewBlogService(loadedOptions)
	if err := blogService.InitializeAndRender(); err != nil {
		return fmt.Errorf("initialize blog service: %w", err)
	}

	httpServer, err := server.New(loadedOptions, blogService)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}
	if err := httpServer.ListenAndServe(); err != nil {
		return fmt.Errorf("start server: %w", err)
	}
	return nil
}
