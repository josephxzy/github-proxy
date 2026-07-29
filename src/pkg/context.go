package utils

import "github.com/gin-gonic/gin"

// GetAuthFromContext 从gin.Context中获取不限速豁免状态（token 在白名单中）。
func GetAuthFromContext(c *gin.Context) bool {
	if v, ok := c.Get("authenticated"); ok {
		return v.(bool)
	}
	return false
}
