package server

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github-proxy/internal/config"
	"github-proxy/internal/handlers"
	"github-proxy/internal/service"

	"github.com/gin-gonic/gin"
)

// fakeStaticFS 内存版 StaticFileSystem，用于测试静态文件服务。
type fakeStaticFS struct {
	files map[string][]byte
}

func (f *fakeStaticFS) ReadFile(name string) ([]byte, error) {
	if data, ok := f.files[name]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}

func (f *fakeStaticFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return nil, nil
}

// TestFormatUptime 运行时长格式化。
func TestFormatUptime(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "30 秒"},
		{90 * time.Second, "1 分钟"},
		{2 * time.Minute, "2 分钟"},
		{1*time.Hour + 5*time.Minute, "1 小时 5 分钟"},
		{25*time.Hour + 3*time.Hour, "1 天 4 小时"},
		{0, "0 秒"},
	}
	for _, tt := range tests {
		if got := formatUptime(tt.d); got != tt.want {
			t.Errorf("formatUptime(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

// TestDetectContentType 扩展名到 MIME 类型的映射。
func TestDetectContentType(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"index.html", "text/html; charset=utf-8"},
		{"app.js", "application/javascript"},
		{"style.css", "text/css"},
		{"favicon.ico", "image/x-icon"},
		{"logo.svg", "image/svg+xml"},
		{"data.json", "application/json"},
		{"pic.png", "image/png"},
		{"photo.jpg", "image/jpeg"},
		{"anim.gif", "image/gif"},
		{"font.woff2", "font/woff2"},
		{"unknown.xyz", "text/html; charset=utf-8"}, // 默认 HTML
	}
	for _, tt := range tests {
		if got := detectContentType(tt.filename); got != tt.want {
			t.Errorf("detectContentType(%q) = %q, want %q", tt.filename, got, tt.want)
		}
	}
}

// TestServeEmbedFile 静态文件服务：命中返回内容与缓存头，未命中返回 404。
func TestServeEmbedFile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	fsys := &fakeStaticFS{files: map[string][]byte{
		"public/index.html":    []byte("<html>hello</html>"),
		"public/assets/app.js": []byte("console.log(1)"),
	}}

	// 命中 HTML：no-cache 缓存策略
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	ServeEmbedFile(c, fsys, "public/index.html")
	if w.Code != http.StatusOK {
		t.Fatalf("index.html 应返回 200, got %d", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, "hello") {
		t.Errorf("body = %q", body)
	}
	if got := w.Header().Get("Cache-Control"); !strings.Contains(got, "no-cache") {
		t.Errorf("HTML 应 no-cache, got %q", got)
	}
	if got := w.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Errorf("Content-Type = %q", got)
	}

	// 命中 JS：immutable 长缓存
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	ServeEmbedFile(c2, fsys, "public/assets/app.js")
	if w2.Code != http.StatusOK {
		t.Fatalf("app.js 应返回 200, got %d", w2.Code)
	}
	if got := w2.Header().Get("Cache-Control"); !strings.Contains(got, "immutable") {
		t.Errorf("JS 应 immutable 缓存, got %q", got)
	}

	// 未命中：404（c.Status 未 flush 时 httptest.Recorder.Code 仍为默认 200，
	// 断言 writer 内部状态码，等价于真实请求中 gin 收尾 flush 后的结果）
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	ServeEmbedFile(c3, fsys, "public/missing.js")
	if got := c3.Writer.Status(); got != http.StatusNotFound {
		t.Errorf("缺失文件应返回 404, got %d", got)
	}
}

// TestBuildRouterReady /ready 健康检查路由返回版本与运行时长。
func TestBuildRouterReady(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	// TokenAuthMiddleware 依赖全局 Application，测试前注入
	handlers.SetApplication(service.NewApplication(cfg))

	start := time.Now().Add(-2 * time.Minute)
	router := BuildRouter(&RouterConfig{
		AppConfig:        cfg,
		Version:          "1.0.0-test",
		BuildTime:        "2025-01-01",
		ServiceStartTime: start,
		StaticFS:         &fakeStaticFS{files: map[string][]byte{}},
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/ready", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("/ready 应返回 200, got %d", w.Code)
	}
	body := w.Body.String()
	for _, want := range []string{`"ready":true`, `"version":"1.0.0-test"`, "uptime"} {
		if !strings.Contains(body, want) {
			t.Errorf("/ready body 缺少 %s: %s", want, body)
		}
	}
}

// TestBuildRouterNoRoute 未知路径走 GitHub 代理（不 404 而是代理处理）。
func TestBuildRouterNoRoute(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfg.Server.EnableFrontend = false
	handlers.SetApplication(service.NewApplication(cfg))

	router := BuildRouter(&RouterConfig{
		AppConfig:        cfg,
		Version:          "dev",
		BuildTime:        "now",
		ServiceStartTime: time.Now(),
		StaticFS:         &fakeStaticFS{files: map[string][]byte{}},
	})

	// 未注册路径会进入 NoRoute 的 GitHubProxyHandler，
	// URL 规范化失败时应返回 403（无效输入）而非 404。
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/not/a/github/path", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("非法代理路径应 403, got %d", w.Code)
	}
}
