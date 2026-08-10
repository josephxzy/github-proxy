package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// ReconnectingReader 支持自动重连的响应体读取器。
//
// 背景：浏览器暂停下载时连接保持存活，服务端停止读取 GitHub 响应体，
// 暂停时间过长后 GitHub/CDN 会掐断空闲连接。恢复下载时生产者再次
// Read 会报错，导致传输中途终止，浏览器只能等待自动重试或手动续传。
//
// 本读取器解决该问题：当底层连接在中途断开（非 EOF、非上下文取消）时，
// 自动以 Range: bytes=<已读绝对偏移>- 重新向原 URL 发起请求，从断点
// 继续读取，保证对外输出的字节流连续不缺失。对上层（水位线缓冲、
// 消费者、浏览器）完全透明。
//
// 关键不变式：输出字节流必须连续。重连成功必须满足：
//   - 新响应为 206 Partial Content
//   - Content-Range 的起始位置与已读偏移一致
//   - 文件总大小与初始响应一致（若已知）
//
// 任何校验失败都会放弃重连，返回原始错误（与当前行为一致，可优雅降级）。
type ReconnectingReader struct {
	ctx        context.Context
	url        string
	headers    http.Header
	body       io.ReadCloser
	base       int64 // 初始响应体对应的文件绝对起始偏移（206 时为 Content-Range 的 start，否则 0）
	total      int64 // 文件总大小（未知为 0）
	offset     int64 // 已从响应体产出的字节数（不含 base）
	retries    int
	maxRetries int
}

// NewReconnectingReader 创建自动重连读取器。
// resp 为初始响应，其 Body 将作为首个数据源。
// 初始响应为 206 时通过 Content-Range 解析 base（绝对偏移）与 total（总大小）。
func NewReconnectingReader(ctx context.Context, url string, headers http.Header, resp *http.Response) *ReconnectingReader {
	r := &ReconnectingReader{
		ctx:        ctx,
		url:        url,
		headers:    headers.Clone(),
		body:       resp.Body,
		maxRetries: 3,
	}
	if resp.StatusCode == http.StatusPartialContent {
		if start, total, ok := parseContentRange(resp.Header.Get("Content-Range")); ok {
			r.base = start
			r.total = total
		}
	} else if resp.ContentLength > 0 {
		r.total = resp.ContentLength
	}
	return r
}

// Read 读取数据。底层连接断开（非 EOF）时尝试自动重连续传。
func (r *ReconnectingReader) Read(p []byte) (int, error) {
	for {
		n, err := r.body.Read(p)
		r.offset += int64(n)
		if err == nil || err == io.EOF {
			return n, err
		}
		// 客户端已断开或重试耗尽：返回原始错误，让上层按原逻辑收尾
		if r.ctx.Err() != nil || r.retries >= r.maxRetries {
			return n, err
		}
		if !r.reconnect() {
			return n, err
		}
		if n > 0 {
			return n, nil
		}
		// n == 0 且重连成功：继续循环从新连接读取
	}
}

// Close 关闭当前数据源。
func (r *ReconnectingReader) Close() error {
	return r.body.Close()
}

// reconnect 以 Range: bytes=<base+offset>- 重新请求，校验偏移后切换数据源。
// 成功返回 true，失败返回 false（内部已关闭失败响应的 Body）。
func (r *ReconnectingReader) reconnect() bool {
	from := r.base + r.offset

	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, r.url, nil)
	if err != nil {
		return false
	}
	req.Header = r.headers.Clone()
	req.Header.Del("Host")
	req.Header.Del("Content-Length")
	req.Header.Del("Content-Type")
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", from))

	resp, err := GetGlobalHTTPClient().Do(req)
	if err != nil {
		return false
	}
	if resp.StatusCode != http.StatusPartialContent {
		// 200：服务器忽略 Range（如 codeload），返回全量数据会导致字节流重复
		// 416：文件已变化，无法从断点继续
		resp.Body.Close()
		return false
	}

	start, total, ok := parseContentRange(resp.Header.Get("Content-Range"))
	if !ok || start != from {
		// 偏移不一致，继续会造成数据重复或缺失，放弃重连
		resp.Body.Close()
		return false
	}
	if r.total > 0 && total > 0 && total != r.total {
		// 文件总大小变化，无法保证完整性，放弃重连
		resp.Body.Close()
		return false
	}

	r.body.Close()
	r.body = resp.Body
	r.retries++
	return true
}

// parseContentRange 解析 Content-Range 头，格式：bytes start-end/total。
// 返回起始偏移与总大小；解析失败返回 ok=false。
func parseContentRange(cr string) (start, total int64, ok bool) {
	slash := strings.IndexByte(cr, '/')
	if slash < 0 {
		return 0, 0, false
	}
	total, err := strconv.ParseInt(strings.TrimSpace(cr[slash+1:]), 10, 64)
	if err != nil || total <= 0 {
		return 0, 0, false
	}
	seg := cr[:slash]
	dash := strings.LastIndexByte(seg, '-')
	if dash < 0 {
		return 0, 0, false
	}
	startStr := strings.TrimSpace(seg[:dash])
	if i := strings.LastIndexByte(startStr, ' '); i >= 0 {
		startStr = startStr[i+1:]
	}
	start, err = strconv.ParseInt(startStr, 10, 64)
	if err != nil || start < 0 {
		return 0, 0, false
	}
	return start, total, true
}
