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

// TestGitCloneWhitelistThrottleExempt 端到端验证 git clone 场景的认证与限速语义：
// 用户 URL 内嵌 token（https://ghp_xxx@host/...）时，git 发送
// Authorization: Basic base64("ghp_xxx:")（token 作为用户名、空密码）。
// git 智能 HTTP 流一律不限速（v1.3.8 起，即使配置了
// DownloadBytesPerSec/GlobalBytesPerSec 且非白名单），白名单豁免判断
// 仅作用于 release / archive / raw 等文件下载。
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

	t.Run("对照组：非白名单 token 的 git 流同样不限速（git 一律不限速）", func(t *testing.T) {
		start := time.Now()
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("pack"))
		// 非白名单凭据：git 流不限速（限速仅作用于 release/archive/raw 下载）
		req.Header.Set("Authorization", basicHeader("octocat", "ghp_otherToken999"))
		r.ServeHTTP(w, req)

		elapsed := time.Since(start)
		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		if elapsed >= 1500*time.Millisecond {
			t.Errorf("git 流不应被限速（git 一律不限速），2KB 耗时 %v", elapsed)
		}
	})
}

// TestGitCloneAuthChallenge 验证 git 智能 HTTP 端点的 401 认证挑战。
// 根因：git 客户端首次请求从不携带凭据（即使 URL 内嵌 token），只有收到 401
// 后才会用 URL 凭据重试；若代理对公共仓库直接透传 200，git 永不发送凭据，
// 白名单 token 无法被提取 → 白名单豁免失效 → 用户被限速。
// 因此配置了 token 白名单时，代理必须对无凭据的 git 请求返回 401 挑战。
func TestGitCloneAuthChallenge(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("TOKEN_WHITELIST", "ghp_whitelistedToken123")
	t.Setenv("DOWNLOAD_RATE", "0") // 只测认证挑战，不限速
	t.Setenv("GLOBAL_RATE", "0")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	SetApplication(service.NewApplication(config.GetConfig()))

	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("refs")) // 模拟 git refs 通告
	})

	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.NoRoute(GitHubProxyHandler)

	t.Run("无凭据 GET /info/refs → 401 + Basic 挑战", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/josephxzy/med-ice.git/info/refs?service=git-upload-pack", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("状态码 = %d, want 401 (body=%s)", w.Code, w.Body.String())
		}
		if got := w.Header().Get("WWW-Authenticate"); !strings.Contains(got, "Basic") {
			t.Errorf("WWW-Authenticate = %q, want Basic 挑战", got)
		}
	})

	t.Run("带凭据重试 → 正常转发 200", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/josephxzy/med-ice.git/info/refs?service=git-upload-pack", nil)
		// git 重试格式：URL 内嵌 token 作为用户名、空密码
		req.Header.Set("Authorization", basicHeader("ghp_whitelistedToken123", ""))
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("状态码 = %d, want 200 (body=%s)", w.Code, w.Body.String())
		}
		if w.Body.String() != "refs" {
			t.Errorf("body = %q, want %q", w.Body.String(), "refs")
		}
	})

	t.Run("无凭据 POST /git-upload-pack 同样 401", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/josephxzy/med-ice.git/git-upload-pack", strings.NewReader("want"))
		r.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("状态码 = %d, want 401 (body=%s)", w.Code, w.Body.String())
		}
	})

	t.Run("web 下载路径不受挑战影响", func(t *testing.T) {
		w := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/josephxzy/med-ice/releases/download/v1/file.zip", nil)
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("web 下载路径不应 401, 状态码 = %d (body=%s)", w.Code, w.Body.String())
		}
	})
}

// TestGitCloneNoChallengeWithoutWhitelist 未配置 token 白名单时，
// 匿名 git clone 不挑战（保持原有可用性，限速逻辑按原样生效）。
func TestGitCloneNoChallengeWithoutWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	t.Setenv("TOKEN_WHITELIST", "")
	t.Setenv("DOWNLOAD_RATE", "0")
	t.Setenv("GLOBAL_RATE", "0")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	SetApplication(service.NewApplication(config.GetConfig()))

	withMockGitHub(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Length", "4")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("refs"))
	})

	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.NoRoute(GitHubProxyHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/josephxzy/med-ice.git/info/refs?service=git-upload-pack", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("未配置白名单时匿名 clone 应放行, 状态码 = %d (body=%s)", w.Code, w.Body.String())
	}
}
