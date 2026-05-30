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

	"github-proxy/config"
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

// GitHubProxyHandler GitHub 代理的主入口处理器
// 所有经过此代理的 GitHub 请求都会进入这个函数
//
//  1. 路径标准化：去除前导斜杠
//  2. 身份验证：如果配置了认证用户，则进行 Basic Auth 验证
//  3. URL 规范化：检查并转换 URL 格式（如 ghproxy.com → github.com）
//  4. 请求分发：根据 URL 类型转发到 API 或下载处理器
//
// 路由示例：
//   - /https://github.com/xxx/yyy → 代理到 GitHub
//   - /repo/user/project → 简写路径格式
func GitHubProxyHandler(c *gin.Context) {
	// 步骤1：标准化路径，去除前导 "/"
	rawPath := normalizePath(c.Request.URL.RequestURI())

	// 获取配置
	cfg := config.GetConfig()
	authenticated := false

	// 步骤2：身份验证（如果配置了认证用户）
	if len(cfg.AuthUsers.Users) > 0 {
		authResult := globalApplication.Auth.Authenticate(rawPath)
		if authResult.AuthPrefix != "" {
			c.Set("authPrefix", authResult.AuthPrefix)
		}
		rawPath = authResult.RawPath
		authenticated = authResult.Authenticated
	}

	// 将认证状态存入上下文，供后续中间件/处理器使用
	c.Set("authenticated", authenticated)

	// 步骤3：根据 URL 类型分流处理
	// API 请求（api.github.com）不需要经过下载模块的 URL 验证
	if ghproxyservice.IsAPIRequest(rawPath) {
		ProxyGitHubRequest(c, rawPath)
		return
	}

	// 步骤4：URL 规范化（仅对下载请求进行验证）
	normalizeResult := globalApplication.GetURLNormalizer().Normalize(rawPath)
	if !normalizeResult.Valid {
		c.String(http.StatusForbidden, normalizeResult.ErrorMessage)
		return
	}
	rawPath = normalizeResult.NormalizedURL

	// 步骤5：分发到对应的处理器
	ProxyGitHubRequest(c, rawPath)
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
