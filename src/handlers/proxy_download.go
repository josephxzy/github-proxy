package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github-proxy/config"
	ghproxyservice "github-proxy/internal/service/github"
	"github-proxy/pkg/network"

	"github.com/gin-gonic/gin"
)

// proxyDownloadRequest 文件下载代理的核心函数。
// 专门处理 releases/archive/blob/raw/gist 等文件下载类请求。
// 实现了从请求创建到数据流式传输的完整流水线。
//
// 处理流程:
//
//  1. 创建 HTTP 请求，复制客户端请求头
//  2. 快速模式检测：主分支 zip 等小文件可跳过预检以提升响应速度
//  3. 断点续传检测：根据客户端 Range 头决定是否跳过预检
//  4. 发送请求到 GitHub（跟随重定向）
//  5. 安全检查：内容类型过滤 + 文件大小限制
//  6. 根据资源类型分发:
//     - 脚本文件 (.sh/.ps1) → handleScriptResponse（URL 替换处理）
//     - 普通文件          → handleNormalResponse（直接转发）
//
// 双模式下载设计：
//
//	快速模式 (fast=true):
//	  → 跳过 Range 预检 → 无进度条但响应更快
//	  → 适用场景：主分支 zip、小文件下载
//
//	标准模式 (fast=false):
//	  → 并发 Range 预检 + 实际下载
//	  → 预检结果用于设置 Content-Length=总大小 → 浏览器显示进度条
//	  → 适用场景：大文件 Release 下载、需要进度反馈的场景
//
// 断点续传与首次下载的解耦设计：
//
//	客户端无 Range (首次下载):
//	  → 非快速模式：并发发起 Range 探测 + 实际下载请求
//	  → 探测结果用于设置 Content-Length=总大小 → 浏览器显示进度条
//
//	客户端有 Range (断点续传):
//	  → 跳过预检 → 透传 GitHub → Content-Length=剩余大小 → 续传数据正确
//
// 参数:
//   - c: Gin 上下文
//   - u: 目标 GitHub 文件下载 URL
//   - redirectCount: 重定向深度计数器（防止循环）
func proxyDownloadRequest(c *gin.Context, u string, redirectCount int) {
	const maxRedirects = 20
	// 防止循环重定向
	if redirectCount > maxRedirects {
		c.String(http.StatusLoopDetected, "重定向次数过多，可能存在循环重定向")
		return
	}

	ctx := c.Request.Context()

	// 创建转发请求
	req, err := http.NewRequestWithContext(ctx, c.Request.Method, u, c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
		return
	}

	// 复制原始请求头
	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Del("Host")

	// 应用 GitHub Token（如果配置了）
	ghproxyservice.ApplyGitHubToken(req, u)

	cfg := config.GetConfig()
	startTime := time.Now()

	// 断点续传与首次下载解耦设计：
	//   - 客户端带 Range → 断点续传路径（跳过预检，透传 GitHub 的 CL）
	//   - 快速模式 (fast=1) → 跳过预检，优先响应速度（无进度条）
	//   - 普通模式 → Range 探测与实际下载并发执行（有进度条）
	isRangeRequest := c.Request.Header.Get("Range") != ""
	isFastMode := c.Query("fast") == "1"
	var preflightSize int64
	preflightCh := make(chan int64, 1)

	// 对于非断点续传、非快速模式的 GET 请求，启动并发的 Range 预检
	// 用于提前获取文件总大小，使浏览器能显示进度条
	if !isRangeRequest && !isFastMode && c.Request.Method == "GET" {
		go func() {
			preflightCh <- ghproxyservice.PrefetchContentLength(ctx, u, c.Request.Header)
		}()
	}

	// 发送实际下载请求到 GitHub
	resp, err := network.GetGlobalHTTPClient().Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		latency := int(time.Since(startTime).Milliseconds())
		c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v (latency=%dms)", err, latency))
		return
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
		}
	}()

	// 等待并发预检结果（与实际下载请求并行，减少浏览器等待时间）
	// 快速模式和断点续传模式跳过预检等待
	if !isRangeRequest && !isFastMode && c.Request.Method == "GET" {
		select {
		case preflightSize = <-preflightCh:
		case <-ctx.Done():
			// 客户端已断开，不再等待
		}
	}

	latency := int(time.Since(startTime).Milliseconds())

	// 安全检查1：内容类型过滤
	// 阻止代理网页类型的内容（text/html），仅允许文件下载
	if c.Request.Method == "GET" && network.IsBlockedContentType(resp.Header.Get("Content-Type")) {
		c.JSON(http.StatusForbidden, map[string]string{
			"error":   "Content type not allowed",
			"message": "检测到网页类型，本服务不支持加速网页，请检查您的链接是否正确",
		})
		return
	}

	// 安全检查2：文件大小限制
	// 防止下载超大文件导致服务器资源耗尽
	if exceeded, msg := network.CheckFileSize(resp.Header.Get("Content-Length"), cfg.Server.FileSize); exceeded {
		c.String(http.StatusRequestEntityTooLarge, msg)
		return
	}

	// 清除安全响应头
	network.CleanSecurityHeaders(resp.Header)

	// 获取真实的主机名（用于脚本中的 URL 替换）
	realHost := network.GetRealHost(c.Request)

	// 根据资源类型分发处理
	if ghproxyservice.IsScriptURL(u) {
		// 脚本文件需要特殊处理（URL 替换）
		handleScriptResponse(c, resp, realHost, latency, redirectCount)
	} else {
		// 普通文件直接转发
		handleNormalResponse(c, resp, preflightSize, isRangeRequest, latency, redirectCount)
	}
}

// getAuthFromContext 从 Gin Context 中提取认证状态。
// 该值由 GitHubProxyHandler 在身份验证后设置。
func getAuthFromContext(c *gin.Context) bool {
	if v, ok := c.Get("authenticated"); ok {
		return v.(bool)
	}
	return false
}

// handleScriptResponse 处理脚本文件 (.sh/.ps1) 的响应。
// 脚本文件需要特殊处理——它们内部可能包含原始的 github.com URL。
// 如果不替换，脚本执行时会直接访问 GitHub（可能被墙或慢）。
//
// 处理流程:
//  1. 检测响应是否为 gzip 压缩
//  2. 使用 ProcessSmart 解压→替换 URL→重新压缩（如需要）
//  3. 内容变化时删除 CL/CE，改为 chunked 传输
//  4. 调用 writeResponse 发送到客户端
func handleScriptResponse(c *gin.Context, resp *http.Response, realHost string, latency int, redirectCount int) {
	// 检测是否为 gzip 压缩
	isGzipCompressed := resp.Header.Get("Content-Encoding") == "gzip"

	// 使用智能处理器解压、替换 URL、重新压缩
	processedBody, processedSize, err := ghproxyservice.ProcessSmart(resp.Body, isGzipCompressed, realHost)
	if err != nil {
		c.String(http.StatusBadGateway, "Script processing failed: %v", err)
		return
	}

	// 如果内容发生了变化（URL 被替换），更新响应头
	if processedSize > 0 {
		resp.Header.Del("Content-Length")               // 内容长度已改变
		resp.Header.Del("Content-Encoding")             // 编码方式可能已改变
		resp.Header.Set("Transfer-Encoding", "chunked") // 改用分块传输
	}

	// 写入响应到客户端
	writeResponse(c, resp, processedBody, processedSize, latency, redirectCount)
}

// handleNormalResponse 处理普通文件（非脚本）的响应。
// 核心职责是确定要发送给客户端的 Content-Length 值，
// 这是断点续传与首次下载解耦的关键节点。
//
// 三种策略按优先级选择:
//
//	isRangeRequest=true  (断点续传):
//	  → 忽略 preflightSize，使用 GitHub 返回的 Content-Length（本次传输剩余部分的大小）
//	  → 确保客户端收到的字节数与 CL 一致，续传不会卡死/报错
//
//	isRangeRequest=false 且 preflightSize>0  (首次下载，预检成功):
//	  → 使用预检得到的总大小
//	  → 浏览器据此显示下载进度条
//
//	上述都不满足 (降级):
//	  → 从响应头中尽力解析：CL 或 Content-Length 字段 或 Content-Range 中的总大小
//	  → 若全部失败则 knownSize=0，走 chunked 传输（无进度但可正常下载）
//
// 参数:
//   - preflightSize: Range 预检获得的文件总大小（仅首次下载有效）
//   - isRangeRequest: 客户端是否携带了 Range 请求头（断点续传标志）
func handleNormalResponse(c *gin.Context, resp *http.Response, preflightSize int64, isRangeRequest bool, latency int, redirectCount int) {
	var knownSize int64

	if isRangeRequest {
		// 断点续传：直接使用 GitHub 返回的 Content-Length（剩余部分大小）
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if size, err := strconv.ParseInt(cl, 10, 64); err == nil && size > 0 {
				knownSize = size
			}
		}
	} else {
		// 首次下载：按优先级尝试多种方式获取文件总大小
		respCL := resp.Header.Get("Content-Length")
		if respCL != "" {
			// 方式1：从 Content-Length 响应头获取
			if size, err := strconv.ParseInt(respCL, 10, 64); err == nil && size > 0 {
				knownSize = size
			}
		} else if resp.ContentLength > 0 {
			// 方式2：从 Go 的 http.Response.ContentLength 获取
			knownSize = resp.ContentLength
		} else if preflightSize > 0 {
			// 方式3：使用预检得到的大小
			knownSize = preflightSize
		} else if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
			// 方式4：从 Content-Range 头解析总大小（格式：bytes start-end/total）
			if parts := strings.Split(contentRange, "/"); len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil && total > 0 {
					knownSize = total
				}
			}
		}
	}

	// 写入响应到客户端
	writeResponse(c, resp, resp.Body, knownSize, latency, redirectCount)
}

// writeResponse 响应处理的最后阶段，负责将数据发送给客户端。
// 执行以下操作:
//  1. 重定向处理：检查 Location 头，GitHub 重定向时递归调用自身
//  2. 状态码与响应头：复制 GitHub 的所有头信息到客户端响应
//  3. Content-Length 设置：仅在非 206 响应时设置（断点续传透传 GitHub 的值）
//  4. 流式传输：通过 streamWithLimit 将数据写入客户端（可能带带宽限制）
//
// 关于 206 与 Content-Length 的关键规则：
//
//	200 + knownSize>0  → 设置 CL=knownSize（首次下载，进度条正常）
//	206               → 不覆盖 CL（断点续传，使用 GitHub 返回的剩余大小）
//	chunked            → 不设置 CL（降级方案，无进度但可下载）
func writeResponse(c *gin.Context, resp *http.Response, body io.Reader, knownSize int64, latency int, redirectCount int) {
	// 处理重定向响应
	if location := resp.Header.Get("Location"); location != "" {
		if _, needRedirect := network.HandleRedirectLocation(c, location, ghproxyservice.MatchURL); needRedirect {
			proxyDownloadRequest(c, location, redirectCount+1)
			return
		}
	}

	// 设置 HTTP 状态码
	c.Status(resp.StatusCode)

	// 复制 GitHub 的所有响应头到客户端
	copyResponseHeaders(c, resp)

	// 仅对非 206（非断点续传）响应设置 Content-Length
	if knownSize > 0 && resp.StatusCode != http.StatusPartialContent {
		c.Header("Content-Length", strconv.FormatInt(knownSize, 10))
		c.Header("Cache-Control", "no-transform") // 防止中间节点修改内容
	}

	// 立即写入状态码和响应头
	c.Writer.WriteHeaderNow()

	// 开始流式传输数据
	transferStart := time.Now()
	bytesWritten := streamToClient(c, body)
	transferTime := time.Since(transferStart)

	// 这些变量可用于日志记录或监控（当前未使用）
	_ = latency
	_ = bytesWritten
	_ = transferTime
}

// copyResponseHeaders 将 GitHub 服务器的所有响应头逐个复制到客户端响应中。
// 保持原始响应的完整元数据（Content-Type、Cache-Control、ETag 等）。
// 注意：此函数在 writeResponse 中于 Content-Length 设置之前调用，
// 因此后续对 CL 的覆盖会生效（针对 200 响应）。
func copyResponseHeaders(c *gin.Context, resp *http.Response) {
	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}
}
