// Package network 提供 GitHub 代理的网络层基础设施：
//
//	HTTP 客户端 (http_client.go):
//	  - 全局复用的 http.Client 实例
//	  - 高性能连接池配置（1000 空闲连接，90s 超时）
//	  - 代理支持（HTTP_PROXY / HTTPS_PROXY 环境变量）
//	  - 禁用自动解压缩（保留原始 Content-Length 用于进度显示）
//
//	重定向处理 (redirect.go):
//	  - GitHub 重定向 URL 的认证前缀注入
//	  - 区分内部重定向（GitHub URL）和外部重定向
//
//	响应工具 (response.go):
//	  - 内容类型黑名单过滤（阻止 HTML/XML 代理）
//	  - 安全响应头清理（移除 CSP/HSTS 等）
//	  - 文件大小限制检查
//	  - 真实 Host 提取（支持反向代理场景）
package network

import (
	"net"
	"net/http"
	"os"
	"time"

	"github-proxy/config"
)

// globalHTTPClient 全局 HTTP 客户端实例。
// 所有对 GitHub 的出站请求共用此连接池，
// 避免频繁创建/销毁 TCP 连接的开销。
var globalHTTPClient *http.Client

// InitHTTPClients 初始化全局 HTTP 客户端。
// 应在应用启动时调用一次（main 函数中）。
//
// 配置项详解：
//
//	代理设置:
//	  如果 config.Access.Proxy 非空，设置 HTTP_PROXY 和 HTTPS_PROXY 环境变量。
//	  http.Client 自动通过代理服务器转发请求（适用于需要翻墙的场景）。
//
//	传输层参数:
//	  DisableCompression: true  → 关键！禁用 Go 默认的自动 gzip 解压。
//	                            因为解压后 Content-Length 会丢失，
//	                            导致无法向客户端报告文件大小（影响下载进度显示）。
//	  MaxIdleConns: 1000       → 最大空闲连接总数。高并发下避免连接重建开销。
//	  MaxIdleConnsPerHost: 1000 → 每个主机（如 github.com）最大空闲连接数。
//	                            GitHub 有多个 CDN 子域名，各占一部分配额。
//	  IdleConnTimeout: 90s      → 空闲连接存活时间。过长浪费资源，过短导致频繁握手。
//	  ResponseHeaderTimeout: 60s → 等待服务器响应头的最大时间。
//	                            防止 GitHub 无响应时连接永久挂起。
func InitHTTPClients() {
	cfg := config.GetConfig()

	if p := cfg.Access.Proxy; p != "" {
		os.Setenv("HTTP_PROXY", p)
		os.Setenv("HTTPS_PROXY", p)
	}

	globalHTTPClient = &http.Client{
		Transport: &http.Transport{
			Proxy:              http.ProxyFromEnvironment,
			DisableCompression: true,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 60 * time.Second,
		},
	}

}

// GetGlobalHTTPClient 获取全局 HTTP 客户端实例。
// 所有包通过此函数获取共享的 http.Client。
// 确保连接池统一管理。
func GetGlobalHTTPClient() *http.Client {
	return globalHTTPClient
}
