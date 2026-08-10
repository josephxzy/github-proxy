package config

import (
	"testing"
)

// TestDefaultConfig 验证默认配置的预设值。
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Host = %q, want 0.0.0.0", cfg.Server.Host)
	}
	if cfg.Server.Port != 5000 {
		t.Errorf("Port = %d, want 5000", cfg.Server.Port)
	}
	if cfg.Server.FileSize != 2*1024*1024*1024 {
		t.Errorf("FileSize = %d, want 2GB", cfg.Server.FileSize)
	}
	if !cfg.Server.EnableFrontend {
		t.Error("EnableFrontend = false, want true")
	}
	if cfg.Server.BufferSize != 8*1024*1024 {
		t.Errorf("BufferSize = %d, want 8MB", cfg.Server.BufferSize)
	}
	if cfg.Server.DownloadIdleTimeout != 300 {
		t.Errorf("DownloadIdleTimeout = %d, want 300", cfg.Server.DownloadIdleTimeout)
	}
	if cfg.RateLimit.APISearchHourly != 1200 {
		t.Errorf("APISearchHourly = %d, want 1200", cfg.RateLimit.APISearchHourly)
	}
	if cfg.RateLimit.APIReleaseHourly != 3333 || cfg.RateLimit.APIRepoHourly != 3333 || cfg.RateLimit.APIOtherHourly != 3333 {
		t.Error("API 限速默认值应为 3333")
	}
	if cfg.RateLimit.DownloadBytesPerSec != 0 || cfg.RateLimit.GlobalBytesPerSec != 0 || cfg.RateLimit.IPRequestLimit != 0 {
		t.Error("限速默认值应为 0（不限速）")
	}
	if len(cfg.Access.WhiteList) != 0 || len(cfg.Access.BlackList) != 0 || len(cfg.TokenWhiteList.Tokens) != 0 {
		t.Error("白/黑名单默认应为空")
	}
}

// TestGetConfigDefault 未初始化时 GetConfig 应返回默认配置。
func TestGetConfigDefault(t *testing.T) {
	setConfig(nil)
	cfg := GetConfig()

	if cfg.Server.Port != 5000 {
		t.Errorf("GetConfig() 未初始化时 Port = %d, want 5000", cfg.Server.Port)
	}
}

// TestGetConfigDeepCopy 修改返回副本的列表不应影响内部配置。
func TestGetConfigDeepCopy(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Access.WhiteList = []string{"owner/repo"}
	setConfig(cfg)

	// 获取副本并篡改列表
	copy1 := GetConfig()
	copy1.Access.WhiteList = append(copy1.Access.WhiteList, "other/repo")
	copy1.Access.BlackList = append(copy1.Access.BlackList, "bad/repo")
	copy1.TokenWhiteList.Tokens = append(copy1.TokenWhiteList.Tokens, "ghp_extra")

	// 再次获取，内部状态应保持原样
	copy2 := GetConfig()
	if len(copy2.Access.WhiteList) != 1 || copy2.Access.WhiteList[0] != "owner/repo" {
		t.Errorf("深拷贝失效: WhiteList = %v", copy2.Access.WhiteList)
	}
	if len(copy2.Access.BlackList) != 0 {
		t.Errorf("深拷贝失效: BlackList = %v", copy2.Access.BlackList)
	}
	if len(copy2.TokenWhiteList.Tokens) != 0 {
		t.Errorf("深拷贝失效: Tokens = %v", copy2.TokenWhiteList.Tokens)
	}
}

// TestOverrideFromEnv 验证环境变量覆盖配置值。
func TestOverrideFromEnv(t *testing.T) {
	cfg := DefaultConfig()

	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "8080")
	t.Setenv("ENABLE_FRONTEND", "false")
	t.Setenv("GITHUB_TOKEN", "ghp_testtoken")
	t.Setenv("MAX_FILE_SIZE", "1048576")
	t.Setenv("BUFFER_SIZE", "4096")
	t.Setenv("DOWNLOAD_IDLE_TIMEOUT", "60")
	t.Setenv("API_SEARCH_HOURLY", "100")
	t.Setenv("API_RELEASE_HOURLY", "200")
	t.Setenv("API_REPO_HOURLY", "300")
	t.Setenv("API_OTHER_HOURLY", "400")
	t.Setenv("DOWNLOAD_RATE", "5000")
	t.Setenv("GLOBAL_RATE", "8000")
	t.Setenv("IP_REQUEST_LIMIT", "50")
	t.Setenv("ACCESS_PROXY", "http://proxy.example:8080")
	t.Setenv("REPO_WHITELIST", "a/b,c/d")
	t.Setenv("REPO_BLACKLIST", "bad/*")
	t.Setenv("TOKEN_WHITELIST", "ghp_1,ghp_2")

	overrideFromEnv(cfg)

	if cfg.Server.Host != "127.0.0.1" || cfg.Server.Port != 8080 {
		t.Errorf("Server 覆盖失败: %+v", cfg.Server)
	}
	if cfg.Server.EnableFrontend {
		t.Error("EnableFrontend 应被覆盖为 false")
	}
	if cfg.Server.GitHubToken != "ghp_testtoken" {
		t.Errorf("GitHubToken = %q", cfg.Server.GitHubToken)
	}
	if cfg.Server.FileSize != 1048576 || cfg.Server.BufferSize != 4096 || cfg.Server.DownloadIdleTimeout != 60 {
		t.Errorf("Server 数值覆盖失败: %+v", cfg.Server)
	}
	if cfg.RateLimit.APISearchHourly != 100 || cfg.RateLimit.APIReleaseHourly != 200 ||
		cfg.RateLimit.APIRepoHourly != 300 || cfg.RateLimit.APIOtherHourly != 400 {
		t.Errorf("API 限速覆盖失败: %+v", cfg.RateLimit)
	}
	if cfg.RateLimit.DownloadBytesPerSec != 5000 || cfg.RateLimit.GlobalBytesPerSec != 8000 || cfg.RateLimit.IPRequestLimit != 50 {
		t.Errorf("限速覆盖失败: %+v", cfg.RateLimit)
	}
	if cfg.Access.Proxy != "http://proxy.example:8080" {
		t.Errorf("Proxy = %q", cfg.Access.Proxy)
	}
	if len(cfg.Access.WhiteList) != 2 || cfg.Access.WhiteList[0] != "a/b" || cfg.Access.WhiteList[1] != "c/d" {
		t.Errorf("WhiteList = %v", cfg.Access.WhiteList)
	}
	if len(cfg.Access.BlackList) != 1 || cfg.Access.BlackList[0] != "bad/*" {
		t.Errorf("BlackList = %v", cfg.Access.BlackList)
	}
	if len(cfg.TokenWhiteList.Tokens) != 2 || cfg.TokenWhiteList.Tokens[1] != "ghp_2" {
		t.Errorf("Tokens = %v", cfg.TokenWhiteList.Tokens)
	}
}

// TestOverrideFromEnvInvalidValues 非法环境变量值应被忽略（保留默认值）。
func TestOverrideFromEnvInvalidValues(t *testing.T) {
	cfg := DefaultConfig()

	t.Setenv("SERVER_PORT", "not-a-number")
	t.Setenv("MAX_FILE_SIZE", "-5")
	t.Setenv("API_SEARCH_HOURLY", "0")
	t.Setenv("DOWNLOAD_IDLE_TIMEOUT", "-1")

	overrideFromEnv(cfg)

	if cfg.Server.Port != 5000 {
		t.Errorf("非法 SERVER_PORT 应被忽略, got %d", cfg.Server.Port)
	}
	if cfg.Server.FileSize != 2*1024*1024*1024 {
		t.Errorf("非法 MAX_FILE_SIZE 应被忽略, got %d", cfg.Server.FileSize)
	}
	if cfg.RateLimit.APISearchHourly != 1200 {
		t.Errorf("非法 API_SEARCH_HOURLY 应被忽略, got %d", cfg.RateLimit.APISearchHourly)
	}
	if cfg.Server.DownloadIdleTimeout != 300 {
		t.Errorf("非法 DOWNLOAD_IDLE_TIMEOUT 应被忽略, got %d", cfg.Server.DownloadIdleTimeout)
	}
}
