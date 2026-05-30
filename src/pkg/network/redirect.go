package network

import (
	"github.com/gin-gonic/gin"
	"github-proxy/pkg"
)

// HandleRedirectLocation 处理 GitHub 返回的重定向 Location 头。
// 当 GitHub CDN 返回 302 重定向时，需要判断目标 URL 是否仍在 GitHub 域名内：
//
//	目标仍是 GitHub URL (checkURL 匹配成功):
//	  - 将认证前缀注入 Location 头
//	  - 返回 needRedirect=false（不需要递归代理，让浏览器直接跟随新 Location）
//
//	目标是外部 URL (checkURL 匹配失败):
//	  - 保持 Location 原样
//	  - 返回 needRedirect=true（需要递归调用 proxyDownloadRequest 继续代理）
//
// 认证前缀注入逻辑:
//	如果用户已认证 (authenticated=true)，且原始请求携带了认证信息（如 user:pwd@），
//	则将认证前缀拼接到重定向 URL 前面。
//	确保重定向后的请求同样携带认证信息。
//
// 参数:
//   - c: Gin 上下文（用于读取 authenticated 状态和写入 Location 响应头）
//   - location: GitHub 返回的 Location 头值（重定向目标 URL）
//   - checkURL: URL 匹配函数（通常是 ghproxygithub.MatchURL）
//
// 返回值:
//   - string: 可能被修改后的 location URL
//   - bool: 是否需要继续递归代理（true=需要，false=已处理完毕）
func HandleRedirectLocation(c *gin.Context, location string, checkURL func(string) []string) (string, bool) {
	if checkURL(location) != nil {
		authenticated := false
		if v, ok := c.Get("authenticated"); ok {
			authenticated = v.(bool)
		}
		if authenticated {
			if authPrefix, ok := utils.GetAuthPrefixFromContext(c); ok {
				location = authPrefix + location
			}
		}
		c.Header("Location", "/"+location)
		return location, false
	}
	return location, true
}
