package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// TestNormalizePath 去除 URI 开头的斜杠。
func TestNormalizePath(t *testing.T) {
	tests := []struct {
		uri  string
		want string
	}{
		{"", ""},
		{"/owner/repo/raw/main/a.sh", "owner/repo/raw/main/a.sh"},
		{"//owner/repo", "owner/repo"},
		{"https://github.com/owner/repo", "https://github.com/owner/repo"},
	}
	for _, tt := range tests {
		if got := normalizePath(tt.uri); got != tt.want {
			t.Errorf("normalizePath(%q) = %q, want %q", tt.uri, got, tt.want)
		}
	}
}

// TestBuildUpstreamRequest 上游请求构造：头复制、Host 移除、Content-Length 透传、
// 用户 Token 应用。
func TestBuildUpstreamRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/owner/repo.git/git-upload-pack", strings.NewReader("pack-data"))
	c.Request.Header.Set("User-Agent", "git/2.40")
	c.Request.Header.Set("Host", "proxy.example.com") // 应被移除
	c.Request.Header.Set("Content-Length", "9")
	c.Set("userToken", "ghp_userToken123")

	req, err := buildUpstreamRequest(c, "POST", "https://github.com/owner/repo.git/git-upload-pack", c.Request.Body)
	if err != nil {
		t.Fatalf("buildUpstreamRequest 出错: %v", err)
	}

	// 请求头复制
	if got := req.Header.Get("User-Agent"); got != "git/2.40" {
		t.Errorf("User-Agent = %q, want git/2.40", got)
	}
	// Host 移除（由 Go 按目标 URL 设置）
	if got := req.Header.Get("Host"); got != "" {
		t.Errorf("Host 应被移除, got %q", got)
	}
	// 用户 Token 应用
	if got := req.Header.Get("Authorization"); got != "token ghp_userToken123" {
		t.Errorf("Authorization = %q, want token ghp_userToken123", got)
	}
	// 请求方法与 URL
	if req.Method != "POST" || req.URL.String() != "https://github.com/owner/repo.git/git-upload-pack" {
		t.Errorf("method/url 错误: %s %s", req.Method, req.URL)
	}
	// Content-Length 透传（httptest 对 strings.Reader body 自动设置 9）
	if req.ContentLength != 9 {
		t.Errorf("ContentLength = %d, want 9", req.ContentLength)
	}
	// 请求体透传
	body, _ := io.ReadAll(req.Body)
	if string(body) != "pack-data" {
		t.Errorf("body = %q, want pack-data", body)
	}
}

// TestBuildUpstreamRequestNoUserToken 无用户 Token 且 URL 非 API 域名时
// 不应设置 Authorization。
// 注：服务器 Token 兜底分支（api.github.com + 无用户 token + 全局配置了
// GITHUB_TOKEN）依赖全局 config 单例，属集成行为，不在此单测覆盖。
func TestBuildUpstreamRequestNoUserToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/owner/repo/raw/main/a.sh", nil)

	req, err := buildUpstreamRequest(c, "GET", "https://github.com/owner/repo/raw/main/a.sh", nil)
	if err != nil {
		t.Fatalf("buildUpstreamRequest 出错: %v", err)
	}
	if got := req.Header.Get("Authorization"); got != "" {
		t.Errorf("无用户 token 时不应设置 Authorization, got %q", got)
	}
}

// TestBuildUpstreamRequestInvalidURL 非法 URL 应返回错误并写入 500。
func TestBuildUpstreamRequestInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/bad", nil)

	_, err := buildUpstreamRequest(c, "GET", "http://[::1]:namedport/", nil)
	if err == nil {
		t.Fatal("非法 URL 应返回错误")
	}
	if w.Code != http.StatusInternalServerError {
		t.Errorf("应写入 500, got %d", w.Code)
	}
}
