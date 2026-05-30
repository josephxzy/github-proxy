package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github-proxy/config"
	"github-proxy/pkg/network"
)

// githubExps GitHub URL 匹配正则表达式列表。
// 用于从各种格式的 GitHub URL 中提取 owner 和 repo 信息
var githubExps = []*regexp.Regexp{
	// 匹配 GitHub API 端点（search、repos 等）
	regexp.MustCompile(`^(?:https?://)?api\.github\.com/(?:search|repos)/.*`),
	// 匹配 releases 和 archive 下载链接
	regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:releases|archive)/.*`),
	// 匹配 blob 和 raw 文件查看/下载链接
	regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:blob|raw)/.*`),
	// 匹配 raw.githubusercontent.com 链接
	regexp.MustCompile(`^(?:https?://)?raw\.github(?:usercontent|github)\.com/([^/]+)/([^/]+)/.+?/.+`),
	// 匹配 gist 链接
	regexp.MustCompile(`^(?:https?://)?gist\.(?:githubusercontent|github)\.com/([^/]+)/([^/]+).*`),
}

// MatchURL 从 GitHub URL 中提取 owner 和 repo 信息。
//
// 支持的 URL 格式：
//   - https://github.com/owner/repo/releases/...
//   - https://github.com/owner/repo/archive/...
//   - https://github.com/owner/repo/blob/...
//   - https://github.com/owner/repo/raw/...
//   - https://raw.githubusercontent.com/owner/repo/...
//   - https://gist.github.com/owner/...
//
// 返回值：
//   - []string{owner, repo} 或 nil（如果不匹配）
func MatchURL(u string) []string {
	for _, exp := range githubExps {
		if matches := exp.FindStringSubmatch(u); matches != nil {
			return matches[1:] // 返回捕获组（跳过完整匹配）
		}
	}
	return nil
}

// ApplyGitHubToken 应用 GitHub Personal Access Token 到请求头。
// 仅对 Release API 请求添加 Token，用于提高 API 速率限制。
//
// 为什么只对 Release API？
//   - Release API 是最常被限流的端点
//   - Token 可以将未认证的 60次/小时 提升到 5000次/小时
//   - 其他 API 通常不需要如此高的速率
func ApplyGitHubToken(req *http.Request, url string) {
	cfg := config.GetConfig()
	// 仅当配置了 Token 且是 Release API 请求时才添加
	if cfg.Server.GitHubToken != "" && strings.Contains(url, "api.github.com/repos/") && strings.Contains(url, "/releases") {
		req.Header.Set("Authorization", "token "+cfg.Server.GitHubToken)
	}
}

// IsGitHubAPIURL 判断 URL 是否指向 GitHub API（api.github.com）。
func IsGitHubAPIURL(u string) bool {
	return strings.Contains(u, "api.github.com")
}

// releaseAPIExp Release API 的正则表达式模式
var releaseAPIExp = regexp.MustCompile(`api\.github\.com/repos/[^/]+/[^/]+/releases`)

// IsReleaseAPIURL 判断 URL 是否为 Release API 请求。
func IsReleaseAPIURL(u string) bool {
	if !strings.Contains(u, "api.github.com") {
		return false
	}
	return releaseAPIExp.MatchString(u)
}

// GetDefaultBranch 获取指定仓库的默认分支名称。
// 通过调用 GitHub API 获取仓库信息。
//
// 参数:
//   - owner: 仓库所有者（用户名或组织名）
//   - repo: 仓库名称
//
// 返回值:
//   - string: 默认分支名称（如 "main"、"master" 等）
//   - error: 错误信息
func GetDefaultBranch(owner, repo string) (string, error) {
	// 构造 API URL
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", err
	}

	// 应用认证（如果有配置 Token）
	cfg := config.GetConfig()
	if cfg.Server.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+cfg.Server.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// 发送请求
	client := network.GetGlobalHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// 解析 JSON 响应
	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return "", err
	}

	// 验证默认分支不为空
	if repoInfo.DefaultBranch == "" {
		return "", fmt.Errorf("no default branch found")
	}

	return repoInfo.DefaultBranch, nil
}
