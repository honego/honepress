package bootstrap

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/honeok/honepress/internal/filesystem"
)

const defaultPostFileName = "世界你好.md"

// GenerateDefaultPostIfEmpty creates the first post only when the user has no Markdown posts yet.
func GenerateDefaultPostIfEmpty(postsDir string) error {
	if err := filesystem.EnsureDirectory(postsDir); err != nil {
		return err
	}

	hasPost, err := hasMarkdownPost(postsDir)
	if err != nil {
		return err
	}
	if hasPost {
		return nil
	}

	targetFilePath, err := filesystem.SafeJoin(postsDir, defaultPostFileName)
	if err != nil {
		return err
	}
	return filesystem.WriteFileCreatingDirectory(targetFilePath, []byte(defaultPostMarkdown(time.Now())), 0644)
}

func hasMarkdownPost(postsDir string) (bool, error) {
	postEntries, err := os.ReadDir(postsDir)
	if err != nil {
		return false, fmt.Errorf("read posts directory %s: %w", postsDir, err)
	}
	for _, postEntry := range postEntries {
		if !postEntry.IsDir() && isMarkdownFile(postEntry.Name()) {
			return true, nil
		}
	}
	return false, nil
}

func isMarkdownFile(fileName string) bool {
	return strings.EqualFold(filepath.Ext(fileName), ".md")
}

func defaultPostMarkdown(now time.Time) string {
	return fmt.Sprintf(`---
title: "世界你好"
icon: "☘️"
date: "%s"
description: "欢迎使用 HonePress。"
draft: false
url: "1.html"
tags:
  - HonePress
---

欢迎使用 HonePress 。这是您的第一篇文章，编辑或删除它，然后开始写作吧！
`, now.Format("2006-01-02 15:04:05"))
}
