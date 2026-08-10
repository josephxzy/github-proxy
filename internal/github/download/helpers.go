package download

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github-proxy/internal/config"
)

// shortGitHubPathExp 短路径（user/repo/...）匹配规则，与 githubExps 的全 URL 规则保持一致：
//   - releases|archive|blob|raw|info 要求后面跟 "/"（如 /info/refs）
//   - git- 后面不要求 "/"，以支持 git 智能 HTTP 端点
//     /git-upload-pack、/git-receive-pack（短链接 git clone/push 的 POST 请求）
var shortGitHubPathExp = regexp.MustCompile(`^[^/\.]+/[^/]+(?:/(?:releases|archive|blob|raw|info)/.*|/git-.*)?$`)

// IsShortGitHubPath 判断字符串是否为合法的 GitHub 短路径（如 "owner/repo/..."）。
// 短路径用于支持 git clone 等场景下省略协议和域名的链接。
func IsShortGitHubPath(path string) bool {
	return shortGitHubPathExp.MatchString(path)
}

// githubExps GitHub URL 匹配正则表达式列表（api 与 download 两个子包共用）。
// 用于从各种格式的 GitHub URL 中提取 owner 和 repo 信息。
// 按用途分为三类：
//   - GitHub API 端点（search、repos）——供 API 代理路径的路由与重定向判断使用
//   - 仓库文件路径（releases/archive/blob/raw/info/git-）——文件下载路径
//   - 原始内容与 gist 域名（raw.githubusercontent.com、gist.github.com）
var githubExps = []*regexp.Regexp{
	// 匹配 GitHub 搜索 API 端点（无 owner/repo，不捕获）
	regexp.MustCompile(`^https?://api\.github\.com/search/.*`),
	// 匹配 GitHub repos API 端点（捕获 owner/repo，供黑白名单检查）
	regexp.MustCompile(`^https?://api\.github\.com/repos/([^/]+)/([^/]+).*`),
	// 匹配 releases 和 archive 下载链接
	regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/(?:releases|archive)/.*`),
	// 匹配 blob 和 raw 文件查看/下载链接
	regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/(?:blob|raw)/.*`),
	// 匹配 info 和 git 智能 HTTP 端点（git clone/push）
	regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/(?:info|git-).*`),
	// 匹配 raw.githubusercontent.com 链接
	regexp.MustCompile(`^https?://raw\.github(?:usercontent|github)\.com/([^/]+)/([^/]+)/.+?/.+`),
	// 匹配 gist 链接
	regexp.MustCompile(`^https?://gist\.(?:githubusercontent|github)\.com/([^/]+)/([^/]+).*`),
}

// MatchURL 从 GitHub URL 中提取 owner 和 repo 信息。
//
// 支持的 URL 格式：
//   - https://api.github.com/repos/...
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

// IsScriptURL 判断 URL 是否为脚本文件（.sh / .ps1）。
// 脚本文件下载后需要替换内部的 github.com 链接，使其经过代理访问。
func IsScriptURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.HasSuffix(lower, ".sh") || strings.HasSuffix(lower, ".ps1")
}

// blobURLExp blob 页面链接的正则表达式模式。
var blobURLExp = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/blob/.*`)

// IsBlobURL 判断 URL 是否为 blob 页面链接。
// blob 链接返回的是 GitHub 的 HTML 渲染页，需改写为 raw 链接才能获取文件内容。
func IsBlobURL(u string) bool {
	return blobURLExp.MatchString(u)
}

// IsGitHubAPIURL 判断 URL 是否指向 GitHub API（api.github.com）。
func IsGitHubAPIURL(u string) bool {
	return strings.Contains(u, "api.github.com")
}

// ApplyGitHubToken 将服务器配置的 GitHub Token 应用到请求头。
// 仅对 api.github.com 的请求添加，且不覆盖客户端已设置的 Authorization。
// 目的是提高 API 速率限制（未认证 60次/小时 → 认证 5000次/小时）。
func ApplyGitHubToken(req *http.Request, url string) {
	cfg := config.GetConfig()
	if cfg.Server.GitHubToken != "" && strings.Contains(url, "api.github.com") {
		if req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "token "+cfg.Server.GitHubToken)
		}
	}
}

// ExtractToken 从请求中提取用户提供的 GitHub Token。
// 提取顺序（优先级从高到低）：
//  1. X-GitHub-Token 请求头
//  2. URL 查询参数 token（如 ?token=xxx）
//  3. HTTP Basic 认证（用户名或密码作为 token）
//
// 提取到的 token 会通过 ApplyUserToken 应用到上游请求。
func ExtractToken(r *http.Request, rawPath string) string {
	if token := r.Header.Get("X-GitHub-Token"); token != "" {
		return token
	}
	if token := extractTokenFromQuery(rawPath); token != "" {
		return token
	}
	return extractTokenFromBasicAuth(r)
}

// extractTokenFromQuery 从 URL 查询参数中提取 token。
func extractTokenFromQuery(rawPath string) string {
	parsed, err := url.Parse(rawPath)
	if err != nil {
		return ""
	}
	return parsed.Query().Get("token")
}

// extractTokenFromBasicAuth 从 HTTP Basic 认证头中提取 token。
// Basic 格式为 "user:pass"，将非空的一侧作为 token（优先取密码）。
func extractTokenFromBasicAuth(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return ""
	}
	payload, err := base64.StdEncoding.DecodeString(auth[6:])
	if err != nil {
		return ""
	}
	parts := strings.SplitN(string(payload), ":", 2)
	if len(parts) != 2 {
		return ""
	}
	if parts[1] != "" {
		return parts[1]
	}
	return parts[0]
}

// ApplyUserToken 将用户提供的 Token 应用到上游请求的 Authorization 头。
func ApplyUserToken(req *http.Request, userToken string) {
	if userToken != "" {
		req.Header.Set("Authorization", "token "+userToken)
	}
}

// StripProxyQueryParams 剥离代理专用的查询参数（token），避免其被转发给 GitHub。
// 代理使用 ?token=xxx 传递认证信息，转发前必须移除，防止 Token 泄露给上游。
func StripProxyQueryParams(rawURL string) string {
	if !strings.Contains(rawURL, "?") {
		return rawURL
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	q := parsed.Query()
	q.Del("token")
	parsed.RawQuery = q.Encode()
	return parsed.String()
}
