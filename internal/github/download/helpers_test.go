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

// IsGitSmartHTTPRequest 的端点判定：git clone/push 的三个端点必须识别，
// web 下载路径（releases/archive/blob/raw）不能误伤。
func TestIsGitSmartHTTPRequest(t *testing.T) {
	tests := []struct {
		rawPath string
		want    bool
	}{
		// git 智能 HTTP 端点（服务发现 + 数据传输）
		{"josephxzy/med-ice.git/info/refs?service=git-upload-pack", true},
		{"josephxzy/med-ice.git/info/refs?service=git-receive-pack", true},
		{"josephxzy/med-ice.git/info/refs", true},
		{"josephxzy/med-ice.git/git-upload-pack", true},
		{"josephxzy/med-ice.git/git-receive-pack", true},
		{"/josephxzy/med-ice.git/info/refs?service=git-upload-pack", true},
		{"https://github.com/josephxzy/med-ice.git/info/refs?service=git-upload-pack", true},
		// 无 .git 后缀的短路径同样识别
		{"josephxzy/med-ice/info/refs?service=git-upload-pack", true},
		{"josephxzy/med-ice/git-upload-pack", true},
		// Git LFS batch 端点
		{"josephxzy/med-ice.git/info/lfs/objects/batch", true},
		// web 下载路径不受影响
		{"josephxzy/med-ice/releases/download/v1.0/file.zip", false},
		{"josephxzy/med-ice/archive/refs/heads/main.zip", false},
		{"josephxzy/med-ice/raw/main/script.sh", false},
		{"josephxzy/med-ice/blob/main/README.md", false},
		{"josephxzy/med-ice", false},
		// 文件名恰好等于 git 端点名，不应误伤
		{"josephxzy/med-ice/releases/download/v1.0/git-upload-pack", false},
		{"josephxzy/med-ice/blob/main/info/refs", false},
		{"josephxzy/med-ice/raw/main/git-receive-pack", false},
		{"josephxzy/med-ice/releases/download/v1.0/info/refs.zip", false},
		// 相似但不匹配的路径
		{"josephxzy/med-ice/gitweb", false},
		{"josephxzy/med-ice/refs/heads/main", false},
	}
	for _, tt := range tests {
		if got := IsGitSmartHTTPRequest(tt.rawPath); got != tt.want {
			t.Errorf("IsGitSmartHTTPRequest(%q) = %v, want %v", tt.rawPath, got, tt.want)
		}
	}
}

// MatchURL 的 owner/repo 提取行为（仓库黑白名单检查依赖该结果）：
// repos API 与各类仓库下载 URL 必须能提取出 owner/repo；
// 搜索 API 无仓库标识，应返回 nil 以便调用方放行。
func TestMatchURLExtraction(t *testing.T) {
	tests := []struct {
		url  string
		want []string
	}{
		{"https://api.github.com/search/repositories?q=proxy", nil},
		{"https://api.github.com/search/code?q=foo", nil},
		{"https://api.github.com/repos/josephxzy/github-proxy/releases", []string{"josephxzy", "github-proxy"}},
		{"https://api.github.com/repos/torvalds/linux", []string{"torvalds", "linux"}},
		{"https://github.com/josephxzy/github-proxy/releases/download/v1.0/x.zip", []string{"josephxzy", "github-proxy"}},
		{"https://github.com/josephxzy/github-proxy/archive/refs/heads/main.zip", []string{"josephxzy", "github-proxy"}},
		{"https://github.com/josephxzy/github-proxy/blob/main/README.md", []string{"josephxzy", "github-proxy"}},
		{"https://raw.githubusercontent.com/josephxzy/github-proxy/main/file.go", []string{"josephxzy", "github-proxy"}},
		{"https://gist.github.com/josephxzy/abc123", []string{"josephxzy", "abc123"}},
	}
	for _, tt := range tests {
		got := MatchURL(tt.url)
		if len(got) != len(tt.want) {
			t.Errorf("MatchURL(%q) = %v, want %v", tt.url, got, tt.want)
			continue
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Errorf("MatchURL(%q) = %v, want %v", tt.url, got, tt.want)
				break
			}
		}
	}
}
