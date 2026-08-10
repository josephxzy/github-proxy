package download

import (
	"encoding/base64"
	"net/http"
	"testing"
)

// 模拟 git 客户端在 Windows 凭据管理器（GCM）场景下会发送的各种
// Authorization: Basic 格式，验证 ExtractToken 能否正确提取出 token。
func TestExtractTokenGitBasicFormats(t *testing.T) {
	whitelistPAT := "ghp_whitelistedToken123"

	tests := []struct {
		name      string
		auth      string
		wantToken string
	}{
		{
			// GCM 标准存储：username = GitHub 用户名，password = PAT
			name:      "GCM username+PAT",
			auth:      basic("octocat", whitelistPAT),
			wantToken: whitelistPAT,
		},
		{
			// URL 嵌入 token（文档方式一）：token 作为用户名，无密码
			// git 发送 Basic base64(ghp_xxx:)，提取应回退到用户名
			name:      "URL embedded token as username, empty password",
			auth:      basic(whitelistPAT, ""),
			wantToken: whitelistPAT,
		},
		{
			// x-access-token 作为用户名，PAT 作为密码（GitHub 官方推荐格式）
			name:      "x-access-token username + PAT password",
			auth:      basic("x-access-token", whitelistPAT),
			wantToken: whitelistPAT,
		},
		{
			// GCM 密码是另一个未加白名单的 token（不匹配场景）
			name:      "password not in whitelist",
			auth:      basic("octocat", "ghp_otherToken999"),
			wantToken: "ghp_otherToken999",
		},
		{
			name:      "no auth header",
			auth:      "",
			wantToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "https://proxy.example.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			rawPath := "owner/repo.git/info/refs?service=git-upload-pack"
			token := ExtractToken(req, rawPath)
			if token != tt.wantToken {
				t.Errorf("ExtractToken() = %q, want %q", token, tt.wantToken)
			}
		})
	}
}

// 模拟 git 智能 HTTP 全流程的凭据提取：
// 无凭据 → 空 token；带 Basic 重试（GET info/refs 与 POST git-upload-pack）→ 提取到 PAT。
func TestExtractTokenGitCloneFlow(t *testing.T) {
	// 第一次：无凭据（git 首次 GET /info/refs，预期 401）
	req1, _ := http.NewRequest("GET", "https://proxy.example.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
	if token := ExtractToken(req1, "owner/repo.git/info/refs?service=git-upload-pack"); token != "" {
		t.Errorf("first request should have no token, got %q", token)
	}

	// 第二次：GCM 凭据重试（GET /info/refs）
	req2, _ := http.NewRequest("GET", "https://proxy.example.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
	req2.Header.Set("Authorization", basic("octocat", "ghp_whitelistedToken123"))
	if token := ExtractToken(req2, "owner/repo.git/info/refs?service=git-upload-pack"); token != "ghp_whitelistedToken123" {
		t.Errorf("retried GET token = %q, want ghp_whitelistedToken123", token)
	}

	// 第三次：传输 pack 数据的 POST /git-upload-pack（同样带 Basic）
	req3, _ := http.NewRequest("POST", "https://proxy.example.com/owner/repo.git/git-upload-pack", nil)
	req3.Header.Set("Authorization", basic("octocat", "ghp_whitelistedToken123"))
	if token := ExtractToken(req3, "owner/repo.git/git-upload-pack"); token != "ghp_whitelistedToken123" {
		t.Errorf("git-upload-pack POST token = %q, want ghp_whitelistedToken123", token)
	}
}

// 验证 git 的 ?service=git-upload-pack 查询参数不影响 Basic 提取。
func TestExtractTokenQueryDoesNotShadowBasic(t *testing.T) {
	req, _ := http.NewRequest("GET", "https://proxy.example.com/owner/repo.git/info/refs?service=git-upload-pack", nil)
	req.Header.Set("Authorization", basic("octocat", "ghp_whitelistedToken123"))
	token := ExtractToken(req, "owner/repo.git/info/refs?service=git-upload-pack")
	if token != "ghp_whitelistedToken123" {
		t.Errorf("ExtractToken() = %q, want ghp_whitelistedToken123", token)
	}
}

func basic(user, pass string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+pass))
}
