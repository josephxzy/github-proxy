package network

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestIsBlockedContentType 内容类型黑名单匹配（含参数与大小写）。
func TestIsBlockedContentType(t *testing.T) {
	tests := []struct {
		contentType string
		want        bool
	}{
		{"text/html", true},
		{"text/html; charset=utf-8", true},
		{"Text/HTML; charset=utf-8", true},
		{"application/xhtml+xml", true},
		{"text/xml", true},
		{"application/xml", true},
		{"application/octet-stream", false},
		{"application/zip", false},
		{"image/png", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := IsBlockedContentType(tt.contentType); got != tt.want {
			t.Errorf("IsBlockedContentType(%q) = %v, want %v", tt.contentType, got, tt.want)
		}
	}
}

// TestCleanSecurityHeaders 应移除可能干扰代理的安全响应头。
func TestCleanSecurityHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Content-Security-Policy", "default-src 'none'")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("Strict-Transport-Security", "max-age=63072000")
	h.Set("Cache-Control", "public, max-age=60")
	h.Set("Content-Type", "application/json")

	CleanSecurityHeaders(h)

	for _, key := range []string{"Content-Security-Policy", "Referrer-Policy", "Strict-Transport-Security"} {
		if v := h.Get(key); v != "" {
			t.Errorf("CleanSecurityHeaders 未移除 %s: %q", key, v)
		}
	}
	if h.Get("Cache-Control") == "" || h.Get("Content-Type") == "" {
		t.Error("CleanSecurityHeaders 不应移除普通响应头")
	}
}

// TestCheckFileSize 文件大小限制检查。
func TestCheckFileSize(t *testing.T) {
	const max = 1000

	tests := []struct {
		name          string
		contentLength string
		wantExceeded  bool
	}{
		{"空 CL 不拦截", "", false},
		{"非法 CL 不拦截", "abc", false},
		{"未超限", "999", false},
		{"恰好等于上限", "1000", false},
		{"超限", "1001", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exceeded, msg := CheckFileSize(tt.contentLength, max)
			if exceeded != tt.wantExceeded {
				t.Errorf("CheckFileSize(%q, %d) exceeded = %v, want %v", tt.contentLength, max, exceeded, tt.wantExceeded)
			}
			if exceeded && msg == "" {
				t.Error("超限时应返回错误描述")
			}
		})
	}
}

// TestGetRealHost 真实 Host 提取（优先 X-Forwarded-Host，自动补 https://）。
func TestGetRealHost(t *testing.T) {
	tests := []struct {
		name      string
		forwarded string
		host      string
		want      string
	}{
		{"X-Forwarded-Host 优先", "proxy.example.com", "127.0.0.1:5000", "https://proxy.example.com"},
		{"X-Forwarded-Host 带 scheme", "http://proxy.example.com", "127.0.0.1", "http://proxy.example.com"},
		{"回退到 r.Host", "", "proxy.example.com", "https://proxy.example.com"},
		{"r.Host 带端口", "", "example.com:8443", "https://example.com:8443"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Host = tt.host
			if tt.forwarded != "" {
				req.Header.Set("X-Forwarded-Host", tt.forwarded)
			}
			if got := GetRealHost(req); got != tt.want {
				t.Errorf("GetRealHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestParseContentRange Content-Range 头解析（reconnecting_reader 内部函数）。
func TestParseContentRange(t *testing.T) {
	tests := []struct {
		header    string
		wantStart int64
		wantTotal int64
		wantOK    bool
	}{
		{"bytes 0-99/1000", 0, 1000, true},
		{"bytes 500-999/1000", 500, 1000, true},
		{"bytes 500-999/*", 0, 0, false}, // 总大小未知：整体解析失败
		{"bytes */1000", 0, 0, false},    // 起始未知
		{"invalid", 0, 0, false},
		{"", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			start, total, ok := parseContentRange(tt.header)
			if ok != tt.wantOK || start != tt.wantStart || total != tt.wantTotal {
				t.Errorf("parseContentRange(%q) = (%d, %d, %v), want (%d, %d, %v)",
					tt.header, start, total, ok, tt.wantStart, tt.wantTotal, tt.wantOK)
			}
		})
	}
}

// TestHandleRedirectLocation 重定向 Location 处理。
func TestHandleRedirectLocation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 本地 URL 匹配函数：含 github.com 视为内部 GitHub URL
	checkURL := func(u string) []string {
		if strings.Contains(u, "github.com") {
			return []string{"owner", "repo"}
		}
		return nil
	}

	// GitHub URL 内部重定向：改写为代理相对路径，无需继续递归代理
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	loc, needRedirect := HandleRedirectLocation(c, "https://github.com/owner/repo/releases/download/v1/x.zip", checkURL)
	if needRedirect {
		t.Error("GitHub 内部重定向应返回 needRedirect=false")
	}
	if got := w.Header().Get("Location"); got != "/https://github.com/owner/repo/releases/download/v1/x.zip" {
		t.Errorf("Location = %q, want 代理相对路径", got)
	}
	_ = loc

	// 外部 URL：保持原样并返回 needRedirect=true（继续递归代理）
	_, needRedirect = HandleRedirectLocation(nil, "https://objects.githubusercontent.com/abc/def", checkURL)
	if !needRedirect {
		t.Error("外部重定向应返回 needRedirect=true")
	}
}
