package handlers

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github-proxy/internal/config"
	"github-proxy/internal/service"

	"github.com/gin-gonic/gin"
)

// 端到端验证 git clone 场景下的白名单链路：
// git 发送 Authorization: Basic（凭据管理器中的 username:PAT），
// TokenAuthMiddleware 提取 token → 白名单命中 → authenticated=true → 下载不限速。
// 覆盖 git 智能 HTTP 的三个请求：GET /info/refs（带凭据重试）与 POST /git-upload-pack。
func TestTokenAuthMiddlewareGitCloneWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfg.TokenWhiteList.Tokens = []string{"ghp_whitelistedToken123"}
	SetApplication(service.NewApplication(cfg))

	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.GET("/owner/repo.git/info/refs", func(c *gin.Context) {
		if !getAuthFromContext(c) {
			t.Error("GET /info/refs 带白名单凭据应 authenticated=true")
		}
		c.Status(http.StatusOK)
	})
	r.POST("/owner/repo.git/git-upload-pack", func(c *gin.Context) {
		if !getAuthFromContext(c) {
			t.Error("POST /git-upload-pack 带白名单凭据应 authenticated=true")
		}
		c.Status(http.StatusOK)
	})

	// git 智能 HTTP：GET info/refs（带 Basic 凭据重试）
	req1 := httptest.NewRequest("GET", "/owner/repo.git/info/refs?service=git-upload-pack", nil)
	req1.Header.Set("Authorization", basicHeader("octocat", "ghp_whitelistedToken123"))
	r.ServeHTTP(httptest.NewRecorder(), req1)

	// git 智能 HTTP：POST git-upload-pack（实际传输 pack 数据的请求）
	req2 := httptest.NewRequest("POST", "/owner/repo.git/git-upload-pack", nil)
	req2.Header.Set("Authorization", basicHeader("octocat", "ghp_whitelistedToken123"))
	r.ServeHTTP(httptest.NewRecorder(), req2)

	// 对照组：无凭据请求应 authenticated=false（限速）
	r2 := gin.New()
	r2.Use(TokenAuthMiddleware())
	r2.GET("/owner/repo.git/info/refs", func(c *gin.Context) {
		if getAuthFromContext(c) {
			t.Error("无凭据请求应 authenticated=false")
		}
		c.Status(http.StatusOK)
	})
	r2.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/owner/repo.git/info/refs?service=git-upload-pack", nil))
}

// 对照组：密码不在白名单时应 authenticated=false（仍然限速）
func TestTokenAuthMiddlewareNonWhitelistedBasic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.DefaultConfig()
	cfg.TokenWhiteList.Tokens = []string{"ghp_whitelistedToken123"}
	SetApplication(service.NewApplication(cfg))

	r := gin.New()
	r.Use(TokenAuthMiddleware())
	r.GET("/owner/repo.git/git-upload-pack", func(c *gin.Context) {
		if getAuthFromContext(c) {
			t.Error("未加白名单的密码应 authenticated=false")
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/owner/repo.git/git-upload-pack", nil)
	req.Header.Set("Authorization", basicHeader("octocat", "ghp_otherToken999"))
	r.ServeHTTP(httptest.NewRecorder(), req)
}

func basicHeader(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}
