package download

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github-proxy/internal/network"
)

// PrefetchContentLength 文件总大小探测。
// 通过 HTTP Range 协议探测（适用于 Release Asset / Raw）。
//
// 探测失败时返回 0，调用方降级为 chunked 传输（无进度条但可正常下载）。
// 注意：宁可返回 0 也绝不返回估算值——错误的 Content-Length 会导致
// 客户端在下载末尾因字节数不符而判定"网络错误"，比没有进度条更糟。
//
// Archive（codeload）链接无法预知大小：codeload 响应为 chunked 且无
// Content-Length，GitHub API 中也不包含源码包大小（release asset 的
// size 与自动生成的源码压缩包无关），因此不做任何估算。
func PrefetchContentLength(ctx context.Context, url string, headers http.Header) int64 {
	return RangeProbeSize(ctx, url, headers)
}

// RangeProbeSize 使用 HTTP Range 协议探测文件大小。
// 发送 Range: bytes=0-0 请求，期望服务器返回 206 Partial Content，
// 并在 Content-Range 响应头中包含文件总大小。
func RangeProbeSize(ctx context.Context, url string, headers http.Header) int64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("Range", "bytes=0-0")
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Del("Host")
	ApplyGitHubToken(req, url)

	resp, err := network.GetGlobalHTTPClient().Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusPartialContent:
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if parts := strings.Split(cr, "/"); len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil && total > 0 {
					return total
				}
			}
		}
	case http.StatusOK:
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if size, err := strconv.ParseInt(cl, 10, 64); err == nil && size > 0 {
				return size
			}
		}
	}
	return 0
}
