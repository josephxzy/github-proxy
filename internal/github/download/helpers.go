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

// IsArchiveURL 判断 URL 是否为 GitHub 源码归档（archive / zipball / tarball）。
// 归档最终由 codeload.github.com 提供服务（github.com 与 api.github.com
// 的 archive 链接都会 302 到 codeload）。codeload 忽略 Range 请求头
// （对 Range 一律返回 200 全量），因此：
//   - 断连后无法按偏移续传，只能整包重拉 + 跳过已发字节（skip-resume）；
//   - Range 预检对归档无意义（拿不到 Content-Range），应跳过。
func IsArchiveURL(url string) bool {
	lower := strings.ToLower(url)
	return strings.Contains(lower, "codeload.github.com") ||
		strings.Contains(lower, "legacy.zip") ||
		strings.Contains(lower, "/archive/") ||
		strings.Contains(lower, "/zipball/") ||
		strings.Contains(lower, "/tarball/")
}

// gitSmartHTTPExp 匹配 git 智能 HTTP 端点路径段：
// info/refs（服务发现，git clone 首请求）、info/lfs（Git LFS batch）、
// git-upload-pack / git-receive-pack（数据传输）。
// 只匹配"段首 → 段尾（? / 或结束）"，误伤下载文件名的情况由
// IsGitSmartHTTPRequest 的下载前缀排除兜底。
var gitSmartHTTPExp = regexp.MustCompile(`(?:^|/)(?:info/refs|info/lfs|git-upload-pack|git-receive-pack)(?:[?/]|$)`)

// gitDownloadPathPrefixes 下载路径特征段：这些前缀存在时，即使文件名恰好为
// git 端点名（如 releases/.../git-upload-pack、blob/.../info/refs），也按普通
// 下载处理——不触发 401 挑战、上游仍用 token 头，避免误伤 web 下载。
var gitDownloadPathPrefixes = []string{"/releases/", "/archive/", "/blob/", "/raw/"}

// IsGitSmartHTTPRequest 判断请求是否为 git 智能 HTTP 端点。
// 用于 git clone/push 场景的白名单认证：
// git 客户端首次请求从不携带凭据（即使 URL 内嵌 token），只有收到 401 后
// 才会用 URL 凭据重试，因此代理需要对这些端点主动发起认证挑战。
// rawPath 为原始请求 URI（含 .git 与 query，如 "owner/repo.git/info/refs?service=git-upload-pack"）。
func IsGitSmartHTTPRequest(rawPath string) bool {
	for _, p := range gitDownloadPathPrefixes {
		if strings.Contains(rawPath, p) {
			return false
		}
	}
	return gitSmartHTTPExp.MatchString(rawPath)
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
// 注意：仅适用于 GitHub API（api.github.com）等接受 "token <PAT>" 头的端点；
// git 智能 HTTP 端点必须使用 ApplyGitBasicAuth（GitHub 只接受 Basic）。
func ApplyUserToken(req *http.Request, userToken string) {
	if userToken != "" {
		req.Header.Set("Authorization", "token "+userToken)
	}
}

// GitBasicAuthValue 构造 GitHub git 智能 HTTP 端点接受的 Basic 认证头值。
// GitHub 的 git 端点（github.com/owner/repo.git）不接受 "Authorization: token <PAT>"
// 头（一律 401），只接受 Basic；官方推荐格式为用户名 x-access-token、密码 PAT。
func GitBasicAuthValue(userToken string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte("x-access-token:"+userToken))
}

// ApplyGitBasicAuth 将用户 Token 以 GitHub git 端点接受的 Basic 格式应用到请求头。
func ApplyGitBasicAuth(req *http.Request, userToken string) {
	if userToken != "" {
		req.Header.Set("Authorization", GitBasicAuthValue(userToken))
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
