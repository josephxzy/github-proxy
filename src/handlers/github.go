// Package handlers 提供 HTTP 请求处理函数
// 作为控制器层，负责：
//   - 接收和解析 HTTP 请求
//   - 调用服务层（service）执行业务逻辑
//   - 构造并返回 HTTP 响应
//
// 主要模块：
//   - GitHubProxyHandler: GitHub 代理的主入口处理器
//   - proxyAPIRequest: 处理 GitHub API 代理请求
//   - proxyDownloadRequest: 处理文件下载代理请求
//   - response_writer: 流式响应写入工具
package handlers

import (
	"net/http"
	"strings"

	"github-proxy/internal/service"
	ghproxyservice "github-proxy/internal/service/github"

	"github.com/gin-gonic/gin"
)

// globalApplication 全局应用实例
// 在 main.go 中通过 SetApplication 注入，供所有处理器使用
var globalApplication *service.Application

// SetApplication 设置全局应用实例
// 在服务启动时由 main.go 调用，将 Application 注入到 handler 层
func SetApplication(app *service.Application) {
	globalApplication = app
}

// GetApplication 获取全局应用实例
// 用于在需要访问服务层的地方获取 Application 对象
func GetApplication() *service.Application {
	return globalApplication
}

// TokenAuthMiddleware token 提取 + 白名单中间件。
// 在所有路由之前执行，确保 downstream handler 和 IP 限流器
// 都能通过 c.Get("authenticated") 获取状态。
func TokenAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		rawPath := strings.TrimLeft(c.Request.URL.RequestURI(), "/")
		token := ghproxyservice.ExtractToken(c.Request, rawPath)
		if token != "" {
			c.Set("userToken", token)
		}
		c.Set("authenticated", globalApplication.TokenWhiteList.IsWhitelisted(token))
		c.Next()
	}
}

// GitHubProxyHandler GitHub 代理的主入口处理器
func GitHubProxyHandler(c *gin.Context) {
	rawPath := normalizePath(c.Request.URL.RequestURI())

	// 剥离代理专用查询参数（token），不发给 GitHub
	rawPath = ghproxyservice.StripProxyQueryParams(rawPath)

	// 步骤1：根据 URL 类型分流处理
	// API 请求（api.github.com）不需要经过下载模块的 URL 验证
	if ghproxyservice.IsAPIRequest(rawPath) {
		// 仓库黑白名单检查（无法提取 owner/repo 的请求如搜索 API 直接放行）
		if !checkRepoAccess(c, rawPath) {
			return
		}
		ProxyGitHubRequest(c, rawPath)
		return
	}

	// 步骤2：URL 规范化（仅对下载请求进行验证）
	normalizeResult := globalApplication.GetURLNormalizer().Normalize(rawPath)
	if !normalizeResult.Valid {
		c.String(http.StatusForbidden, normalizeResult.ErrorMessage)
		return
	}
	rawPath = normalizeResult.NormalizedURL

	// 仓库黑白名单检查
	if !checkRepoAccess(c, rawPath) {
		return
	}

	// 步骤3：分发到对应的处理器
	ProxyGitHubRequest(c, rawPath)
}

// checkRepoAccess 执行仓库黑白名单检查。
// 从 URL 提取 owner/repo，提取失败（如搜索 API、健康检查路径）时放行——
// 黑白名单仅针对具体仓库的访问，不适用于无仓库标识的请求。
// 检查失败时写入错误响应并返回 false。
func checkRepoAccess(c *gin.Context, url string) bool {
	// 请求路径可能不带 scheme（如 API 分支的 "api.github.com/repos/..."），
	// 先补全为完整 URL，否则 MatchURL 的正则（要求 ^https?:// 开头）无法匹配。
	// scheme 统一为小写：既避免大写 scheme（"HTTP://..."）被二次补全成
	// 无法匹配的 URL，也保证大小写混合 scheme 能通过 MatchURL 的正则匹配。
	switch {
	case len(url) >= 7 && strings.EqualFold(url[:7], "http://"):
		url = "http://" + url[7:]
	case len(url) >= 8 && strings.EqualFold(url[:8], "https://"):
		url = "https://" + url[8:]
	default:
		url = "https://" + url
	}
	matches := ghproxyservice.MatchURL(url)
	if len(matches) < 2 {
		return true
	}
	result := globalApplication.AccessCtrl.CheckRepoAccess(matches)
	if !result.Allowed {
		c.Abort()
		c.String(result.ErrorCode, result.ErrorMessage)
		return false
	}
	return true
}

// normalizePath 标准化请求路径
// 去除 URI 前面的 "/" 斜杠，使其成为标准的 URL 格式
func normalizePath(uri string) string {
	return strings.TrimLeft(uri, "/")
}

// ProxyGitHubRequest 根据 URL 类型分发请求到不同的处理器
// 判断逻辑：
//   - API 请求（api.github.com/*）→ proxyAPIRequest
//   - 文件下载请求（releases/archive/blob/raw等）→ proxyDownloadRequest
func ProxyGitHubRequest(c *gin.Context, u string) {
	if ghproxyservice.IsAPIRequest(u) {
		proxyAPIRequest(c, u, 0)
	} else {
		proxyDownloadRequest(c, u, 0)
	}
}
