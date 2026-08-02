package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// IPRateLimiter 基于固定窗口的 IP 请求频率限制器。
type IPRateLimiter struct {
	mu         sync.Mutex
	entries    map[string]*ipEntry
	limit      int
	windowSize time.Duration
	cleanupAge time.Duration // 超过此时间未访问的条目被清理
}

// ipEntry IP 限流条目，记录固定时间窗口内的请求计数。
type ipEntry struct {
	count       int
	windowStart time.Time
}

// NewIPRateLimiter 创建 IP 限流器。
// limit: 每小时每 IP 最大请求数。0 表示不限流。
func NewIPRateLimiter(limit int) *IPRateLimiter {
	if limit <= 0 {
		return &IPRateLimiter{}
	}
	l := &IPRateLimiter{
		entries:    make(map[string]*ipEntry),
		limit:      limit,
		windowSize: time.Hour,
		cleanupAge: 2 * time.Hour,
	}
	go l.cleanupLoop()
	return l
}

// Middleware 返回 Gin 中间件，对每个请求执行 IP 限流检查。
// 以下请求豁免限流：
//   - 白名单 token 用户（authenticated）
//   - 前端静态资源路径（isFrontendPath）
//
// 超限时返回 429 并中止请求。
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if l.limit <= 0 {
			return
		}
		if getAuthFromContext(c) {
			return
		}
		if isFrontendPath(c.Request.URL.Path) {
			return
		}

		ip := normalizeIP(c.ClientIP())
		if !l.allow(ip) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "请求太频繁，请稍后再试",
			})
			return
		}
		c.Next()
	}
}

// allow 对指定 IP 执行固定窗口计数。
// 新 IP 或窗口过期时创建新窗口；否则递增计数并判断是否超限。
func (l *IPRateLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	entry, ok := l.entries[ip]
	if !ok || now.Sub(entry.windowStart) >= l.windowSize {
		l.entries[ip] = &ipEntry{count: 1, windowStart: now}
		return true
	}
	entry.count++
	return entry.count <= l.limit
}

// isFrontendPath 判断请求路径是否为前端静态资源（豁免 IP 限流）。
func isFrontendPath(path string) bool {
	return path == "/" ||
		path == "/ready" ||
		path == "/favicon.ico" ||
		strings.HasPrefix(path, "/assets/") ||
		strings.HasPrefix(path, "/public/")
}

// normalizeIP 归一化客户端 IP。
// IPv4 原样返回；IPv6 仅保留 /64 前缀（清零低 64 位），
// 避免同一用户的临时地址变动导致限流失效。
func normalizeIP(ip string) string {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return ip
	}
	if parsed.To4() != nil {
		return ip // IPv4 原样
	}
	// IPv6：仅保留 /64 前缀（清零低 64 位）
	mask := net.CIDRMask(64, 128)
	return parsed.Mask(mask).String()
}

// cleanupLoop 周期性地清理过期条目，防止 map 无限增长。
func (l *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(20 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		l.cleanup()
	}
}

// cleanup 删除超过 cleanupAge 未访问的条目。
func (l *IPRateLimiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	for ip, entry := range l.entries {
		if now.Sub(entry.windowStart) > l.cleanupAge {
			delete(l.entries, ip)
		}
	}
}
