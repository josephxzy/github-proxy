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
		Host                string `toml:"host"`                // 监听地址，默认 "0.0.0.0"
		Port                int    `toml:"port"`                // 监听端口，默认 5000
		FileSize            int64  `toml:"fileSize"`            // 单文件最大大小（字节），默认 2GB
		EnableFrontend      bool   `toml:"enableFrontend"`      // 是否启用 Web 前端界面，默认 true
		GitHubToken         string `toml:"githubToken"`         // GitHub Personal Access Token，用于提高 API 速率限制
		BufferSize          int64  `toml:"bufferSize"`          // 水位线缓冲区大小（字节），默认 8MB
		DownloadIdleTimeout int64  `toml:"downloadIdleTimeout"` // 下载停滞超时（秒），客户端超过该时间无写入进展则关闭上游 GitHub 连接，0=禁用，默认 300
	} `toml:"server"`

	// RateLimit 速率限制配置
	RateLimit struct {
		APISearchHourly      int   `toml:"apiSearchHourly"`
		APIReleaseHourly     int   `toml:"apiReleaseHourly"`
		APIRepoHourly        int   `toml:"apiRepoHourly"`
		APIOtherHourly       int   `toml:"apiOtherHourly"`
		DownloadBytesPerSec  int64 `toml:"downloadBytesPerSec"`  // 单用户下载限速（字节/秒），0=不限速
		GlobalBytesPerSec    int64 `toml:"globalBytesPerSec"`    // 全局限速（字节/秒），0=不限速
		IPRequestLimit       int   `toml:"ipRequestLimit"`       // IP 请求限流（次/小时），0=不限
	} `toml:"rateLimit"`

	// Access 访问控制配置
	Access struct {
		WhiteList []string `toml:"whiteList"` // 仓库白名单，仅允许访问这些仓库（空表示不限制）
		BlackList []string `toml:"blackList"` // 仓库黑名单，禁止访问这些仓库
		Proxy     string   `toml:"proxy"`     // 上游代理地址，用于转发请求到 GitHub
	} `toml:"access"`

	// TokenWhiteList Token 白名单配置
	TokenWhiteList struct {
		Tokens []string `toml:"tokens"` // 白名单 GitHub Token 列表，匹配则不限速
	} `toml:"tokenWhiteList"`
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
			Host                string `toml:"host"`
			Port                int    `toml:"port"`
			FileSize            int64  `toml:"fileSize"`
			EnableFrontend      bool   `toml:"enableFrontend"`
			GitHubToken         string `toml:"githubToken"`
			BufferSize          int64  `toml:"bufferSize"`
			DownloadIdleTimeout int64  `toml:"downloadIdleTimeout"`
		}{
			Host:                "0.0.0.0",
			Port:                5000,
			FileSize:            2 * 1024 * 1024 * 1024,
			EnableFrontend:      true,
			GitHubToken:         "",
			BufferSize:          8 * 1024 * 1024,
			DownloadIdleTimeout: 300,
		},
		RateLimit: struct {
			APISearchHourly      int   `toml:"apiSearchHourly"`
			APIReleaseHourly     int   `toml:"apiReleaseHourly"`
			APIRepoHourly        int   `toml:"apiRepoHourly"`
			APIOtherHourly       int   `toml:"apiOtherHourly"`
			DownloadBytesPerSec  int64 `toml:"downloadBytesPerSec"`
			GlobalBytesPerSec    int64 `toml:"globalBytesPerSec"`
			IPRequestLimit       int   `toml:"ipRequestLimit"`
		}{
			APISearchHourly:     1200,
			APIReleaseHourly:    3333,
			APIRepoHourly:       3333,
			APIOtherHourly:      3333,
			DownloadBytesPerSec: 0,
			GlobalBytesPerSec:   0,
			IPRequestLimit:      0,
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
		TokenWhiteList: struct {
			Tokens []string `toml:"tokens"`
		}{
			Tokens: []string{}, // 无白名单 token
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
	configCopy.TokenWhiteList.Tokens = append([]string(nil), appConfig.TokenWhiteList.Tokens...)

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

// overrideFromEnv 从环境变量中读取配置并覆盖配置文件中的值
// 所有环境变量都采用大写命名，使用下划线分隔
// 支持的环境变量列表：
//
//	SERVER_HOST          - 服务器监听地址
//	SERVER_PORT          - 服务器监听端口
//	ENABLE_FRONTEND      - 是否启用前端 (true/false)
//	GITHUB_TOKEN         - GitHub Personal Access Token
//	MAX_FILE_SIZE        - 最大文件大小（字节）
//	API_SEARCH_HOURLY    - 搜索 API 每小时限制
//	API_RELEASE_HOURLY   - 发布 API 每小时限制
//	API_REPO_HOURLY      - 仓库 API 每小时限制
//	API_OTHER_HOURLY     - 其他 API 每小时限制
//	DOWNLOAD_RATE        - 下载限速（字节/秒），0=不限速
//	BUFFER_SIZE          - 水位线缓冲区大小（字节）
//	DOWNLOAD_IDLE_TIMEOUT - 下载停滞超时（秒），客户端长时间无写入进展时关闭上游连接
//	ACCESS_PROXY         - 上游代理地址
//	REPO_WHITELIST       - 仓库白名单（逗号分隔）
//	REPO_BLACKLIST       - 仓库黑名单（逗号分隔）
//	TOKEN_WHITELIST      - Token 白名单（逗号分隔）
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
	if val := os.Getenv("GITHUB_TOKEN"); val != "" {
		cfg.Server.GitHubToken = val
	}
	if val := os.Getenv("BUFFER_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil && size > 0 {
			cfg.Server.BufferSize = size
		}
	}
	if val := os.Getenv("DOWNLOAD_IDLE_TIMEOUT"); val != "" {
		if t, err := strconv.ParseInt(val, 10, 64); err == nil && t >= 0 {
			cfg.Server.DownloadIdleTimeout = t
		}
	}
	if val := os.Getenv("MAX_FILE_SIZE"); val != "" {
		if size, err := strconv.ParseInt(val, 10, 64); err == nil && size > 0 {
			cfg.Server.FileSize = size
		}
	}

	// 速率限制配置
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
	if val := os.Getenv("DOWNLOAD_RATE"); val != "" {
		if v, err := strconv.ParseInt(val, 10, 64); err == nil && v > 0 {
			cfg.RateLimit.DownloadBytesPerSec = v
		}
	}
	if val := os.Getenv("GLOBAL_RATE"); val != "" {
		if v, err := strconv.ParseInt(val, 10, 64); err == nil && v > 0 {
			cfg.RateLimit.GlobalBytesPerSec = v
		}
	}
	if val := os.Getenv("IP_REQUEST_LIMIT"); val != "" {
		if v, err := strconv.Atoi(val); err == nil && v > 0 {
			cfg.RateLimit.IPRequestLimit = v
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

	// Token 白名单配置
	if val := os.Getenv("TOKEN_WHITELIST"); val != "" {
		cfg.TokenWhiteList.Tokens = append(cfg.TokenWhiteList.Tokens, strings.Split(val, ",")...)
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
