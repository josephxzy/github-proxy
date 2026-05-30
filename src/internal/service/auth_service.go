package service

import "strings"

// AuthService 用户认证服务。
// 实现 Basic Auth 认证机制，用于保护代理服务免受未授权访问
//
// 支持的认证格式：
//   - URL 前缀格式：user:password@https://github.com/...
//   - 多用户支持：可配置多个用户名密码对
type AuthService struct {
	authUsers []string // 允许的用户列表（格式："username:password"）
}

// NewAuthService 创建认证服务实例。
//
// 参数:
//   - authUsers: 用户列表，每个元素格式为 "username:password"
func NewAuthService(authUsers []string) *AuthService {
	return &AuthService{
		authUsers: authUsers,
	}
}

// AuthResult 认证结果结构体。
// 包含认证状态和处理后的路径信息
type AuthResult struct {
	Authenticated bool   // 是否认证成功
	RawPath       string // 认证信息移除后的实际路径
	AuthPrefix    string // 从原始路径中提取的认证前缀（用于日志等）
}

// Authenticate 执行用户认证并提取实际的 GitHub URL。
//
// 处理流程：
//  1. 标准化输入路径
//  2. 如果配置了认证用户，则尝试提取认证信息
//  3. 验证用户名密码是否匹配
//  4. 返回认证结果和清理后的路径
//
// 参数:
//   - rawURI: 原始请求 URI，可能包含认证信息
//
// 返回:
//   - *AuthResult: 包含认证状态和处理的路径
func (s *AuthService) Authenticate(rawURI string) *AuthResult {
	result := &AuthResult{}

	// 步骤1：标准化路径，去除前导 "/"
	rawPath := s.normalizePath(rawURI)

	// 步骤2-3：仅在配置了认证用户时才进行验证
	if len(s.authUsers) > 0 {
		user, pwd, url, hasAuth := s.extractAuthAndURL(rawPath)
		if hasAuth {
			// 提取认证前缀（用于后续处理或日志）
			result.AuthPrefix = rawPath[:len(rawPath)-len(url)]
			rawPath = url
			// 验证用户凭据
			result.Authenticated = s.verifyAuth(user, pwd)
		}
	}

	// 设置处理后的路径
	result.RawPath = rawPath
	return result
}

// normalizePath 标准化路径。
// 去除前导的 "/" 斜杠
func (s *AuthService) normalizePath(uri string) string {
	return strings.TrimLeft(uri, "/")
}

// extractAuthAndURL 从路径中提取认证信息和 GitHub URL。
//
// 支持的格式示例：
//   - "username:password/https://github.com/user/repo"
//   - "username:password/http://github.com/user/repo"
//
// 返回:
//   - user: 用户名
//   - pwd: 密码
//   - githubURL: 提取出的 GitHub URL
//   - hasAuth: 是否成功提取到认证信息
func (s *AuthService) extractAuthAndURL(rawPath string) (user, pwd, githubURL string, hasAuth bool) {
	// 查找 https:// 或 http:// 的位置
	httpsIdx := strings.Index(rawPath, "https://")
	httpIdx := strings.Index(rawPath, "http://")

	// 确定 URL 的起始位置
	urlStart := -1
	if httpsIdx >= 0 {
		urlStart = httpsIdx
	}
	if httpIdx >= 0 && (urlStart < 0 || httpIdx < urlStart) {
		urlStart = httpIdx
	}

	// 如果没有找到 URL，说明没有认证信息
	if urlStart < 0 {
		return "", "", rawPath, false
	}

	// 分离认证前缀和 URL
	prefix := strings.Trim(rawPath[:urlStart], "/")
	githubURL = rawPath[urlStart:]

	// 如果前缀为空，说明没有认证信息
	if prefix == "" {
		return "", "", githubURL, false
	}

	// 解析用户名:密码 格式
	parts := strings.SplitN(prefix, "/", 2)
	if len(parts) != 2 {
		return "", "", rawPath, false
	}

	return parts[0], parts[1], githubURL, true
}

// verifyAuth 验证用户名密码是否在允许列表中。
//
// 参数:
//   - user: 待验证的用户名
//   - pwd: 待验证的密码
//
// 返回:
//   - bool: 是否验证通过
func (s *AuthService) verifyAuth(user, pwd string) bool {
	for _, u := range s.authUsers {
		// 解析配置中的用户项（格式："username:password"）
		parts := strings.SplitN(u, ":", 2)
		if len(parts) == 2 && parts[0] == user && parts[1] == pwd {
			return true
		}
	}
	return false
}
