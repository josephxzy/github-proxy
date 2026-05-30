package network

import (
	"net/http"
	"strconv"
	"strings"
)

// blockedContentTypes 被阻止的内容类型白名单。
// 这些类型的响应将被拒绝代理，防止滥用 GitHub 代理服务加速网页内容。
var blockedContentTypes = map[string]bool{
	"text/html":             true, // HTML 页面（最常见的误用场景）
	"application/xhtml+xml": true, // XHTML 页面
	"text/xml":              true, // XML 文档
	"application/xml":       true, // XML 数据
}

// IsBlockedContentType 检查给定的 Content-Type 是否在被阻止列表中。
// 解析 Content-Type 时去除参数部分（如 "text/html; charset=utf-8" → "text/html"），
// 并转为小写后进行匹配。
func IsBlockedContentType(contentType string) bool {
	return blockedContentTypes[strings.ToLower(strings.Split(contentType, ";")[0])]
}

// CleanSecurityHeaders 清除可能干扰代理的安全相关响应头。
// GitHub 服务器会在响应中设置这些头：
//   - Content-Security-Policy: 限制资源加载来源，可能阻止代理页面正常工作
//   - Referrer-Policy: 控制 Referer 发送策略
//   - Strict-Transport-Security: 强制 HTTPS（由我们自己或 nginx 处理即可）
//
// 移除这些头可以避免与代理服务的安全策略冲突。
func CleanSecurityHeaders(header http.Header) {
	header.Del("Content-Security-Policy")
	header.Del("Referrer-Policy")
	header.Del("Strict-Transport-Security")
}

// GetRealHost 从请求中提取真实的主机地址。
// 用于脚本文件 URL 替换时确定代理服务器的真实可访问地址。
//
// 优先级:
//  1. X-Forwarded-Host → 反向代理设置的原始 Host（最准确）
//  2. r.Host → 直接从请求行获取（无反代时使用）
//
// 返回值始终包含 https:// 前缀（如果尚未包含的话）。
// 因为脚本中的 URL 替换需要完整的 scheme://host 格式。
func GetRealHost(r *http.Request) string {
	realHost := r.Header.Get("X-Forwarded-Host")
	if realHost == "" {
		realHost = r.Host
	}
	if !strings.HasPrefix(realHost, "http://") && !strings.HasPrefix(realHost, "https://") {
		realHost = "https://" + realHost
	}
	return realHost
}

// CheckFileSize 检查响应声明的文件大小是否超过配置的限制。
// 从 Content-Length 头中解析字节数并与 maxSize 比较。
//
// 参数:
//   - contentLength: HTTP 响应头的 Content-Length 值（字符串形式）
//   - maxSize: 配置的最大允许文件大小（字节）
//
// 返回值:
//   - bool: true 表示文件过大，应返回 413 错误
//   - string: 超大时的错误描述信息（含人类可读的大小单位）
//   - bool=false, string="" : 文件大小合法或无法解析（Content-Length 为空或非法时不拦截）
func CheckFileSize(contentLength string, maxSize int64) (bool, string) {
	if contentLength == "" {
		return false, ""
	}
	size, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return false, ""
	}
	if size > maxSize {
		return true, "文件过大，限制大小: " + strconv.FormatInt(maxSize/(1024*1024), 10) + " MB"
	}
	return false, ""
}
