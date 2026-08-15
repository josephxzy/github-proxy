// Package github 提供 GitHub 代理服务的统一工具层。
// 它聚合 api 与 download 子包的包级函数，对外提供统一接口。
//
// 注意：本项目没有"编排层"服务对象——请求流水线实现在 handlers 包，
// 这里只导出 handler / router 实际调用的纯函数，避免无意义的中间层。
package github

import (
	"context"
	"io"
	"net/http"
	"sync"

	api "github-proxy/internal/github/api"
	download "github-proxy/internal/github/download"
)

type URLNormalizer = download.URLNormalizer

// NewURLNormalizer 创建 URL 规范化器实例。
func NewURLNormalizer() *URLNormalizer {
	return download.NewURLNormalizer()
}

// InitGlobalAPILimiters 初始化全局 API 速率限制器（应用启动时调用一次）。
func InitGlobalAPILimiters(searchPerHour, releasePerHour, repoPerHour, defaultPerHour int) {
	api.InitGlobalAPILimiters(searchPerHour, releasePerHour, repoPerHour, defaultPerHour)
}

// CheckAPIQueue 检查 API 请求是否需要排队等待限速。
// 白名单 token 用户（authenticated=true）豁免 API 限速。
func CheckAPIQueue(ctx context.Context, url string, authenticated bool) error {
	return api.CheckAPIQueue(ctx, url, authenticated)
}

// IsAPIRequest 判断 URL 是否应走 API 代理路径（api.github.com）。
func IsAPIRequest(url string) bool {
	return download.IsAPIRequest(url)
}

// MatchURL 从 GitHub URL 中提取 owner/repo 信息。
func MatchURL(url string) []string {
	return download.MatchURL(url)
}

// IsScriptURL 判断 URL 是否为需要 URL 替换的脚本文件（.sh/.ps1）。
func IsScriptURL(url string) bool {
	return download.IsScriptURL(url)
}

// IsArchiveURL 判断 URL 是否为 GitHub 源码归档（archive/zipball/tarball，
// 最终由忽略 Range 的 codeload 提供服务）。
func IsArchiveURL(url string) bool {
	return download.IsArchiveURL(url)
}

// IsGitSmartHTTPRequest 判断请求是否为 git 智能 HTTP 端点（git clone/push）。
func IsGitSmartHTTPRequest(rawPath string) bool {
	return download.IsGitSmartHTTPRequest(rawPath)
}

// ApplyGitHubToken 将服务器配置的 GitHub Token 应用到请求头。
func ApplyGitHubToken(req *http.Request, url string) {
	download.ApplyGitHubToken(req, url)
}

// ExtractToken 从请求中提取用户提供的 GitHub Token。
func ExtractToken(r *http.Request, rawPath string) string {
	return download.ExtractToken(r, rawPath)
}

// ApplyUserToken 将用户提供的 Token 应用到上游请求头。
func ApplyUserToken(req *http.Request, userToken string) {
	download.ApplyUserToken(req, userToken)
}

// GitBasicAuthValue 构造 GitHub git 端点接受的 Basic 认证头值（x-access-token:PAT）。
func GitBasicAuthValue(userToken string) string {
	return download.GitBasicAuthValue(userToken)
}

// ApplyGitBasicAuth 将用户 Token 以 GitHub git 端点接受的 Basic 格式应用到请求头。
func ApplyGitBasicAuth(req *http.Request, userToken string) {
	download.ApplyGitBasicAuth(req, userToken)
}

// StripProxyQueryParams 剥离代理专用查询参数（token）。
func StripProxyQueryParams(rawURL string) string {
	return download.StripProxyQueryParams(rawURL)
}

// PrefetchContentLength 通过 Range 协议探测文件总大小。
func PrefetchContentLength(ctx context.Context, url string, headers http.Header) int64 {
	return download.PrefetchContentLength(ctx, url, headers)
}

// ProcessSmart 智能处理脚本文件内容（URL 替换）。
func ProcessSmart(input io.Reader, isCompressed bool, host string) (io.Reader, int64, error) {
	return download.ProcessSmart(input, isCompressed, host)
}

// defaultBranchCache 默认分支的进程内缓存（key: "owner/repo"，value: 分支名）。
// 避免每次请求都实时调用 GitHub API，降低对上游的依赖。
var defaultBranchCache sync.Map

// GetDefaultBranchWithCache 获取指定仓库的默认分支名称（带进程内缓存）。
// 缓存未命中时实时查询 GitHub API；查询失败时回退为 "main"。
func GetDefaultBranchWithCache(owner, repo string) string {
	key := owner + "/" + repo
	if v, ok := defaultBranchCache.Load(key); ok {
		return v.(string)
	}
	branch, err := api.GetDefaultBranch(owner, repo)
	if err != nil {
		return "main"
	}
	defaultBranchCache.Store(key, branch)
	return branch
}
