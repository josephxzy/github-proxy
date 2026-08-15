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
// 自动重新向原 URL 发起请求，从断点继续读取，保证对外输出的字节流连续
// 不缺失。对上层（水位线缓冲、消费者、浏览器）完全透明。
//
// 两种续传策略：
//
//  1. Range 续传（默认）：以 Range: bytes=<已读绝对偏移>- 重连，服务器
//     返回 206 + Content-Range，校验起始偏移一致后无缝切换数据源。
//     适用于支持 Range 的上游（release CDN、raw.githubusercontent.com）。
//
//  2. Skip 续传（skipResume=true，仅归档 URL）：codeload.github.com 等
//     上游忽略 Range 请求头（一律返回 200 全量）。此时整包重拉，丢弃
//     新连接开头的 <已读偏移> 字节后再继续输出——字节流仍连续，代价是
//     重复下载已传部分。归档内容按 commit ref 不可变，跳过安全。
//
// 关键不变式：输出字节流必须连续。重连成功必须满足：
//   - 新响应为 206 Partial Content 且 Content-Range 起始位置与已读偏移一致；
//     或（skip 续传）新响应为 200 且总大小与初始响应一致（若已知）
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
	skipResume bool // 上游忽略 Range（如 codeload）时，以"整包重拉 + 跳过已发字节"续传
}

// NewReconnectingReader 创建自动重连读取器。
// resp 为初始响应，其 Body 将作为首个数据源。
// 初始响应为 206 时通过 Content-Range 解析 base（绝对偏移）与 total（总大小）。
// skipResume 仅对忽略 Range 的归档上游（codeload）开启，见类型注释。
func NewReconnectingReader(ctx context.Context, url string, headers http.Header, resp *http.Response, skipResume bool) *ReconnectingReader {
	r := &ReconnectingReader{
		ctx:        ctx,
		url:        url,
		headers:    headers.Clone(),
		body:       resp.Body,
		maxRetries: 3,
		skipResume: skipResume,
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

// reconnect 重新请求数据源并校验连续性后切换。
// 成功返回 true，失败返回 false（内部已关闭失败响应的 Body）。
//
// 策略优先级：
//  1. 206 Partial Content：Range 续传（偏移校验）
//  2. 200 OK（且启用 skipResume）：服务器忽略 Range（如 codeload），
//     整包重拉并丢弃开头 from 字节，输出字节流保持连续
//  3. 其他（416 文件变化等）：放弃
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

	// 策略 1：206 Range 续传
	if resp.StatusCode == http.StatusPartialContent {
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

	// 策略 2：200 + skipResume（服务器忽略 Range，如 codeload）
	if r.skipResume && resp.StatusCode == http.StatusOK {
		if r.total > 0 && resp.ContentLength > 0 && resp.ContentLength != r.total {
			// 总大小变化（ref 已变），无法保证字节流连续，放弃
			resp.Body.Close()
			return false
		}
		// 丢弃新连接开头的 from 字节：这些内容客户端已收到。
		// 阻塞期间客户端无数据流入（上游在补传已发部分），
		// 但完成后字节流无缝衔接，下载不中断。
		if _, err := io.CopyN(io.Discard, resp.Body, from); err != nil {
			resp.Body.Close()
			return false
		}
		r.body.Close()
		r.body = resp.Body
		r.retries++
		return true
	}

	// 策略 3：其他状态（416 文件变化等），放弃
	resp.Body.Close()
	return false
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
