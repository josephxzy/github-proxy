// Package github 提供 GitHub 代理服务的统一导出层。
// 作为 api 和 download 子包的聚合入口，对外提供统一的类型和函数接口。
package github

import (
	"context"
	"io"
	"net/http"
	"sync"

	"github-proxy/config"
	api "github-proxy/internal/service/github/api"
	download "github-proxy/internal/service/github/download"
)

type URLNormalizer = download.URLNormalizer
type DownloadService = download.DownloadService
type DownloadRequest = download.DownloadRequest
type DownloadResult = download.DownloadResult
type APIService = api.APIService
type APIRequest = api.APIRequest
type APIResult = api.APIResult

// NewURLNormalizer 创建 URL 规范化器实例。
func NewURLNormalizer() *URLNormalizer {
	return download.NewURLNormalizer()
}

// NewDownloadService 创建文件下载服务实例。
func NewDownloadService(cfg *config.AppConfig) *DownloadService {
	return download.NewDownloadService(cfg)
}

// NewAPIService 创建 API 代理服务实例。
func NewAPIService() *APIService {
	return api.NewAPIService()
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

// IsGitHubAPIURL 判断 URL 是否指向 GitHub API。
func IsGitHubAPIURL(url string) bool {
	return download.IsGitHubAPIURL(url)
}

// IsScriptURL 判断 URL 是否为需要 URL 替换的脚本文件（.sh/.ps1）。
func IsScriptURL(url string) bool {
	return download.IsScriptURL(url)
}

// IsBlobURL 判断 URL 是否为 blob 页面链接。
func IsBlobURL(url string) bool {
	return download.IsBlobURL(url)
}

// IsReleaseAPIURL 判断 URL 是否为 Release API 请求。
func IsReleaseAPIURL(url string) bool {
	return download.IsReleaseAPIURL(url)
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

// IsShortGitHubPath 判断字符串是否为合法的 GitHub 短路径。
func IsShortGitHubPath(path string) bool {
	return download.IsShortGitHubPath(path)
}

const MaxShellSize = download.MaxShellSize

// GetDefaultBranch 获取指定仓库的默认分支名称（实时请求 GitHub API）。
func GetDefaultBranch(owner, repo string) (string, error) {
	return api.GetDefaultBranch(owner, repo)
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
	branch, err := GetDefaultBranch(owner, repo)
	if err != nil {
		return "main"
	}
	defaultBranchCache.Store(key, branch)
	return branch
}
