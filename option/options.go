package option

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

	"github.com/honeok/honepress/constant"
	"github.com/honeok/honepress/model"
)

// config.yaml 结构
type Config struct {
	Data    DataConfig    `yaml:"data"`
	Admin   AdminConfig   `yaml:"admin"`
	Site    SiteConfig    `yaml:"site"`
	Comment CommentConfig `yaml:"comment"`
	Theme   ThemeConfig   `yaml:"theme"`
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

// 前台默认主题
type ThemeConfig struct {
	Default string `yaml:"default"`
	Font    string `yaml:"font"`
}

// 运行配置
type Options struct {
	ConfigPath    string
	Config        Config
	Title         string
	Description   string
	SiteIconURL   string
	ThemeDefault  string
	Font          string
	DataDir       string
	ContentDir    string
	PostsDir      string
	PublicDir     string
	AssetsDir     string
	AdminUsername string
	AdminPassword string
	Comment       CommentOptions
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
	if strings.TrimSpace(os.Getenv("HONEPRESS_CONFIG")) != "" {
		return os.Getenv("HONEPRESS_CONFIG"), nil
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
		log.Println("警告：giscus 配置不完整，文章页不会输出评论容器。")
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
			Username: "admin",
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
		Theme: ThemeConfig{
			Default: "auto",
			Font:    "default",
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
	var configFileBuffer bytes.Buffer
	configEncoder := yaml.NewEncoder(&configFileBuffer)
	configEncoder.SetIndent(2)
	if err := configEncoder.Encode(config); err != nil {
		return fmt.Errorf("生成配置文件失败：%w", err)
	}
	if err := configEncoder.Close(); err != nil {
		return fmt.Errorf("关闭配置文件编码器失败：%w", err)
	}
	configFileContent := configFileBuffer.Bytes()
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
		Title:         config.Site.Title,
		Description:   config.Site.Description,
		SiteIconURL:   config.Site.IconURL,
		ThemeDefault:  config.Theme.Default,
		Font:          config.Theme.Font,
		DataDir:       dataDirectory,
		ContentDir:    contentDirectory,
		PostsDir:      filepath.Join(contentDirectory, "posts"),
		PublicDir:     filepath.Join(dataDirectory, "public"),
		AssetsDir:     filepath.Join(dataDirectory, "assets"),
		AdminUsername: config.Admin.Username,
		AdminPassword: config.Admin.Password,
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
	if strings.TrimSpace(config.Admin.Username) == "" {
		config.Admin.Username = defaultConfig.Admin.Username
	}
	config.Site.Title = strings.TrimSpace(config.Site.Title)
	config.Site.Description = strings.TrimSpace(config.Site.Description)
	config.Site.IconURL = strings.TrimSpace(config.Site.IconURL)
	normalizeGiscusConfig(&config.Comment.Giscus)
	config.Theme.Default = normalizeThemeDefault(config.Theme.Default)
	config.Theme.Font = normalizeThemeFont(config.Theme.Font)
}

// 应用后台站点设置
func ApplySiteSettings(config Config, siteSettings model.SiteSettings) Config {
	config.Site.Title = strings.TrimSpace(siteSettings.Title)
	config.Site.Description = strings.TrimSpace(siteSettings.Description)
	config.Site.IconURL = strings.TrimSpace(siteSettings.IconURL)
	config.Comment.Enabled = siteSettings.CommentEnabled
	config.Comment.Giscus.Repo = strings.TrimSpace(siteSettings.GiscusRepo)
	config.Comment.Giscus.RepoID = strings.TrimSpace(siteSettings.GiscusRepoID)
	config.Comment.Giscus.Category = strings.TrimSpace(siteSettings.GiscusCategory)
	config.Comment.Giscus.CategoryID = strings.TrimSpace(siteSettings.GiscusCategoryID)
	config.Theme.Default = normalizeThemeDefault(siteSettings.ThemeDefault)
	config.Theme.Font = normalizeThemeFont(siteSettings.Font)
	NormalizeConfig(&config)
	return config
}

// 生成后台站点设置
func SiteSettingsFromOptions(options Options) model.SiteSettings {
	return model.SiteSettings{
		Title:            options.Title,
		Description:      options.Description,
		IconURL:          options.SiteIconURL,
		CommentEnabled:   options.Comment.Enabled,
		GiscusRepo:       options.Comment.GiscusRepo,
		GiscusRepoID:     options.Comment.GiscusRepoID,
		GiscusCategory:   options.Comment.GiscusCategory,
		GiscusCategoryID: options.Comment.GiscusCategoryID,
		ThemeDefault:     options.ThemeDefault,
		Font:             options.Font,
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
