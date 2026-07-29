package network

import (
	"github.com/gin-gonic/gin"
)

// HandleRedirectLocation 处理 GitHub 返回的重定向 Location 头。
// 当 GitHub CDN 返回 302 重定向时，需要判断目标 URL 是否仍在 GitHub 域名内：
//
//	目标仍是 GitHub URL (checkURL 匹配成功):
//	  - 将 Location 改为指向代理的相对路径
//	  - 返回 needRedirect=false（浏览器直接跟随新 Location）
//
//	目标是外部 URL (checkURL 匹配失败):
//	  - 保持 Location 原样
//	  - 返回 needRedirect=true（递归调用 proxyDownloadRequest 继续代理）
//
// 参数:
//   - c: Gin 上下文（用于写入 Location 响应头，可为 nil）
//   - location: GitHub 返回的 Location 头值（重定向目标 URL）
//   - checkURL: URL 匹配函数（通常是 ghproxygithub.MatchURL）
//
// 返回值:
//   - string: 可能被修改后的 location URL
//   - bool: 是否需要继续递归代理（true=需要，false=已处理完毕）
func HandleRedirectLocation(c *gin.Context, location string, checkURL func(string) []string) (string, bool) {
	if checkURL(location) != nil {
		if c != nil {
			c.Header("Location", "/"+location)
		}
		return location, false
	}
	return location, true
}
