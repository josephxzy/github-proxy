package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// TestNormalizeIP IP 归一化：IPv4 原样，IPv6 保留 /64 前缀。
func TestNormalizeIP(t *testing.T) {
	tests := []struct {
		ip   string
		want string
	}{
		{"1.2.3.4", "1.2.3.4"},
		{"203.0.113.55", "203.0.113.55"},
		{"2001:db8:85a3:1:0:8a2e:370:7334", "2001:db8:85a3:1::"}, // 低 64 位清零
		{"2001:db8:85a3:2::1", "2001:db8:85a3:2::"},
		{"invalid-ip", "invalid-ip"}, // 无法解析原样返回
	}
	for _, tt := range tests {
		if got := normalizeIP(tt.ip); got != tt.want {
			t.Errorf("normalizeIP(%q) = %q, want %q", tt.ip, got, tt.want)
		}
	}
}

// TestIsFrontendPath 前端静态资源路径应豁免 IP 限流。
func TestIsFrontendPath(t *testing.T) {
	exempt := []string{
		"/",
		"/ready",
		"/favicon.ico",
		"/favicon.svg",
		"/assets/index-abc.js",
		"/public/index.html",
	}
	limited := []string{
		"/owner/repo/releases/download/v1/x.zip",
		"/api/repos/o/r",
		"/api.github.com/repos/o/r",
	}
	for _, p := range exempt {
		if !isFrontendPath(p) {
			t.Errorf("isFrontendPath(%q) = false, want true", p)
		}
	}
	for _, p := range limited {
		if isFrontendPath(p) {
			t.Errorf("isFrontendPath(%q) = true, want false", p)
		}
	}
}

// TestIPRateLimiterAllow 固定窗口内的计数与超限拒绝。
func TestIPRateLimiterAllow(t *testing.T) {
	l := NewIPRateLimiter(2) // 每小时每 IP 2 次

	if !l.allow("1.2.3.4") {
		t.Error("第 1 次应放行")
	}
	if !l.allow("1.2.3.4") {
		t.Error("第 2 次应放行")
	}
	if l.allow("1.2.3.4") {
		t.Error("第 3 次应被拒绝")
	}
	// 不同 IP 独立计数
	if !l.allow("5.6.7.8") {
		t.Error("其他 IP 应放行")
	}
}

// TestIPRateLimiterWindowReset 窗口过期后计数重置。
func TestIPRateLimiterWindowReset(t *testing.T) {
	l := NewIPRateLimiter(1)
	if !l.allow("1.2.3.4") {
		t.Fatal("第 1 次应放行")
	}
	if l.allow("1.2.3.4") {
		t.Fatal("窗口内第 2 次应被拒绝")
	}

	// 模拟窗口过期
	l.mu.Lock()
	l.entries["1.2.3.4"].windowStart = time.Now().Add(-2 * time.Hour)
	l.mu.Unlock()

	if !l.allow("1.2.3.4") {
		t.Error("窗口过期后应重新放行")
	}
}

// TestIPRateLimiterCleanup 过期条目应被清理。
func TestIPRateLimiterCleanup(t *testing.T) {
	l := NewIPRateLimiter(10)
	l.allow("1.2.3.4")

	// 把条目改为 3 小时前（超过 cleanupAge=2h）
	l.mu.Lock()
	l.entries["1.2.3.4"].windowStart = time.Now().Add(-3 * time.Hour)
	l.mu.Unlock()

	l.cleanup()

	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.entries["1.2.3.4"]; ok {
		t.Error("过期条目未被清理")
	}
}

// TestIPRateLimiterDisabled limit<=0 时中间件应完全放行
// （生产路径由 Middleware 的 limit<=0 早退保护；allow 本身对既有 IP
// 第二次调用会拒绝，但那不是生产可达路径）。
func TestIPRateLimiterDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("authenticated", false)
		c.Next()
	})
	r.Use(NewIPRateLimiter(0).Middleware())
	r.GET("/owner/repo/releases/download/v1/x.zip", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/owner/repo/releases/download/v1/x.zip", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	// 同一 IP 连续多次都应放行
	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("第 %d 次请求应放行, got %d", i+1, w.Code)
		}
	}
}

// TestIPRateLimiterMiddleware 中间件在超限时返回 429。
func TestIPRateLimiterMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 构造带认证中间件的上下文：authenticated=false，路径非前端
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("authenticated", false)
		c.Next()
	})
	l := NewIPRateLimiter(1)
	r.Use(l.Middleware())
	r.GET("/owner/repo/releases/download/v1/x.zip", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/owner/repo/releases/download/v1/x.zip", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req)
	if w1.Code != http.StatusOK {
		t.Errorf("第 1 次请求应放行, got %d", w1.Code)
	}

	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("第 2 次请求应 429, got %d", w2.Code)
	}
}
