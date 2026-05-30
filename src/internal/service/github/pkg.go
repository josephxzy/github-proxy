// Package github 提供 GitHub 代理服务的统一导出层。
// 作为 api 和 download 子包的聚合入口，对外提供统一的类型和函数接口。
package github

import (
	"context"
	"io"
	"net/http"

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

func NewURLNormalizer() *URLNormalizer {
	return download.NewURLNormalizer()
}

func NewDownloadService(cfg *config.AppConfig) *DownloadService {
	return download.NewDownloadService(cfg)
}

func NewAPIService() *APIService {
	return api.NewAPIService()
}

func InitGlobalAPILimiters(searchPerHour, releasePerHour, repoPerHour, defaultPerHour int) {
	api.InitGlobalAPILimiters(searchPerHour, releasePerHour, repoPerHour, defaultPerHour)
}

func CheckAPIQueue(ctx context.Context, url string) error {
	return api.CheckAPIQueue(ctx, url)
}

func IsAPIRequest(url string) bool {
	return download.IsAPIRequest(url)
}

func MatchURL(url string) []string {
	return download.MatchURL(url)
}

func IsGitHubAPIURL(url string) bool {
	return download.IsGitHubAPIURL(url)
}

func IsScriptURL(url string) bool {
	return download.IsScriptURL(url)
}

func IsBlobURL(url string) bool {
	return download.IsBlobURL(url)
}

func IsReleaseAPIURL(url string) bool {
	return download.IsReleaseAPIURL(url)
}

func ApplyGitHubToken(req *http.Request, url string) {
	download.ApplyGitHubToken(req, url)
}

func PrefetchContentLength(ctx context.Context, url string, headers http.Header) int64 {
	return download.PrefetchContentLength(ctx, url, headers)
}

func ProcessSmart(input io.Reader, isCompressed bool, host string) (io.Reader, int64, error) {
	return download.ProcessSmart(input, isCompressed, host)
}

const MaxShellSize = download.MaxShellSize

func GetDefaultBranch(owner, repo string) (string, error) {
	return api.GetDefaultBranch(owner, repo)
}

func GetDefaultBranchWithCache(owner, repo string) string {
	return "main"
}
