package utils

import "github.com/gin-gonic/gin"

// GetAuthFromContext 从gin.Context中获取认证状态。
func GetAuthFromContext(c *gin.Context) bool {
	if v, ok := c.Get("authenticated"); ok {
		return v.(bool)
	}
	return false
}

// GetAuthPrefixFromContext 从gin.Context中获取认证前缀
func GetAuthPrefixFromContext(c *gin.Context) (string, bool) {
	if v, ok := c.Get("authPrefix"); ok {
		return v.(string), true
	}
	return "", false
}
