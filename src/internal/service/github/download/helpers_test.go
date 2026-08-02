package download
import "testing"

// 短链接 git clone 回归测试：
// git 智能 HTTP 流程先 GET /user/repo.git/info/refs?service=git-upload-pack，
// 再 POST /user/repo.git/git-upload-pack。后者路径中 "git-" 后面没有 "/"，
// 旧正则 (?:info|git-)/.* 无法匹配，导致 POST 被 403 拒绝
// （git 客户端表现为 "RPC failed; HTTP 403" + "expected flush after ref listing"）。
func TestShortGitHubPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		// git smart HTTP 端点
		{"josephxzy/AI-Novel-Writing-Assistant.git/info/refs?service=git-upload-pack", true},
		{"josephxzy/AI-Novel-Writing-Assistant.git/info/refs?service=git-receive-pack", true},
		{"josephxzy/AI-Novel-Writing-Assistant.git/git-upload-pack", true},
		{"josephxzy/AI-Novel-Writing-Assistant.git/git-receive-pack", true},
		// Git LFS 端点
		{"josephxzy/repo.git/info/lfs/objects/batch", true},
		// 既有下载路径不受影响
		{"josephxzy/repo/releases/download/v1.0/file.zip", true},
		{"josephxzy/repo/archive/refs/heads/main.zip", true},
		{"josephxzy/repo/raw/main/script.sh", true},
		{"josephxzy/repo/blob/main/README.md", true},
		{"josephxzy/repo", true},
		// 非白名单路径仍然拒绝
		{"josephxzy/repo/issues/123", false},
		{"josephxzy/repo/releasesxyz", false},
		{"josephxzy/repo/gitweb", false},
	}
	for _, tt := range tests {
		if got := IsShortGitHubPath(tt.path); got != tt.want {
			t.Errorf("IsShortGitHubPath(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

// 短路径经 Normalize 后必须补全为合法 GitHub URL 并通过 MatchURL 校验。
func TestNormalizeShortGitClonePath(t *testing.T) {
	n := NewURLNormalizer()
	for _, p := range []string{
		"josephxzy/AI-Novel-Writing-Assistant.git/info/refs?service=git-upload-pack",
		"josephxzy/AI-Novel-Writing-Assistant.git/git-upload-pack",
		"josephxzy/AI-Novel-Writing-Assistant.git/git-receive-pack",
	} {
		res := n.Normalize(p)
		if !res.Valid {
			t.Errorf("Normalize(%q) rejected: %v", p, res.Error)
			continue
		}
		want := "https://github.com/" + p
		if res.NormalizedURL != want {
			t.Errorf("Normalize(%q) = %q, want %q", p, res.NormalizedURL, want)
		}
	}
}
