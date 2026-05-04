package option

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/honeok/blog/constant"
	"github.com/honeok/blog/model"
)

// Config 是 config.yaml 的完整结构，站点运行配置统一从这里读取。
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Data    DataConfig    `yaml:"data"`
	Admin   AdminConfig   `yaml:"admin"`
	Site    SiteConfig    `yaml:"site"`
	Comment CommentConfig `yaml:"comment"`
	Theme   ThemeConfig   `yaml:"theme"`
}

// ServerConfig 保存监听地址，修改后需要重启程序才会生效。
type ServerConfig struct {
	Listen string `yaml:"listen"`
}

// DataConfig 保存外置数据目录，文章和生成文件都位于这个目录下。
type DataConfig struct {
	Directory string `yaml:"directory"`
}

// AdminConfig 保存后台认证配置，后台设置页不会修改这些敏感字段。
type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// SiteConfig 保存站点展示配置，后台设置页会修改这些字段。
type SiteConfig struct {
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	BaseURL     string `yaml:"baseURL"`
	Language    string `yaml:"language"`
	GitHubURL   string `yaml:"githubURL"`
	TelegramURL string `yaml:"telegramURL"`
}

// CommentConfig 保存评论开关和 giscus 配置。
type CommentConfig struct {
	Enabled  bool         `yaml:"enabled"`
	Provider string       `yaml:"provider"`
	Giscus   GiscusConfig `yaml:"giscus"`
}

// GiscusConfig 保存 giscus 输出脚本所需字段。
type GiscusConfig struct {
	Repo             string `yaml:"repo"`
	RepoID           string `yaml:"repoID"`
	Category         string `yaml:"category"`
	CategoryID       string `yaml:"categoryID"`
	Mapping          string `yaml:"mapping"`
	Strict           string `yaml:"strict"`
	ReactionsEnabled string `yaml:"reactionsEnabled"`
	EmitMetadata     string `yaml:"emitMetadata"`
	InputPosition    string `yaml:"inputPosition"`
	Theme            string `yaml:"theme"`
	Lang             string `yaml:"lang"`
}

// ThemeConfig 保存前台默认主题。
type ThemeConfig struct {
	Default string `yaml:"default"`
}

// Options 保存服务启动后使用的派生配置。
type Options struct {
	ConfigPath    string
	Config        Config
	Address       string
	BaseURL       string
	Title         string
	Description   string
	Language      string
	GitHubURL     string
	TelegramURL   string
	ThemeDefault  string
	DataDir       string
	ContentDir    string
	PostsDir      string
	PublicDir     string
	AdminUsername string
	AdminPassword string
	Comment       CommentOptions
}

// CommentOptions 保存 giscus 输出所需的派生配置，后端只负责渲染脚本，不保存评论数据。
type CommentOptions struct {
	Enabled          bool
	Provider         string
	GiscusRepo       string
	GiscusRepoID     string
	GiscusCategory   string
	GiscusCategoryID string
	GiscusMapping    string
	GiscusStrict     string
	ReactionsEnabled string
	EmitMetadata     string
	InputPosition    string
	Theme            string
	Language         string
}

// ResolveConfigPath 根据命令行参数、BLOG_CONFIG 和默认路径确定配置文件。
func ResolveConfigPath(arguments []string) (string, error) {
	flagSet := flag.NewFlagSet(constant.ProjectName, flag.ContinueOnError)
	flagSet.SetOutput(os.Stderr)
	shortConfigPath := flagSet.String("c", "", "配置文件路径")
	longConfigPath := flagSet.String("config", "", "配置文件路径")
	if err := flagSet.Parse(arguments); err != nil {
		return "", fmt.Errorf("解析启动参数失败：%w", err)
	}
	if strings.TrimSpace(*shortConfigPath) != "" {
		return *shortConfigPath, nil
	}
	if strings.TrimSpace(*longConfigPath) != "" {
		return *longConfigPath, nil
	}
	if strings.TrimSpace(os.Getenv("BLOG_CONFIG")) != "" {
		return os.Getenv("BLOG_CONFIG"), nil
	}
	return "./config.yaml", nil
}

// Load 从 config.yaml 读取配置；文件不存在时会先生成默认配置再继续启动。
func Load(configPath string) (Options, error) {
	absoluteConfigPath, err := filepath.Abs(configPath)
	if err != nil {
		return Options{}, fmt.Errorf("解析配置文件路径失败：%w", err)
	}

	if _, err := os.Stat(absoluteConfigPath); errors.Is(err, os.ErrNotExist) {
		defaultConfig := DefaultConfig()
		if err := WriteConfig(absoluteConfigPath, defaultConfig); err != nil {
			return Options{}, err
		}
		log.Printf("配置文件不存在，已生成默认配置：%s", absoluteConfigPath)
	} else if err != nil {
		return Options{}, fmt.Errorf("检查配置文件失败：%s：%w", absoluteConfigPath, err)
	}

	configFileContent, err := os.ReadFile(absoluteConfigPath)
	if err != nil {
		return Options{}, fmt.Errorf("读取配置文件失败：%s：%w", absoluteConfigPath, err)
	}

	loadedConfig := DefaultConfig()
	if err := yaml.Unmarshal(configFileContent, &loadedConfig); err != nil {
		return Options{}, fmt.Errorf("解析配置文件失败：%s：%w", absoluteConfigPath, err)
	}
	NormalizeConfig(&loadedConfig)

	loadedOptions := OptionsFromConfig(absoluteConfigPath, loadedConfig)
	if loadedOptions.AdminPassword == "" {
		log.Println("警告：未设置后台密码，后台接口将不安全。")
	}
	if loadedOptions.Comment.Enabled && !loadedOptions.Comment.HasRequiredGiscusConfig() {
		log.Println("警告：giscus 配置不完整，文章页不会输出评论脚本。")
	}

	return loadedOptions, nil
}

// DefaultConfig 返回自动生成 config.yaml 时使用的安全默认值。
func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Listen: constant.DefaultAddress,
		},
		Data: DataConfig{
			Directory: "data",
		},
		Admin: AdminConfig{
			Username: "admin",
			Password: "",
		},
		Site: SiteConfig{
			Title:       "",
			Description: "",
			BaseURL:     "",
			Language:    "zh-CN",
			GitHubURL:   "",
			TelegramURL: "",
		},
		Comment: CommentConfig{
			Enabled:  false,
			Provider: "giscus",
			Giscus: GiscusConfig{
				Repo:             "",
				RepoID:           "",
				Category:         "",
				CategoryID:       "",
				Mapping:          "pathname",
				Strict:           "0",
				ReactionsEnabled: "1",
				EmitMetadata:     "0",
				InputPosition:    "bottom",
				Theme:            "preferred_color_scheme",
				Lang:             "zh-CN",
			},
		},
		Theme: ThemeConfig{
			Default: "auto",
		},
	}
}

// WriteConfig 把配置写回磁盘，后台保存站点设置时会复用同一套 YAML 输出。
func WriteConfig(configPath string, config Config) error {
	NormalizeConfig(&config)
	configDirectoryPath := filepath.Dir(configPath)
	if err := os.MkdirAll(configDirectoryPath, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败：%s：%w", configDirectoryPath, err)
	}
	configFileContent, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("生成配置文件失败：%w", err)
	}
	if err := os.WriteFile(configPath, configFileContent, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败：%s：%w", configPath, err)
	}
	return nil
}

// OptionsFromConfig 把 YAML 配置转换为运行时路径和评论等派生选项。
func OptionsFromConfig(configPath string, config Config) Options {
	NormalizeConfig(&config)

	dataDirectory := config.Data.Directory
	contentDirectory := filepath.Join(dataDirectory, "content")

	return Options{
		ConfigPath:    configPath,
		Config:        config,
		Address:       config.Server.Listen,
		BaseURL:       strings.TrimRight(config.Site.BaseURL, "/"),
		Title:         config.Site.Title,
		Description:   config.Site.Description,
		Language:      config.Site.Language,
		GitHubURL:     config.Site.GitHubURL,
		TelegramURL:   config.Site.TelegramURL,
		ThemeDefault:  config.Theme.Default,
		DataDir:       dataDirectory,
		ContentDir:    contentDirectory,
		PostsDir:      filepath.Join(contentDirectory, "posts"),
		PublicDir:     filepath.Join(dataDirectory, "public"),
		AdminUsername: config.Admin.Username,
		AdminPassword: config.Admin.Password,
		Comment: CommentOptions{
			Enabled:          config.Comment.Enabled,
			Provider:         config.Comment.Provider,
			GiscusRepo:       config.Comment.Giscus.Repo,
			GiscusRepoID:     config.Comment.Giscus.RepoID,
			GiscusCategory:   config.Comment.Giscus.Category,
			GiscusCategoryID: config.Comment.Giscus.CategoryID,
			GiscusMapping:    config.Comment.Giscus.Mapping,
			GiscusStrict:     config.Comment.Giscus.Strict,
			ReactionsEnabled: config.Comment.Giscus.ReactionsEnabled,
			EmitMetadata:     config.Comment.Giscus.EmitMetadata,
			InputPosition:    config.Comment.Giscus.InputPosition,
			Theme:            config.Comment.Giscus.Theme,
			Language:         config.Comment.Giscus.Lang,
		},
	}
}

// NormalizeConfig 补齐缺失字段，允许用户只在 config.yaml 写自己关心的配置。
func NormalizeConfig(config *Config) {
	defaultConfig := DefaultConfig()
	if strings.TrimSpace(config.Server.Listen) == "" {
		config.Server.Listen = defaultConfig.Server.Listen
	}
	if strings.TrimSpace(config.Data.Directory) == "" {
		config.Data.Directory = defaultConfig.Data.Directory
	}
	if strings.TrimSpace(config.Admin.Username) == "" {
		config.Admin.Username = defaultConfig.Admin.Username
	}
	config.Site.Title = strings.TrimSpace(config.Site.Title)
	config.Site.Description = strings.TrimSpace(config.Site.Description)
	if strings.TrimSpace(config.Site.Language) == "" {
		config.Site.Language = defaultConfig.Site.Language
	}
	config.Site.BaseURL = strings.TrimRight(strings.TrimSpace(config.Site.BaseURL), "/")
	config.Site.GitHubURL = strings.TrimSpace(config.Site.GitHubURL)
	config.Site.TelegramURL = strings.TrimSpace(config.Site.TelegramURL)
	if strings.TrimSpace(config.Comment.Provider) == "" {
		config.Comment.Provider = defaultConfig.Comment.Provider
	}
	normalizeGiscusConfig(&config.Comment.Giscus, defaultConfig.Comment.Giscus)
	config.Theme.Default = normalizeThemeDefault(config.Theme.Default)
}

// ApplySiteSettings 只修改后台允许管理的字段，启动级和敏感字段保持不变。
func ApplySiteSettings(config Config, siteSettings model.SiteSettings) Config {
	config.Site.Title = strings.TrimSpace(siteSettings.Title)
	config.Site.Description = strings.TrimSpace(siteSettings.Description)
	config.Site.BaseURL = strings.TrimRight(strings.TrimSpace(siteSettings.BaseURL), "/")
	config.Site.Language = strings.TrimSpace(siteSettings.Language)
	config.Site.GitHubURL = strings.TrimSpace(siteSettings.GitHubURL)
	config.Site.TelegramURL = strings.TrimSpace(siteSettings.TelegramURL)
	config.Comment.Enabled = siteSettings.CommentEnabled
	config.Theme.Default = normalizeThemeDefault(siteSettings.ThemeDefault)
	NormalizeConfig(&config)
	return config
}

// SiteSettingsFromOptions 返回后台站点设置区域需要编辑的字段。
func SiteSettingsFromOptions(options Options) model.SiteSettings {
	return model.SiteSettings{
		Title:          options.Title,
		Description:    options.Description,
		BaseURL:        options.BaseURL,
		Language:       options.Language,
		GitHubURL:      options.GitHubURL,
		TelegramURL:    options.TelegramURL,
		CommentEnabled: options.Comment.Enabled,
		ThemeDefault:   options.ThemeDefault,
	}
}

// HasRequiredGiscusConfig 判断 giscus 是否具备渲染脚本所需的最小配置。
func (commentOptions CommentOptions) HasRequiredGiscusConfig() bool {
	return strings.TrimSpace(commentOptions.GiscusRepo) != "" &&
		strings.TrimSpace(commentOptions.GiscusRepoID) != "" &&
		strings.TrimSpace(commentOptions.GiscusCategory) != "" &&
		strings.TrimSpace(commentOptions.GiscusCategoryID) != ""
}

// AbsoluteURL 把站内路径转换成 RSS 和 sitemap 需要的链接；baseURL 为空时保留站内路径。
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
	if strings.TrimSpace(options.BaseURL) == "" {
		return publicPath
	}
	return options.BaseURL + publicPath
}

func normalizeGiscusConfig(giscusConfig *GiscusConfig, defaultGiscusConfig GiscusConfig) {
	if strings.TrimSpace(giscusConfig.Mapping) == "" {
		giscusConfig.Mapping = defaultGiscusConfig.Mapping
	}
	if strings.TrimSpace(giscusConfig.Strict) == "" {
		giscusConfig.Strict = defaultGiscusConfig.Strict
	}
	if strings.TrimSpace(giscusConfig.ReactionsEnabled) == "" {
		giscusConfig.ReactionsEnabled = defaultGiscusConfig.ReactionsEnabled
	}
	if strings.TrimSpace(giscusConfig.EmitMetadata) == "" {
		giscusConfig.EmitMetadata = defaultGiscusConfig.EmitMetadata
	}
	if strings.TrimSpace(giscusConfig.InputPosition) == "" {
		giscusConfig.InputPosition = defaultGiscusConfig.InputPosition
	}
	if strings.TrimSpace(giscusConfig.Theme) == "" {
		giscusConfig.Theme = defaultGiscusConfig.Theme
	}
	if strings.TrimSpace(giscusConfig.Lang) == "" {
		giscusConfig.Lang = defaultGiscusConfig.Lang
	}
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
