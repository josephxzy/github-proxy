package service

import "strings"

// TokenWhiteListService GitHub Token 白名单服务。
// 用于控制哪些用户可以不限速使用代理。
//
// 规则：
//   - 未传入 token → 限速
//   - 传入 token 但不在白名单 → 限速
//   - 传入 token 且在白名单 → 不限速
type TokenWhiteListService struct {
	tokens map[string]bool
}

// NewTokenWhiteListService 创建 Token 白名单服务实例。
//
// 参数:
//   - tokens: 白名单 token 列表（支持逗号分隔的多 token）
func NewTokenWhiteListService(tokens []string) *TokenWhiteListService {
	m := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		t = strings.TrimSpace(t)
		if t != "" {
			m[t] = true
		}
	}
	return &TokenWhiteListService{tokens: m}
}

// IsWhitelisted 检查 token 是否在白名单中。
// 白名单为空时始终返回 false（无人豁免限速）。
func (s *TokenWhiteListService) IsWhitelisted(token string) bool {
	if len(s.tokens) == 0 {
		return false
	}
	return s.tokens[token]
}
