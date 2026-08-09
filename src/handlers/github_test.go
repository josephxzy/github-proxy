package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github-proxy/config"
	"github-proxy/internal/service"

	"github.com/gin-gonic/gin"
)

// TestCheckRepoAccessHandler 验证 checkRepoAccess 对 handler 真实输入的处理：
// API 分支传入的路径不带 scheme（"api.github.com/repos/..."），必须先补全
// scheme 才能被 MatchURL 提取 owner/repo；搜索 API 无仓库标识应放行。
func TestCheckRepoAccessHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 黑名单场景
	cfg := config.DefaultConfig()
	cfg.Access.BlackList = []string{"baduser/*"}
	SetApplication(service.NewApplication(cfg))

	tests := []struct {
		name string
		path string // 与 GitHubProxyHandler 传入 checkRepoAccess 的路径一致
		want bool   // checkRepoAccess 是否放行
	}{
		{"api repos blacklisted", "api.github.com/repos/baduser/repo/releases", false},
		{"api repos blacklisted exact", "api.github.com/repos/baduser/malicious-repo", false},
		{"api search passes", "api.github.com/search/repositories?q=proxy", true},
		{"full download url blacklisted", "https://github.com/baduser/repo/releases/download/v1.0/x.zip", false},
		{"uppercase scheme not bypassed", "HTTP://api.github.com/repos/baduser/repo/releases", false},
		{"mixed-case scheme not bypassed", "HtTpS://api.github.com/repos/baduser/repo/releases", false},
		{"gist blacklisted", "gist.github.com/baduser/abc123", false},
		{"other repo allowed", "api.github.com/repos/gooduser/repo/releases", true},
	}
	for _, tt := range tests {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		got := checkRepoAccess(c, tt.path)
		if got != tt.want {
			t.Errorf("%s: checkRepoAccess(%q) = %v, want %v (code=%d)", tt.name, tt.path, got, tt.want, w.Code)
		}
	}

	// 白名单 fail-closed：未命中白名单的请求应被拒绝
	cfg2 := config.DefaultConfig()
	cfg2.Access.WhiteList = []string{"gooduser/repo"}
	SetApplication(service.NewApplication(cfg2))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if checkRepoAccess(c, "api.github.com/repos/otheruser/repo") {
		t.Error("白名单未命中应拒绝（fail-closed）")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("白名单拒绝应返回 403, got %d", w.Code)
	}
}
