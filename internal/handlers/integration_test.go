package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github-proxy/internal/config"
	"github-proxy/internal/network"
	"github-proxy/internal/service"

	"github.com/gin-gonic/gin"
)

// roundTripFunc 将普通函数适配为 http.RoundTripper，用于在测试中
// 把全局 HTTP 客户端的出站请求改写到本地 mock server。
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// mockGitHubRoundTripper 返回一个 RoundTripper：把所有出站请求的
// scheme/host 改写成 ts.URL（本地 mock），其余（路径、头、方法）原样透传。
func mockGitHubRoundTripper(ts *httptest.Server) http.RoundTripper {
	return roundTripFunc(func(req *http.Request) (*http.Response, error) {
		u := *req.URL
		u.Scheme = "http"
		u.Host = strings.TrimPrefix(ts.URL, "http://")
		req2 := req.Clone(req.Context())
		req2.URL = &u
		req2.Host = u.Host
		return http.DefaultTransport.RoundTrip(req2)
	})
}

// withMockGitHub 初始化全局 HTTP 客户端并把 Transport 指向 mock server，
// 测试结束时恢复原 Transport。返回 mock server 地址。
func withMockGitHub(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()

	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	network.InitHTTPClients()
	client := network.GetGlobalHTTPClient()
	if client == nil {
		t.Fatal("GetGlobalHTTPClient() 返回 nil")
	}
	oldTransport := client.Transport
	client.Transport = mockGitHubRoundTripper(ts)
	t.Cleanup(func() { client.Transport = oldTransport })

	return ts
}

// newProxyContext 构造带 Gin Context 的代理请求上下文。
// path 为代理请求路径（如 "/https://github.com/owner/repo/raw/main/a.txt"）。
func newProxyContext(method, path string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	c.Request.Host = "proxy.example.com"
	return c, w
}

// TestProxyDownloadIntegration 全链路下载：URL 规范化 → 黑白名单 →
// 下载流水线 → mock 上游 → 流式输出。
// mock 返回 200 + Content-Length，浏览器应看到进度条（CL=总大小）。
func TestProxyDownloadIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	mock := withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "bytes=0-0" {
			// Range 预检：206 + 总大小
			w.Header().Set("Content-Range", "bytes 0-0/5")
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte("h"))
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "5")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("hello"))
	})

	c, w := newProxyContext("GET", "/https://github.com/owner/repo/raw/main/a.txt")
	GitHubProxyHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if w.Body.String() != "hello" {
		t.Errorf("body = %q, want %q", w.Body.String(), "hello")
	}
	if got := w.Header().Get("Content-Length"); got != "5" {
		t.Errorf("Content-Length = %q, want 5", got)
	}
	_ = mock
}

// TestProxyDownloadScriptReplacement .sh 脚本中的 GitHub URL 应被替换为代理地址。
func TestProxyDownloadScriptReplacement(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	script := "curl -L https://github.com/owner/repo/raw/main/install.sh -o /tmp/i.sh\n"
	mock := withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(len(script)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(script))
	})

	c, w := newProxyContext("GET", "/https://github.com/owner/repo/raw/main/install.sh")
	GitHubProxyHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	body := w.Body.String()
	if !strings.Contains(body, "https://proxy.example.com/https://github.com/owner/repo/raw/main/install.sh") {
		t.Errorf("脚本 URL 未被替换为代理地址:\n%s", body)
	}
	if strings.Contains(body, "curl -L https://github.com/owner/repo") {
		t.Error("脚本中仍存在未替换的 GitHub URL")
	}
	_ = mock
}

// TestProxyDownloadContentTypeBlocked 网页类型（text/html）应被 403 拦截。
func TestProxyDownloadContentTypeBlocked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html>not allowed</html>"))
	})

	c, w := newProxyContext("GET", "/https://github.com/owner/repo/raw/main/page.html")
	GitHubProxyHandler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("text/html 应返回 403, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "Content type not allowed") {
		t.Errorf("响应应包含拒绝原因: %s", w.Body.String())
	}
}

// TestProxyDownloadFileSizeLimit 超限文件应返回 413。
func TestProxyDownloadFileSizeLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "3000000000") // 3GB > 默认 2GB 上限
		w.WriteHeader(http.StatusOK)
	})

	c, w := newProxyContext("GET", "/https://github.com/owner/repo/releases/download/v1/big.bin")
	GitHubProxyHandler(c)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("超大文件应返回 413, got %d", w.Code)
	}
}

// TestProxyDownloadRangeResume 断点续传：客户端带 Range 时透传，
// 206 响应保留剩余部分 Content-Length，不做预检。
func TestProxyDownloadRangeResume(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	preflightSeen := false
	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "bytes=0-0" {
			preflightSeen = true
			w.Header().Set("Content-Range", "bytes 0-0/5")
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			return
		}
		if r.Header.Get("Range") != "bytes=2-4" {
			t.Errorf("Range = %q, want bytes=2-4", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 2-4/5")
		w.Header().Set("Content-Length", "3")
		w.WriteHeader(http.StatusPartialContent)
		w.Write([]byte("llo"))
	})

	c, w := newProxyContext("GET", "/https://github.com/owner/repo/raw/main/a.txt")
	c.Request.Header.Set("Range", "bytes=2-4")
	GitHubProxyHandler(c)

	if w.Code != http.StatusPartialContent {
		t.Fatalf("状态码 = %d, want 206 (body=%s)", w.Code, w.Body.String())
	}
	if w.Body.String() != "llo" {
		t.Errorf("body = %q, want %q", w.Body.String(), "llo")
	}
	if got := w.Header().Get("Content-Length"); got != "3" {
		t.Errorf("Content-Length = %q, want 3（剩余部分大小）", got)
	}
	if preflightSeen {
		t.Error("断点续传不应触发 Range 预检")
	}
}

// TestProxyAPIRequestIntegration API 代理：JSON 转发 + no-store 缓存策略。
func TestProxyAPIRequestIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("ETag", `"abc"`)
		w.WriteHeader(http.StatusOK)
		io.WriteString(w, `{"default_branch":"main"}`)
	})

	c, w := newProxyContext("GET", "/https://api.github.com/repos/owner/repo")
	GitHubProxyHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"default_branch":"main"`) {
		t.Errorf("API 响应未转发: %s", w.Body.String())
	}
	if got := w.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if got := w.Header().Get("ETag"); got != `"abc"` {
		t.Errorf("ETag = %q, want 透传上游值", got)
	}
}

// TestProxyDownloadInvalidURL 非法 GitHub URL 应被 URL 规范化拒绝（403）。
func TestProxyDownloadInvalidURL(t *testing.T) {
	gin.SetMode(gin.TestMode)
	SetApplication(service.NewApplication(config.DefaultConfig()))

	c, w := newProxyContext("GET", "/not/a/github/url")
	GitHubProxyHandler(c)

	if w.Code != http.StatusForbidden {
		t.Errorf("非法 URL 应返回 403, got %d", w.Code)
	}
}
