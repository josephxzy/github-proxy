package handlers

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github-proxy/internal/config"
	"github-proxy/internal/service"

	"github.com/gin-gonic/gin"
)

// TestGitCloneWhitelistThrottleExempt 端到端验证 git clone 场景的白名单豁免：
// 用户 URL 内嵌 token（https://ghp_xxx@host/...）时，git 发送
// Authorization: Basic base64("ghp_xxx:")（token 作为用户名、空密码）。
// 若白名单命中，下载必须完全不限速（即使配置了 DownloadBytesPerSec/GlobalBytesPerSec）。
func TestGitCloneWhitelistThrottleExempt(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 通过环境变量 + LoadConfig 设置全局配置（getDownloadLimiter 读取全局
	// config.GetConfig()，与生产行为一致）：较慢的限速 1000 B/s，
	// 2KB 数据若被限速需约 2 秒。
	t.Setenv("DOWNLOAD_RATE", "1000")
	t.Setenv("GLOBAL_RATE", "2000")
	t.Setenv("TOKEN_WHITELIST", "ghp_whitelistedToken123")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	SetApplication(service.NewApplication(config.GetConfig()))

	payload := strings.Repeat("x", 2*1024)
	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(payload))
	})

	// 组装完整路由：Token 中间件 → IP 限流 → GitHub 代理
	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.Use(NewIPRateLimiter(0).Middleware())
	r.NoRoute(GitHubProxyHandler)

	t.Run("白名单 token（URL 内嵌格式）豁免限速", func(t *testing.T) {
		start := time.Now()
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("pack"))
		req.Header.Set("Authorization", basicHeader("ghp_whitelistedToken123", "")) // token 用户名 + 空密码
		r.ServeHTTP(w, req)

		elapsed := time.Since(start)
		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		if w.Body.String() != payload {
			t.Errorf("响应体不完整: len=%d, want %d", len(w.Body.String()), len(payload))
		}
		if elapsed >= 1500*time.Millisecond {
			t.Errorf("白名单 token 仍被限速：2KB 耗时 %v（>1.5s），豁免未生效", elapsed)
		}
	})

	t.Run("GCM 格式（用户名:PAT）同样豁免", func(t *testing.T) {
		start := time.Now()
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("pack"))
		req.Header.Set("Authorization", basicHeader("octocat", "ghp_whitelistedToken123"))
		r.ServeHTTP(w, req)

		elapsed := time.Since(start)
		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200", w.Code)
		}
		if elapsed >= 1500*time.Millisecond {
			t.Errorf("GCM 格式白名单 token 仍被限速：耗时 %v（>1.5s）", elapsed)
		}
	})

	t.Run("对照组：无凭据请求被限速（证明限速配置生效）", func(t *testing.T) {
		start := time.Now()
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("pack"))
		r.ServeHTTP(w, req)

		elapsed := time.Since(start)
		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200", w.Code)
		}
		if elapsed < 1500*time.Millisecond {
			t.Errorf("对照组应被限速（2KB@1000B/s≈2s），实际仅 %v", elapsed)
		}
	})
}
