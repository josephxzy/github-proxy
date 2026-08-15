package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github-proxy/internal/config"
	"github-proxy/internal/service"

	"github.com/gin-gonic/gin"
)

// TestGitCloneStreamNoPreflightAndChunked 验证 git 智能 HTTP 数据流的直连透传路径：
//  1. GET /info/refs（服务发现）不触发 Range 预检（修复前会多打一次上游请求）
//  2. POST /git-upload-pack 无 Content-Length（真实 GitHub 行为：流式 chunked）
//     时数据完整透传——覆盖直连 io.Copy 下 knownSize=0 的 LimitReader 分支
//  3. 普通下载仍触发 Range 预检（回归对照，证明排除逻辑只针对 git 请求）
//
// 背景：git-upload-pack 是 POST 流式响应、不支持 Range 重连，因此代理对
// git 请求不再走水位线缓冲（缓冲模式会让上游连接空闲被掐断 → early EOF），
// 而是直连 pipe 透传，预检/重连包装一并跳过。
func TestGitCloneStreamNoPreflightAndChunked(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("TOKEN_WHITELIST", "")
	t.Setenv("DOWNLOAD_RATE", "0")
	t.Setenv("GLOBAL_RATE", "0")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	SetApplication(service.NewApplication(config.GetConfig()))

	var mu sync.Mutex
	preflightCount := 0
	packPayload := strings.Repeat("PACKDATA", 2048) // 16KB
	refsPayload := "001e# service=git-upload-pack\n0000"

	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "bytes=0-0" {
			mu.Lock()
			preflightCount++
			mu.Unlock()
			w.Header().Set("Content-Range", "bytes 0-0/5")
			w.Header().Set("Content-Length", "1")
			w.WriteHeader(http.StatusPartialContent)
			w.Write([]byte("h"))
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, "/git-upload-pack"):
			// 真实 GitHub 行为：无 Content-Length，流式 chunked
			w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(packPayload))
		case strings.HasSuffix(r.URL.Path, "/info/refs"):
			w.Header().Set("Content-Type", "application/x-git-upload-pack-advertisement")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(refsPayload))
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", "5")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("hello"))
		}
	})

	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.Use(NewIPRateLimiter(0).Middleware())
	r.NoRoute(GitHubProxyHandler)

	t.Run("git-upload-pack chunked 直连透传数据完整", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("want 001\n"))
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		if got := w.Body.String(); got != packPayload {
			t.Errorf("git-upload-pack 响应体不完整: len=%d, want %d", len(got), len(packPayload))
		}
		if got := w.Header().Get("Content-Type"); got != "application/x-git-upload-pack-result" {
			t.Errorf("Content-Type = %q, want 透传上游值", got)
		}
	})

	t.Run("info/refs 不触发 Range 预检且数据完整", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/josephxzy/med-ice.git/info/refs?service=git-upload-pack", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		if got := w.Body.String(); got != refsPayload {
			t.Errorf("info/refs 响应体不完整: %q, want %q", got, refsPayload)
		}
	})

	t.Run("普通下载仍触发 Range 预检（回归对照）", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/https://github.com/owner/repo/raw/main/a.txt", nil)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		if got := w.Body.String(); got != "hello" {
			t.Errorf("普通下载响应体 = %q, want %q", got, "hello")
		}
		if got := w.Header().Get("Content-Length"); got != "5" {
			t.Errorf("普通下载 Content-Length = %q, want 5（预检总大小）", got)
		}
	})

	t.Run("git 请求未触发任何 Range 预检", func(t *testing.T) {
		mu.Lock()
		defer mu.Unlock()
		if preflightCount != 1 {
			t.Errorf("Range 预检次数 = %d, want 1（仅普通下载触发；git 请求应跳过）", preflightCount)
		}
	})
}

// TestGitCloneStreamLimitedThrottledComplete 未认证 git 流在限速下数据仍完整：
// 直连 pipe 保留了限速（getDownloadLimiter 不变），2KB @ 1000B/s 应耗时约 2 秒
// 且字节完整——验证"修复不破坏限速"。
func TestGitCloneStreamLimitedThrottledComplete(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("TOKEN_WHITELIST", "")
	t.Setenv("DOWNLOAD_RATE", "1000")
	t.Setenv("GLOBAL_RATE", "0")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	SetApplication(service.NewApplication(config.GetConfig()))

	payload := strings.Repeat("x", 2*1024)
	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	})

	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.NoRoute(GitHubProxyHandler)

	start := time.Now()
	w := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("want"))
	r.ServeHTTP(w, req)
	elapsed := time.Since(start)

	if w.Code != http.StatusOK {
		t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
	}
	if w.Body.String() != payload {
		t.Errorf("限速下 git 流数据不完整: len=%d, want %d", len(w.Body.String()), len(payload))
	}
	if elapsed < 1500*time.Millisecond {
		t.Errorf("未认证 git 流应被限速（2KB@1000B/s≈2s），实际仅 %v", elapsed)
	}
}
