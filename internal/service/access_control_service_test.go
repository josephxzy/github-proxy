package service

import "testing"

// checkList 的匹配规则（仓库黑白名单）：
// owner/repo 精确、owner 匹配用户全部仓库、owner/* 同 owner、prefix* 前缀匹配。
func TestAccessControlCheckList(t *testing.T) {
	s := NewAccessControlService(nil, nil)
	matches := []string{"josephxzy", "github-proxy"}

	tests := []struct {
		item string
		want bool
	}{
		{"josephxzy/github-proxy", true}, // 精确匹配 owner/repo
		{"josephxzy/other-repo", false},
		{"josephxzy", true},   // 匹配该用户所有仓库
		{"josephxzy/*", true}, // 同 owner
		{"torvalds/*", false},
		{"joseph*", true},           // 前缀匹配 owner
		{"josephxzy/github*", true}, // 前缀匹配 repo
		{"josephxzy/gitlab", false},
		{"", false},                 // 空条目忽略
		{"  ", false},               // 空白条目忽略
		{"*", false},                // 裸 *（空前缀）不匹配一切
		{"*/malicious-repo", false}, // 后缀通配不支持
	}
	for _, tt := range tests {
		if got := s.checkList(matches, []string{tt.item}); got != tt.want {
			t.Errorf("checkList(%v, [%q]) = %v, want %v", matches, tt.item, got, tt.want)
		}
	}
}

// CheckRepoAccess 的整体行为：黑名单拒绝、白名单优先、fail-closed、无配置放行。
func TestCheckRepoAccess(t *testing.T) {
	tests := []struct {
		name      string
		whiteList []string
		blackList []string
		matches   []string
		want      bool
	}{
		{"blacklist denies", nil, []string{"baduser/*"}, []string{"baduser", "repo"}, false},
		{"blacklist exact denies", nil, []string{"baduser/malicious-repo"}, []string{"baduser", "malicious-repo"}, false},
		{"whitelist allows", []string{"gooduser/repo"}, nil, []string{"gooduser", "repo"}, true},
		{"whitelist priority over blacklist", []string{"gooduser/repo"}, []string{"gooduser/*"}, []string{"gooduser", "repo"}, true},
		{"whitelist fail-closed", []string{"gooduser/repo"}, nil, []string{"otheruser", "repo"}, false},
		{"no config allows", nil, nil, []string{"anyuser", "anyrepo"}, true},
		{"invalid format denied", nil, nil, []string{"onlyowner"}, false},
	}
	for _, tt := range tests {
		s := NewAccessControlService(tt.whiteList, tt.blackList)
		got := s.CheckRepoAccess(tt.matches).Allowed
		if got != tt.want {
			t.Errorf("%s: CheckRepoAccess(%v).Allowed = %v, want %v", tt.name, tt.matches, got, tt.want)
		}
	}
}
