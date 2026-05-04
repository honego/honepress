package option

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveConfigPathPriority(t *testing.T) {
	t.Setenv("BLOG_CONFIG", "env.yaml")

	configPath, err := ResolveConfigPath([]string{"--config", "long.yaml"})
	if err != nil {
		t.Fatalf("解析 --config 失败：%v", err)
	}
	if configPath != "long.yaml" {
		t.Fatalf("配置路径不一致：got %s want long.yaml", configPath)
	}

	configPath, err = ResolveConfigPath([]string{"-c", "short.yaml", "--config", "long.yaml"})
	if err != nil {
		t.Fatalf("解析 -c 失败：%v", err)
	}
	if configPath != "short.yaml" {
		t.Fatalf("配置路径不一致：got %s want short.yaml", configPath)
	}

	configPath, err = ResolveConfigPath([]string{})
	if err != nil {
		t.Fatalf("解析 BLOG_CONFIG 失败：%v", err)
	}
	if configPath != "env.yaml" {
		t.Fatalf("配置路径不一致：got %s want env.yaml", configPath)
	}
}

func TestLoadGeneratesDefaultConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	loadedOptions, err := Load(configPath)
	if err != nil {
		t.Fatalf("加载默认配置失败：%v", err)
	}
	if loadedOptions.Title != "" {
		t.Fatalf("站点标题不一致：%s", loadedOptions.Title)
	}
	if loadedOptions.Font != "default" {
		t.Fatalf("默认字体不一致：got %s want default", loadedOptions.Font)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("配置文件没有生成：%v", err)
	}
}
