package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	ghproxyservice "github-proxy/internal/service/github"
	"github-proxy/pkg/network"

	"github.com/gin-gonic/gin"
)

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
		c.String(http.StatusLoopDetected, "重定向次数过多，可能存在循环重定向")
		return
	}

	ctx := c.Request.Context()

	// 检查 API 请求队列，实现速率限制
	if err := ghproxyservice.CheckAPIQueue(ctx, u); err != nil {
		// 如果客户端已断开连接，直接返回
		if ctx.Err() != nil {
			return
		}
		c.String(http.StatusGatewayTimeout, "API 请求排队超时，请稍后重试")
		return
	}

	// 创建转发到 GitHub 的 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, u, c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
		return
	}

	// 复制原始请求的所有头部信息
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	// 删除 Host 头，让 Go 自动设置正确的 Host
	req.Header.Del("Host")

	if c.Request.ContentLength > 0 {
		req.ContentLength = c.Request.ContentLength
	} else if cl := c.Request.Header.Get("Content-Length"); cl != "" {
		if size, err := strconv.ParseInt(cl, 10, 64); err == nil && size > 0 {
			req.ContentLength = size
			req.Header.Del("Content-Length")
		}
	}

	// 应用用户提供的 GitHub Token（优先级高于服务器 Token）
	if userToken, ok := c.Get("userToken"); ok {
		ghproxyservice.ApplyUserToken(req, userToken.(string))
	}
	// 如果配置了 GitHub Token，则添加到请求头中以提高速率限制
	ghproxyservice.ApplyGitHubToken(req, u)

	// 发送请求到 GitHub
	resp, err := network.GetGlobalHTTPClient().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
		return
	}

	// 用户 token 失效时，用服务器 token 重试一次（仅 GET/HEAD，避免 body 已消费）
	_, hasUserToken := c.Get("userToken")
	if hasUserToken && (resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden) && (c.Request.Method == "GET" || c.Request.Method == "HEAD") {
		resp.Body.Close()
		req2, err := http.NewRequestWithContext(ctx, c.Request.Method, u, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
			return
		}
		for key, values := range c.Request.Header {
			for _, value := range values {
				req2.Header.Add(key, value)
			}
		}
		req2.Header.Del("Host")
		ghproxyservice.ApplyGitHubToken(req2, u)
		resp, err = network.GetGlobalHTTPClient().Do(req2)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
			return
		}
		c.Header("X-Token-Status", "invalid")
	}
	defer resp.Body.Close()
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
