package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigPathPriority(t *testing.T) {
	t.Setenv("HONEPRESS_CONFIG", "env.yaml")

	configPath, err := ResolveConfigPath([]string{"--config", "long.yaml"})
	if err != nil {
		t.Fatalf("parse --config failed: %v", err)
	}
	if configPath != "long.yaml" {
		t.Fatalf("config path mismatch: got %s, want long.yaml", configPath)
	}

	configPath, err = ResolveConfigPath([]string{"-c", "short.yaml", "--config", "long.yaml"})
	if err != nil {
		t.Fatalf("parse -c failed: %v", err)
	}
	if configPath != "short.yaml" {
		t.Fatalf("config path mismatch: got %s, want short.yaml", configPath)
	}

	configPath, err = ResolveConfigPath([]string{})
	if err != nil {
		t.Fatalf("parse HONEPRESS_CONFIG failed: %v", err)
	}
	if configPath != "env.yaml" {
		t.Fatalf("config path mismatch: got %s, want env.yaml", configPath)
	}
}

func TestLoadGeneratesDefaultConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	loadedOptions, err := Load(configPath)
	if err != nil {
		t.Fatalf("load default config failed: %v", err)
	}
	if loadedOptions.Title != "" {
		t.Fatalf("site title mismatch: %s", loadedOptions.Title)
	}
	if loadedOptions.Font != "default" {
		t.Fatalf("default font mismatch: got %s, want default", loadedOptions.Font)
	}
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file was not generated: %v", err)
	}
	configFileContent, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config file failed: %v", err)
	}
	generatedConfig := string(configFileContent)
	if strings.Contains(generatedConfig, "server:") {
		t.Fatalf("default config must not include mutable server.listen")
	}
	if strings.Contains(generatedConfig, "provider:") {
		t.Fatalf("default config must not include fixed comment provider")
	}
	giscusAdvancedKeys := []string{"mapping:", "strict:", "reactionsEnabled:", "emitMetadata:", "inputPosition:", "lang:"}
	for _, giscusAdvancedKey := range giscusAdvancedKeys {
		if strings.Contains(generatedConfig, giscusAdvancedKey) {
			t.Fatalf("default config must not include advanced giscus key %s", giscusAdvancedKey)
		}
	}
}
