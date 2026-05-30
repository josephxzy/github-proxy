// Package config 提供应用程序的配置管理功能
// 支持 TOML 配置文件和环境变量两种配置方式
// 环境变量会覆盖配置文件中的同名配置项
//
// 配置加载优先级：
//  1. 默认值（硬编码）
//  2. TOML 配置文件（config.toml）
//  3. 环境变量（最高优先级）
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/pelletier/go-toml/v2"
)

// AppConfig 应用程序的主配置结构体
// 包含所有可配置的服务参数，支持 TOML 反序列化
type AppConfig struct {
	// Server 服务器基础配置
	Server struct {
		Host           string `toml:"host"`           // 监听地址，默认 "0.0.0.0"
		Port           int    `toml:"port"`           // 监听端口，默认 5000
		FileSize       int64  `toml:"fileSize"`       // 单文件最大大小（字节），默认 2GB
		EnableH2C      bool   `toml:"enableH2C"`      // 是否启用 H2C（HTTP/2 Cleartext），默认 false
		EnableFrontend bool   `toml:"enableFrontend"` // 是否启用 Web 前端界面，默认 true
		GitHubToken    string `toml:"githubToken"`    // GitHub Personal Access Token，用于提高 API 速率限制
	} `toml:"server"`

	// RateLimit 速率限制配置
	RateLimit struct {
		RequestLimit     int     `toml:"requestLimit"`     // 全局请求限制数量，默认 500
		PeriodHours      float64 `toml:"periodHours"`      // 限制周期（小时），默认 3 小时
		APISearchHourly  int     `toml:"apiSearchHourly"`  // 搜索 API 每小时请求限制，默认 1200
		APIReleaseHourly int     `toml:"apiReleaseHourly"` // 发布版本 API 每小时请求限制，默认 3333
		APIRepoHourly    int     `toml:"apiRepoHourly"`    // 仓库 API 每小时请求限制，默认 3333
		APIOtherHourly   int     `toml:"apiOtherHourly"`   // 其他 API 每小时请求限制，默认 3333
	} `toml:"rateLimit"`

	// Access 访问控制配置
	Access struct {
		WhiteList []string `toml:"whiteList"` // 仓库白名单，仅允许访问这些仓库（空表示不限制）
		BlackList []string `toml:"blackList"` // 仓库黑名单，禁止访问这些仓库
		Proxy     string   `toml:"proxy"`     // 上游代理地址，用于转发请求到 GitHub
	} `toml:"access"`

	// NodeRegistry 节点注册中心配置
	NodeRegistry struct {
		URLs      []string `toml:"urls"`      // 其他节点的 URL 列表，用于集群部署
		PublicURL string   `toml:"publicUrl"` // 当前节点的公网访问地址
	} `toml:"nodeRegistry"`

	// AuthUsers 认证用户配置
	AuthUsers struct {
		Users []string `toml:"users"` // 允许访问的用户 Token 列表（Basic Auth）
	} `toml:"authUsers"`
}

// GetRegistryURLs 返回节点注册中心的 URL 列表
func (c *AppConfig) GetRegistryURLs() []string {
	return c.NodeRegistry.URLs
}

var (
	// appConfig 全局配置实例（单例模式）
	appConfig     *AppConfig
	// appConfigLock 配置读写锁，保证并发安全
	appConfigLock sync.RWMutex
)

// DefaultConfig 创建并返回默认配置实例
// 所有配置项都使用预设的安全默认值
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: struct {
			Host           string `toml:"host"`
			Port           int    `toml:"port"`
			FileSize       int64  `toml:"fileSize"`
			EnableH2C      bool   `toml:"enableH2C"`
			EnableFrontend bool   `toml:"enableFrontend"`
			GitHubToken    string `toml:"githubToken"`
		}{
			Host:           "0.0.0.0",              // 监听所有网络接口
			Port:           5000,                    // 默认端口
			FileSize:       2 * 1024 * 1024 * 1024, // 2GB 文件大小限制
			EnableH2C:      false,                   // 默认不启用 H2C
			EnableFrontend: true,                    // 默认启用前端
			GitHubToken:    "",                      // 无默认 Token
		},
		RateLimit: struct {
			RequestLimit     int     `toml:"requestLimit"`
			PeriodHours      float64 `toml:"periodHours"`
			APISearchHourly  int     `toml:"apiSearchHourly"`
			APIReleaseHourly int     `toml:"apiReleaseHourly"`
			APIRepoHourly    int     `toml:"apiRepoHourly"`
			APIOtherHourly   int     `toml:"apiOtherHourly"`
		}{
			RequestLimit:     500,   // 每3小时最多500次请求
			PeriodHours:      3.0,   // 限制周期为3小时
			APISearchHourly:  1200,  // 搜索API：每小时1200次
			APIReleaseHourly: 3333,  // 发布API：每小时3333次
			APIRepoHourly:    3333,  // 仓库API：每小时3333次
			APIOtherHourly:   3333,  // 其他API：每小时3333次
		},
		Access: struct {
			WhiteList []string `toml:"whiteList"`
			BlackList []string `toml:"blackList"`
			Proxy     string   `toml:"proxy"`
		}{
			WhiteList: []string{}, // 空白名单，不限制
			BlackList: []string{}, // 空黑名单
			Proxy:     "",        // 无上游代理
		},
		NodeRegistry: struct {
			URLs      []string `toml:"urls"`
			PublicURL string   `toml:"publicUrl"`
		}{
			URLs:      []string{}, // 无其他节点
			PublicURL: "",        // 无公网地址
		},
		AuthUsers: struct {
			Users []string `toml:"users"`
		}{
			Users: []string{}, // 无认证用户
		},
	}
}

// GetConfig 获取当前配置的副本（线程安全）
// 返回配置的深拷贝，防止外部修改影响内部状态
// 如果配置未初始化，则返回默认配置
func GetConfig() *AppConfig {
	appConfigLock.RLock()
	defer appConfigLock.RUnlock()

	if appConfig == nil {
		return DefaultConfig()
	}

	// 创建配置的深拷贝，避免并发修改问题
	configCopy := *appConfig
	configCopy.Access.WhiteList = append([]string(nil), appConfig.Access.WhiteList...)
	configCopy.Access.BlackList = append([]string(nil), appConfig.Access.BlackList...)
	configCopy.AuthUsers.Users = append([]string(nil), appConfig.AuthUsers.Users...)

	return &configCopy
}

// setConfig 设置全局配置（内部使用，线程安全）
func setConfig(cfg *AppConfig) {
	appConfigLock.Lock()
	defer appConfigLock.Unlock()
	appConfig = cfg
}

// LoadConfig 加载配置文件的入口函数
// 按以下顺序加载配置：
//  1. 创建默认配置
//  2. 尝试读取 config.toml 文件并覆盖默认值
//  3. 读取环境变量并覆盖配置文件值
//  4. 将最终配置设置为全局配置
func LoadConfig() error {
	cfg := DefaultConfig()

	// 尝试读取配置文件
	if data, err := os.ReadFile("config.toml"); err == nil {
		if err := toml.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("解析配置文件失败: %v", err)
		}
	} else {
		fmt.Println("未找到config.toml，使用默认配置")
	}

	// 用环境变量覆盖配置文件中的值
	overrideFromEnv(cfg)

	// 保存到全局配置
	setConfig(cfg)

	return nil
}

// normalizeURL 标准化 URL 格式
// 去除已有的协议头（http:// 或 https://），然后统一添加 https://
// 用于确保所有节点 URL 使用一致的协议
func normalizeURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	// 去除可能存在的协议头
	rawURL = strings.TrimPrefix(rawURL, "https://")
	rawURL = strings.TrimPrefix(rawURL, "http://")
	// 默认使用 https://
	return "https://" + rawURL
}

// overrideFromEnv 从环境变量中读取配置并覆盖配置文件中的值
// 所有环境变量都采用大写命名，使用下划线分隔
// 支持的环境变量列表：
//
//	SERVER_HOST          - 服务器监听地址
//	SERVER_PORT          - 服务器监听端口
//	ENABLE_FRONTEND      - 是否启用前端 (true/false)
//	ENABLE_H2C           - 是否启用 H2C (true/false)
//	GITHUB_TOKEN         - GitHub Personal Access Token
//	MAX_FILE_SIZE        - 最大文件大小（字节）
//	REQUEST_LIMIT        - 全局请求限制数
//	REQUEST_PERIOD_HOURS - 请求限制周期（小时）
//	API_SEARCH_HOURLY    - 搜索 API 每小时限制
//	API_RELEASE_HOURLY   - 发布 API 每小时限制
//	API_REPO_HOURLY      - 仓库 API 每小时限制
//	API_OTHER_HOURLY     - 其他 API 每小时限制
//	ACCESS_PROXY         - 上游代理地址
//	REPO_WHITELIST       - 仓库白名单（逗号分隔）
//	REPO_BLACKLIST       - 仓库黑名单（逗号分隔）
//	NODE_REGISTRY_URLS   - 节点注册 URL 列表（逗号分隔）
//	NODE_PUBLIC_URL      - 当前节点公网地址
//	AUTH_USERS           - 认证用户列表（逗号分隔）
func overrideFromEnv(cfg *AppConfig) {
	// 服务器配置
	if val := os.Getenv("SERVER_HOST"); val != "" {
		cfg.Server.Host = val
	}
	if val := os.Getenv("SERVER_PORT"); val != "" {
		if port, err := strconv.Atoi(val); err == nil && port > 0 {
			cfg.Server.Port = port
		}
	}
	if val := os.Getenv("ENABLE_FRONTEND"); val != "" {
		cfg.Server.EnableFrontend = val == "true" || val == "1"
	}
	if val := os.Getenv("ENABLE_H2C"); val != "" {
		cfg.Server.EnableH2C = val == "true" || val == "1"
	}
	if val := os.Getenv("GITHUB_TOKEN"); val != "" {
		cfg.Server.GitHubToken = val
	}
	if val := os.Getenv("MAX_FILE_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil && size > 0 {
			cfg.Server.FileSize = size
		}
	}

	// 速率限制配置
	if val := os.Getenv("REQUEST_LIMIT"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil && limit > 0 {
			cfg.RateLimit.RequestLimit = limit
		}
	}
	if val := os.Getenv("REQUEST_PERIOD_HOURS"); val != "" {
		if hours, err := strconv.ParseFloat(val, 64); err == nil && hours > 0 {
			cfg.RateLimit.PeriodHours = hours
		}
	}
	if val := os.Getenv("API_SEARCH_HOURLY"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			cfg.RateLimit.APISearchHourly = v
		}
	}
	if val := os.Getenv("API_RELEASE_HOURLY"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			cfg.RateLimit.APIReleaseHourly = v
		}
	}
	if val := os.Getenv("API_REPO_HOURLY"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			cfg.RateLimit.APIRepoHourly = v
		}
	}
	if val := os.Getenv("API_OTHER_HOURLY"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			cfg.RateLimit.APIOtherHourly = v
		}
	}

	// 访问控制配置
	if val := os.Getenv("ACCESS_PROXY"); val != "" {
		cfg.Access.Proxy = val
	}
	if val := os.Getenv("REPO_WHITELIST"); val != "" {
		cfg.Access.WhiteList = append(cfg.Access.WhiteList, strings.Split(val, ",")...)
	}
	if val := os.Getenv("REPO_BLACKLIST"); val != "" {
		cfg.Access.BlackList = append(cfg.Access.BlackList, strings.Split(val, ",")...)
	}

	// 节点注册配置
	if val := os.Getenv("NODE_REGISTRY_URLS"); val != "" {
		for _, u := range strings.Split(val, ",") {
			cfg.NodeRegistry.URLs = append(cfg.NodeRegistry.URLs, normalizeURL(strings.TrimSpace(u)))
		}
	}
	if val := os.Getenv("NODE_PUBLIC_URL"); val != "" {
		cfg.NodeRegistry.PublicURL = normalizeURL(strings.TrimSpace(val))
	}

	// 认证用户配置
	if val := os.Getenv("AUTH_USERS"); val != "" {
		cfg.AuthUsers.Users = append(cfg.AuthUsers.Users, strings.Split(val, ",")...)
	}
}

// CreateDefaultConfigFile 生成默认的 config.toml 配置文件
// 用于快速创建初始配置文件模板
func CreateDefaultConfigFile() error {
	cfg := DefaultConfig()

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("序列化默认配置失败: %v", err)
	}

	return os.WriteFile("config.toml", data, 0644)
}
