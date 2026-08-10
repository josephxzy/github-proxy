package handlers

import (
	"io"
	"net/http"

	ghproxyservice "github-proxy/internal/github"
	"github-proxy/internal/network"

	"github.com/gin-gonic/gin"
)

// waitAPIAccess 在请求将使用站内 token（消耗站内共享配额）时排队等待。
// 返回 true 表示已获得配额（或被豁免）；返回 false 表示排队超时/取消，调用方应中止请求。
//
// 计费规则：代理的共享 API 池只限制「站内 token」的消耗——
//   - 白名单 token 用户：豁免（authenticated=true，直接放行）
//   - 用户自带有效 token：首次请求走用户自己的配额（GitHub 侧计费），不占用共享池
//   - 用户 token 失效后改用站内 token 重试：视为一次站内消耗，重试前占用共享池
func waitAPIAccess(c *gin.Context, u string, authenticated bool) bool {
	ctx := c.Request.Context()
	if err := ghproxyservice.CheckAPIQueue(ctx, u, authenticated); err != nil {
		if ctx.Err() != nil {
			return false
		}
		c.String(http.StatusGatewayTimeout, "API 请求排队超时，请稍后重试")
		return false
	}
	return true
}

// proxyAPIRequest 处理 GitHub API 请求的代理函数。
// 专门处理 api.github.com/repos/... 类型的 JSON API 请求。
// 如获取 release 列表、tag 信息等。
//
// 与文件下载路径 (proxyDownloadRequest) 的区别：
//
//	特性          API 路径	  下载路径
//	Range 预检	      不执行       执行（获取总大小）
//	内容类型检查    跳过	       执行（阻止 HTML 代理）
//	文件大小限制	    跳过	       执行（防超大文件）
//	带宽限制	        无       有（未认证用户限速）
//	脚本 URL 替换	    不执行       执行 (.sh/.ps1)
//	传输方式	        io.Copy 直接转发  streamWithLimit 流式传输
//
// 参数:
//   - c: Gin 上下文，包含请求/响应对象
//   - u: 目标 GitHub API 的完整 URL
//   - redirectCount: 当前重定向深度，防止循环重定向（上限 20 次）
func proxyAPIRequest(c *gin.Context, u string, redirectCount int) {
	const maxRedirects = 20
	// 检查重定向次数，防止无限循环
	if redirectCount > maxRedirects {
		loopDetectedError(c)
		return
	}

	ctx := c.Request.Context()

	// 是否携带用户 token：携带则首次请求使用用户自己的配额（GitHub 侧计费），
	// 不消耗代理共享池；仅当用户 token 失效、改用站内 token 重试时才计费。
	_, hasUserToken := c.Get("userToken")

	// 未携带用户 token：请求将使用站内 token（或匿名），属于站内资源消耗，需占用共享配额排队
	if !hasUserToken {
		if !waitAPIAccess(c, u, false) {
			return
		}
	}

	// 创建转发到 GitHub 的 HTTP 请求（复用公共构造逻辑）
	req, err := buildUpstreamRequest(c, c.Request.Method, u, c.Request.Body)
	if err != nil {
		return
	}

	// 发送请求到 GitHub
	resp, err := network.GetGlobalHTTPClient().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		serverError(c, err)
		return
	}

	// 用户 token 失效时，用服务器 token 重试一次（仅 GET/HEAD，避免 body 已消费）
	if hasUserToken && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && (c.Request.Method == "GET" || c.Request.Method == "HEAD") {
		resp.Body.Close()

		// 重试将改用站内 token，视为一次站内资源消耗，重试前占用共享配额
		if !waitAPIAccess(c, u, false) {
			return
		}

		req2, err := buildUpstreamRequest(c, c.Request.Method, u, nil)
		if err != nil {
			return
		}
		// 重试不再携带用户 token：移除其 Authorization，改用服务器 token
		req2.Header.Del("Authorization")
		ghproxyservice.ApplyGitHubToken(req2, u)
		resp, err = network.GetGlobalHTTPClient().Do(req2)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			serverError(c, err)
			return
		}
		c.Header("X-Token-Status", "invalid")
	}
	defer resp.Body.Close()

	// 处理重定向：GitHub 可能返回 301/302 重定向
	if location := resp.Header.Get("Location"); location != "" {
		if _, needRedirect := network.HandleRedirectLocation(c, location, ghproxyservice.MatchURL); needRedirect {
			// 递归处理重定向，增加重定向计数
			proxyAPIRequest(c, location, redirectCount+1)
			return
		}
	}

	// 清除安全相关的响应头，避免泄露服务器信息
	network.CleanSecurityHeaders(resp.Header)

	// 设置状态码并复制响应头
	c.Status(resp.StatusCode)
	copyResponseHeaders(c, resp)
	c.Header("Cache-Control", "no-store")
	c.Header("Access-Control-Expose-Headers", "Link, X-RateLimit-Limit, X-RateLimit-Remaining, X-RateLimit-Reset, X-GitHub-Request-Id, X-Token-Status")
	c.Writer.WriteHeaderNow()

	// 直接将响应体流式复制到客户端（API 响应通常较小）
	io.Copy(c.Writer, resp.Body)
}
