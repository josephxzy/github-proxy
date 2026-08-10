package download

import (
	"fmt"
	"net/http"
	"strings"
)

// URLNormalizer GitHub 下载 URL 规范化器。
// 负责将用户提交的各种形式的 GitHub URL 统一为合法的 HTTPS 下载链接：
//   - 自动补全缺失的协议前缀（http:// → https://，短路径 → github.com 完整链接）
//   - 将 blob 链接转换为 raw 链接（blob 返回 HTML 页面，raw 才是文件内容）
//   - 校验 URL 是否为合法的 GitHub 下载链接
type URLNormalizer struct{}

// NewURLNormalizer 创建 URL 规范化器实例。
func NewURLNormalizer() *URLNormalizer {
	return &URLNormalizer{}
}

// NormalizeResult URL 规范化结果。
type NormalizeResult struct {
	Valid         bool   // 是否规范化成功
	NormalizedURL string // 规范化后的完整 URL（Valid 为 true 时有效）
	Error         error  // 底层错误对象
	ErrorCode     int    // 拒绝时的 HTTP 状态码
	ErrorMessage  string // 面向用户的错误提示
}

// Normalize 规范化输入的下载路径。
// 流程：
//  1. 补全 HTTPS 前缀（ensureHTTPS）
//  2. 校验是否为合法的 GitHub 下载 URL（validateGitHubURL）
//  3. 将 blob 链接改写为 raw 链接，返回规范化结果
func (u *URLNormalizer) Normalize(rawPath string) *NormalizeResult {
	result := &NormalizeResult{}

	rawPath = u.ensureHTTPS(rawPath)

	if !u.validateGitHubURL(rawPath, result) {
		return result
	}

	if IsBlobURL(rawPath) {
		rawPath = strings.Replace(rawPath, "/blob/", "/raw/", 1)
	}

	result.Valid = true
	result.NormalizedURL = rawPath
	return result
}

// ensureHTTPS 为输入补全协议前缀。
// 规则：
//   - 已是 https:// → 原样返回
//   - 是 http://    → 强制升级为 https://
//   - 是短路径（如 owner/repo/...）→ 补全为 https://github.com/ 前缀
//   - 其他 → 原样返回（后续由 validateGitHubURL 判定合法性）
func (u *URLNormalizer) ensureHTTPS(rawPath string) string {
	if strings.HasPrefix(rawPath, "https://") {
		return rawPath
	}
	if strings.HasPrefix(rawPath, "http://") {
		return "https://" + rawPath[7:]
	}
	if IsShortGitHubPath(rawPath) {
		return "https://github.com/" + rawPath
	}
	return rawPath
}

// validateGitHubURL 校验 URL 是否为合法的 GitHub 下载链接。
// 通过 MatchURL 提取 owner/repo，提取失败则视为非法输入并填充错误信息。
func (u *URLNormalizer) validateGitHubURL(rawPath string, result *NormalizeResult) bool {
	matches := MatchURL(rawPath)
	if matches == nil {
		result.Error = fmt.Errorf("invalid input")
		result.ErrorCode = http.StatusForbidden
		result.ErrorMessage = "无效输入"
		return false
	}
	return true
}

// IsAPIRequest 判断 URL 是否应走 API 代理路径（api.github.com）。
// 在请求路由分发时使用：API 请求无需经过下载模块的 URL 规范化。
func IsAPIRequest(url string) bool {
	return IsGitHubAPIURL(url)
}
