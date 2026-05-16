package config

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/honeok/honepress/internal/core"
	"github.com/honeok/honepress/internal/model"
	"github.com/honeok/honepress/internal/validation"
)

const legacyPermalinkStructure = "/%postname%.html"

// config.yaml 结构
type Config struct {
	Data      DataConfig      `yaml:"data"`
	Admin     AdminConfig     `yaml:"admin"`
	Site      SiteConfig      `yaml:"site"`
	Comment   CommentConfig   `yaml:"comment"`
	Permalink PermalinkConfig `yaml:"permalink"`
	Theme     ThemeConfig     `yaml:"theme"`
}

// 数据目录
type DataConfig struct {
	Directory string `yaml:"directory"`
}

// 后台认证配置
type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// 站点展示配置
type SiteConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	IconURL     string `yaml:"iconURL"`
}

// 评论配置
type CommentConfig struct {
	Enabled bool         `yaml:"enabled"`
	Giscus  GiscusConfig `yaml:"giscus"`
}

// giscus 配置
type GiscusConfig struct {
	Repo       string `yaml:"repo"`
	RepoID     string `yaml:"repoID"`
	Category   string `yaml:"category"`
	CategoryID string `yaml:"categoryID"`
}

// 固定链接配置
type PermalinkConfig struct {
	Structure string `yaml:"structure"`
}

// 前台默认主题
type ThemeConfig struct {
	Default string `yaml:"default"`
	Font    string `yaml:"font"`
}

// 运行配置
type Options struct {
	ConfigPath         string
	Config             Config
	Title              string
	Description        string
	SiteIconURL        string
	ThemeDefault       string
	Font               string
	DataDir            string
	ContentDir         string
	PostsDir           string
	PublicDir          string
	AssetsDir          string
	AdminDistDir       string
	ThemeDistDir       string
	TemplateDir        string
	AdminUsername      string
	AdminPassword      string
	Comment            CommentOptions
	PermalinkStructure string
}

// 评论运行配置
type CommentOptions struct {
	Enabled          bool
	GiscusRepo       string
	GiscusRepoID     string
	GiscusCategory   string
	GiscusCategoryID string
}

// 解析配置文件路径
func ResolveConfigPath(arguments []string) (string, error) {
	flagSet := flag.NewFlagSet(core.ProjectName, flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	shortConfigPath := flagSet.String("c", "", "config file path")
	longConfigPath := flagSet.String("config", "", "config file path")
	if err := flagSet.Parse(arguments); err != nil {
		return "", fmt.Errorf("parse startup arguments: %w", err)
	}
	if strings.TrimSpace(*shortConfigPath) != "" {
		return *shortConfigPath, nil
	}
	if strings.TrimSpace(*longConfigPath) != "" {
		return *longConfigPath, nil
	}
	if strings.TrimSpace(os.Getenv("HONEPRESS_CONFIG")) != "" {
		return os.Getenv("HONEPRESS_CONFIG"), nil
	}
	return "./config.yaml", nil
}

// 读取配置
func Load(configPath string) (Options, error) {
	absoluteConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return Options{}, fmt.Errorf("resolve config path %s: %w", configPath, err)
	}

	if _, err := os.Stat(absoluteConfigPath); errors.Is(err, os.ErrNotExist) {
		defaultConfig := DefaultConfig()
		if err := WriteConfig(absoluteConfigPath, defaultConfig); err != nil {
			return Options{}, err
		}
		log.Printf("generated default config at %s", absoluteConfigPath)
	} else if err != nil {
		return Options{}, fmt.Errorf("stat config at %s: %w", absoluteConfigPath, err)
	}

	configFileContent, err := os.ReadFile(absoluteConfigPath)
	if err != nil {
		return Options{}, fmt.Errorf("read config at %s: %w", absoluteConfigPath, err)
	}

	migratedConfigFileContent, migratedConfig, err := migrateConfigContent(configFileContent)
	if err != nil {
		return Options{}, fmt.Errorf("migrate config at %s: %w", absoluteConfigPath, err)
	}
	if migratedConfig {
		if err := os.WriteFile(absoluteConfigPath, migratedConfigFileContent, 0644); err != nil {
			return Options{}, fmt.Errorf("write migrated config at %s: %w", absoluteConfigPath, err)
		}
		configFileContent = migratedConfigFileContent
		log.Printf("updated config at %s with missing defaults", absoluteConfigPath)
	}

	loadedConfig := DefaultConfig()
	if err := yaml.Unmarshal(configFileContent, &loadedConfig); err != nil {
		return Options{}, fmt.Errorf("decode config at %s: %w", absoluteConfigPath, err)
	}
	NormalizeConfig(&loadedConfig)
	if err := validation.ValidatePermalinkStructure(loadedConfig.Permalink.Structure); err != nil {
		return Options{}, fmt.Errorf("invalid permalink structure: %w", err)
	}

	loadedOptions := OptionsFromConfig(absoluteConfigPath, loadedConfig)
	if loadedOptions.AdminPassword == "" {
		log.Println("warning: admin password is not set; admin API is insecure")
	}
	if loadedOptions.Comment.Enabled && !loadedOptions.Comment.HasRequiredGiscusConfig() {
		log.Println("warning: giscus config is incomplete; comments will not be rendered")
	}

	return loadedOptions, nil
}

// 默认配置
func DefaultConfig() Config {
	return Config{
		Data: DataConfig{
			Directory: "data",
		},
		Admin: AdminConfig{
			Username: "",
			Password: "",
		},
		Site: SiteConfig{
			Title:       "",
			Description: "",
			IconURL:     "",
		},
		Comment: CommentConfig{
			Enabled: false,
			Giscus: GiscusConfig{
				Repo:       "",
				RepoID:     "",
				Category:   "",
				CategoryID: "",
			},
		},
		Permalink: PermalinkConfig{
			Structure: validation.DefaultPermalinkStructure,
		},
		Theme: ThemeConfig{
			Default: "auto",
			Font:    "default",
		},
	}
}

// 写入配置文件
func WriteConfig(configPath string, config Config) error {
	NormalizeConfig(&config)
	if err := validation.ValidatePermalinkStructure(config.Permalink.Structure); err != nil {
		return fmt.Errorf("invalid permalink structure: %w", err)
	}
	configDirectoryPath := filepath.Dir(configPath)
	if err := os.MkdirAll(configDirectoryPath, 0755); err != nil {
		return fmt.Errorf("create config directory at %s: %w", configDirectoryPath, err)
	}
	var configFileBuffer bytes.Buffer
	configEncoder := yaml.NewEncoder(&configFileBuffer)
	configEncoder.SetIndent(2)
	if err := configEncoder.Encode(config); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	if err := configEncoder.Close(); err != nil {
		return fmt.Errorf("close config encoder: %w", err)
	}
	configFileContent := configFileBuffer.Bytes()
	if err := os.WriteFile(configPath, configFileContent, 0644); err != nil {
		return fmt.Errorf("write config at %s: %w", configPath, err)
	}
	return nil
}

func migrateConfigContent(configFileContent []byte) ([]byte, bool, error) {
	var configDocument yaml.Node
	changed := false
	markChanged := func(fieldChanged bool) {
		if fieldChanged {
			changed = true
		}
	}

	if len(bytes.TrimSpace(configFileContent)) == 0 {
		configDocument.Kind = yaml.DocumentNode
		configDocument.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		changed = true
	} else if err := yaml.Unmarshal(configFileContent, &configDocument); err != nil {
		return nil, false, err
	}

	configRoot, rootChanged, err := configRootNode(&configDocument)
	if err != nil {
		return nil, false, err
	}
	markChanged(rootChanged)

	defaultConfig := DefaultConfig()
	dataConfig, dataChanged, err := ensureConfigMapping(configRoot, "data")
	if err != nil {
		return nil, false, err
	}
	markChanged(dataChanged)
	markChanged(ensureConfigScalar(dataConfig, "directory", defaultConfig.Data.Directory, "!!str"))

	adminConfig, adminChanged, err := ensureConfigMapping(configRoot, "admin")
	if err != nil {
		return nil, false, err
	}
	markChanged(adminChanged)
	markChanged(ensureConfigScalar(adminConfig, "username", defaultConfig.Admin.Username, "!!str"))
	markChanged(ensureConfigScalar(adminConfig, "password", defaultConfig.Admin.Password, "!!str"))

	siteConfig, siteChanged, err := ensureConfigMapping(configRoot, "site")
	if err != nil {
		return nil, false, err
	}
	markChanged(siteChanged)
	markChanged(ensureConfigScalar(siteConfig, "title", defaultConfig.Site.Title, "!!str"))
	markChanged(ensureConfigScalar(siteConfig, "description", defaultConfig.Site.Description, "!!str"))
	markChanged(ensureConfigScalar(siteConfig, "iconURL", defaultConfig.Site.IconURL, "!!str"))

	commentConfig, commentChanged, err := ensureConfigMapping(configRoot, "comment")
	if err != nil {
		return nil, false, err
	}
	markChanged(commentChanged)
	markChanged(ensureConfigScalar(commentConfig, "enabled", fmt.Sprintf("%t", defaultConfig.Comment.Enabled), "!!bool"))
	giscusConfig, giscusChanged, err := ensureConfigMapping(commentConfig, "giscus")
	if err != nil {
		return nil, false, err
	}
	markChanged(giscusChanged)
	markChanged(ensureConfigScalar(giscusConfig, "repo", defaultConfig.Comment.Giscus.Repo, "!!str"))
	markChanged(ensureConfigScalar(giscusConfig, "repoID", defaultConfig.Comment.Giscus.RepoID, "!!str"))
	markChanged(ensureConfigScalar(giscusConfig, "category", defaultConfig.Comment.Giscus.Category, "!!str"))
	markChanged(ensureConfigScalar(giscusConfig, "categoryID", defaultConfig.Comment.Giscus.CategoryID, "!!str"))

	permalinkConfig, permalinkChanged, err := ensureConfigMapping(configRoot, "permalink")
	if err != nil {
		return nil, false, err
	}
	markChanged(permalinkChanged)
	markChanged(ensureConfigScalar(permalinkConfig, "structure", legacyPermalinkStructure, "!!str"))

	themeConfig, themeChanged, err := ensureConfigMapping(configRoot, "theme")
	if err != nil {
		return nil, false, err
	}
	markChanged(themeChanged)
	markChanged(ensureConfigScalar(themeConfig, "default", defaultConfig.Theme.Default, "!!str"))
	markChanged(ensureConfigScalar(themeConfig, "font", defaultConfig.Theme.Font, "!!str"))

	if !changed {
		return configFileContent, false, nil
	}

	var configFileBuffer bytes.Buffer
	configEncoder := yaml.NewEncoder(&configFileBuffer)
	configEncoder.SetIndent(2)
	if err := configEncoder.Encode(&configDocument); err != nil {
		return nil, false, err
	}
	if err := configEncoder.Close(); err != nil {
		return nil, false, err
	}
	return configFileBuffer.Bytes(), true, nil
}

func configRootNode(configDocument *yaml.Node) (*yaml.Node, bool, error) {
	changed := false
	if configDocument.Kind == 0 {
		configDocument.Kind = yaml.DocumentNode
		changed = true
	}
	if configDocument.Kind != yaml.DocumentNode {
		return nil, false, fmt.Errorf("config document must be a YAML document")
	}
	if len(configDocument.Content) == 0 {
		configDocument.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		changed = true
	}
	configRoot := configDocument.Content[0]
	if configRoot.Kind == yaml.ScalarNode && configRoot.Tag == "!!null" {
		configRoot.Kind = yaml.MappingNode
		configRoot.Tag = "!!map"
		configRoot.Value = ""
		changed = true
	}
	if configRoot.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("config root must be a mapping")
	}
	return configRoot, changed, nil
}

func ensureConfigMapping(parentNode *yaml.Node, key string) (*yaml.Node, bool, error) {
	valueNode := configMappingValue(parentNode, key)
	if valueNode == nil {
		valueNode = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		parentNode.Content = append(parentNode.Content, configKeyNode(key), valueNode)
		return valueNode, true, nil
	}
	if valueNode.Kind == yaml.ScalarNode && valueNode.Tag == "!!null" {
		valueNode.Kind = yaml.MappingNode
		valueNode.Tag = "!!map"
		valueNode.Value = ""
		valueNode.Content = nil
		return valueNode, true, nil
	}
	if valueNode.Kind != yaml.MappingNode {
		return nil, false, fmt.Errorf("config field %s must be a mapping", key)
	}
	return valueNode, false, nil
}

func ensureConfigScalar(parentNode *yaml.Node, key string, value string, tag string) bool {
	if configMappingValue(parentNode, key) != nil {
		return false
	}
	parentNode.Content = append(parentNode.Content, configKeyNode(key), &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: value})
	return true
}

func configMappingValue(mappingNode *yaml.Node, key string) *yaml.Node {
	for contentIndex := 0; contentIndex+1 < len(mappingNode.Content); contentIndex += 2 {
		if mappingNode.Content[contentIndex].Value == key {
			return mappingNode.Content[contentIndex+1]
		}
	}
	return nil
}

func configKeyNode(key string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
}

// 转换运行配置
func OptionsFromConfig(configPath string, config Config) Options {
	NormalizeConfig(&config)

	dataDirectory := config.Data.Directory
	contentDirectory := filepath.Join(dataDirectory, "content")

	return Options{
		ConfigPath:         configPath,
		Config:             config,
		Title:              config.Site.Title,
		Description:        config.Site.Description,
		SiteIconURL:        config.Site.IconURL,
		ThemeDefault:       config.Theme.Default,
		Font:               config.Theme.Font,
		DataDir:            dataDirectory,
		ContentDir:         contentDirectory,
		PostsDir:           filepath.Join(contentDirectory, "posts"),
		PublicDir:          filepath.Join(dataDirectory, "public"),
		AssetsDir:          filepath.Join(dataDirectory, "assets"),
		AdminDistDir:       filepath.Join("dist", "admin"),
		ThemeDistDir:       filepath.Join("dist", "theme"),
		TemplateDir:        filepath.Join("frontend", "theme", "templates"),
		AdminUsername:      config.Admin.Username,
		AdminPassword:      config.Admin.Password,
		PermalinkStructure: config.Permalink.Structure,
		Comment: CommentOptions{
			Enabled:          config.Comment.Enabled,
			GiscusRepo:       config.Comment.Giscus.Repo,
			GiscusRepoID:     config.Comment.Giscus.RepoID,
			GiscusCategory:   config.Comment.Giscus.Category,
			GiscusCategoryID: config.Comment.Giscus.CategoryID,
		},
	}
}

// 补齐配置默认值
func NormalizeConfig(config *Config) {
	defaultConfig := DefaultConfig()
	if strings.TrimSpace(config.Data.Directory) == "" {
		config.Data.Directory = defaultConfig.Data.Directory
	}
	config.Site.Title = strings.TrimSpace(config.Site.Title)
	config.Site.Description = strings.TrimSpace(config.Site.Description)
	config.Site.IconURL = strings.TrimSpace(config.Site.IconURL)
	normalizeGiscusConfig(&config.Comment.Giscus)
	config.Permalink.Structure = validation.NormalizePermalinkStructure(config.Permalink.Structure)
	config.Theme.Default = normalizeThemeDefault(config.Theme.Default)
	config.Theme.Font = normalizeThemeFont(config.Theme.Font)
}

// 应用后台站点设置
func ApplySiteSettings(config Config, siteSettings model.SiteSettings) Config {
	config.Site.Title = strings.TrimSpace(siteSettings.Title)
	config.Site.Description = strings.TrimSpace(siteSettings.Description)
	config.Site.IconURL = strings.TrimSpace(siteSettings.IconURL)
	config.Admin.Username = strings.TrimSpace(siteSettings.AdminUsername)
	config.Admin.Password = strings.TrimSpace(siteSettings.AdminPassword)
	config.Comment.Enabled = siteSettings.CommentEnabled
	config.Comment.Giscus.Repo = strings.TrimSpace(siteSettings.GiscusRepo)
	config.Comment.Giscus.RepoID = strings.TrimSpace(siteSettings.GiscusRepoID)
	config.Comment.Giscus.Category = strings.TrimSpace(siteSettings.GiscusCategory)
	config.Comment.Giscus.CategoryID = strings.TrimSpace(siteSettings.GiscusCategoryID)
	config.Permalink.Structure = validation.NormalizePermalinkStructure(siteSettings.PermalinkStructure)
	config.Theme.Default = normalizeThemeDefault(siteSettings.ThemeDefault)
	config.Theme.Font = normalizeThemeFont(siteSettings.Font)
	NormalizeConfig(&config)
	return config
}

// 生成后台站点设置
func SiteSettingsFromOptions(options Options) model.SiteSettings {
	return model.SiteSettings{
		Title:              options.Title,
		Description:        options.Description,
		IconURL:            options.SiteIconURL,
		AdminUsername:      options.AdminUsername,
		AdminPassword:      options.AdminPassword,
		CommentEnabled:     options.Comment.Enabled,
		GiscusRepo:         options.Comment.GiscusRepo,
		GiscusRepoID:       options.Comment.GiscusRepoID,
		GiscusCategory:     options.Comment.GiscusCategory,
		GiscusCategoryID:   options.Comment.GiscusCategoryID,
		PermalinkStructure: options.PermalinkStructure,
		ThemeDefault:       options.ThemeDefault,
		Font:               options.Font,
	}
}

// 判断 giscus 配置是否完整
func (commentOptions CommentOptions) HasRequiredGiscusConfig() bool {
	return strings.TrimSpace(commentOptions.GiscusRepo) != "" &&
		strings.TrimSpace(commentOptions.GiscusRepoID) != "" &&
		strings.TrimSpace(commentOptions.GiscusCategory) != "" &&
		strings.TrimSpace(commentOptions.GiscusCategoryID) != ""
}

// 生成公开链接
func (options Options) ValidateRuntimeFiles() error {
	requiredDirectories := map[string]string{
		"admin dist directory": options.AdminDistDir,
		"theme dist directory": options.ThemeDistDir,
	}
	for directoryName, directoryPath := range requiredDirectories {
		if strings.TrimSpace(directoryPath) == "" {
			return fmt.Errorf("%s is not configured", directoryName)
		}
		directoryInfo, err := os.Stat(directoryPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%s does not exist at %s; build frontend assets and keep runtime templates", directoryName, directoryPath)
			}
			return fmt.Errorf("check %s at %s: %w", directoryName, directoryPath, err)
		}
		if !directoryInfo.IsDir() {
			return fmt.Errorf("%s is not a directory: %s", directoryName, directoryPath)
		}
	}

	requiredFiles := map[string]string{
		"admin entry file": filepath.Join(options.AdminDistDir, "index.html"),
		"theme entry file": filepath.Join(options.ThemeDistDir, "index.html"),
	}
	for fileName, filePath := range requiredFiles {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("%s does not exist at %s", fileName, filePath)
			}
			return fmt.Errorf("check %s at %s: %w", fileName, filePath, err)
		}
		if fileInfo.IsDir() {
			return fmt.Errorf("%s is not a file: %s", fileName, filePath)
		}
	}
	return nil
}

func (options Options) AbsoluteURL(publicPath string) string {
	if publicPath == "" {
		publicPath = "/"
	}
	if strings.HasPrefix(publicPath, "http://") || strings.HasPrefix(publicPath, "https://") {
		return publicPath
	}
	if !strings.HasPrefix(publicPath, "/") {
		publicPath = "/" + publicPath
	}
	return publicPath
}

func normalizeGiscusConfig(giscusConfig *GiscusConfig) {
	giscusConfig.Repo = strings.TrimSpace(giscusConfig.Repo)
	giscusConfig.RepoID = strings.TrimSpace(giscusConfig.RepoID)
	giscusConfig.Category = strings.TrimSpace(giscusConfig.Category)
	giscusConfig.CategoryID = strings.TrimSpace(giscusConfig.CategoryID)
}

func normalizeThemeDefault(themeDefault string) string {
	switch strings.ToLower(strings.TrimSpace(themeDefault)) {
	case "light":
		return "light"
	case "dark":
		return "dark"
	default:
		return "auto"
	}
}

func normalizeThemeFont(themeFont string) string {
	switch strings.ToLower(strings.TrimSpace(themeFont)) {
	case "douyin-sans":
		return "douyin-sans"
	default:
		return "default"
	}
}
