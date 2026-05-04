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

// config.yaml 结构
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Data    DataConfig    `yaml:"data"`
	Admin   AdminConfig   `yaml:"admin"`
	Site    SiteConfig    `yaml:"site"`
	Comment CommentConfig `yaml:"comment"`
	Theme   ThemeConfig   `yaml:"theme"`
}

// 监听地址
type ServerConfig struct {
	Listen string `yaml:"listen"`
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
	BaseURL     string `yaml:"baseURL"`
	Language    string `yaml:"language"`
	GitHubURL   string `yaml:"githubURL"`
	TelegramURL string `yaml:"telegramURL"`
}

// 评论配置
type CommentConfig struct {
	Enabled  bool         `yaml:"enabled"`
	Provider string       `yaml:"provider"`
	Giscus   GiscusConfig `yaml:"giscus"`
}

// giscus 配置
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

// 前台默认主题
type ThemeConfig struct {
	Default string `yaml:"default"`
}

// 运行配置
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

// 评论运行配置
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

// 解析配置文件路径
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

// 读取配置
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

// 默认配置
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

// 写入配置文件
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

// 转换运行配置
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

// 补齐配置默认值
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

// 应用后台站点设置
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

// 生成后台站点设置
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

// 判断 giscus 配置是否完整
func (commentOptions CommentOptions) HasRequiredGiscusConfig() bool {
	return strings.TrimSpace(commentOptions.GiscusRepo) != "" &&
		strings.TrimSpace(commentOptions.GiscusRepoID) != "" &&
		strings.TrimSpace(commentOptions.GiscusCategory) != "" &&
		strings.TrimSpace(commentOptions.GiscusCategoryID) != ""
}

// 生成公开链接
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
