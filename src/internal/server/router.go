package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github-proxy/config"
	"github-proxy/handlers"
	ghproxyservice "github-proxy/internal/service/github"

	"github.com/gin-gonic/gin"
)

// RouterConfig 路由器配置选项。
type RouterConfig struct {
	FrequencyLimiter interface{}
	AppConfig        *config.AppConfig
	Version          string
	BuildTime        string
	ServiceStartTime time.Time
	StaticFS         StaticFileSystem
}

// BuildRouter 创建并配置 Gin 引擎实例。
// 注册所有路由、中间件和处理器。
func BuildRouter(cfg *RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.Use(gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Internal server error",
			"code":  "INTERNAL_ERROR",
		})
	}))

	if cfg.FrequencyLimiter != nil {
		router.Use(cfg.FrequencyLimiter.(gin.HandlerFunc))
	}

	// Token 提取 + 白名单（IP 限流器依赖 authenticated 标志）
	router.Use(handlers.TokenAuthMiddleware())

	// IP 请求频率限制（白名单 token 用户豁免）
	ipLimiter := handlers.NewIPRateLimiter(cfg.AppConfig.RateLimit.IPRequestLimit)
	router.Use(ipLimiter.Middleware())

	registerHealthRoutes(router, cfg)
	registerAPIRoutes(router, cfg)
	registerStaticRoutes(router, cfg.AppConfig, cfg.StaticFS)

	router.NoRoute(handlers.GitHubProxyHandler)

	return router
}

// registerHealthRoutes 注册健康检查相关路由。
func registerHealthRoutes(router *gin.Engine, cfg *RouterConfig) {
	router.GET("/ready", func(c *gin.Context) {
		now := time.Now()
		uptimeDuration := now.Sub(cfg.ServiceStartTime)

		uptimeStr := formatUptime(uptimeDuration)

		c.JSON(http.StatusOK, gin.H{
			"ready":           true,
			"service":         "Github-Proxy",
			"version":         cfg.Version,
			"build_time":      cfg.BuildTime,
			"start_time_unix": cfg.ServiceStartTime.Unix(),
			"uptime":          uptimeStr,
			"uptime_seconds":  int(uptimeDuration.Seconds()),
		})
	})
}

// formatUptime 将运行时长格式化为易读的中文格式
func formatUptime(d time.Duration) string {
	totalSeconds := int(d.Seconds())
	days := totalSeconds / (24 * 3600)
	hours := (totalSeconds % (24 * 3600)) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if days > 0 {
		return fmt.Sprintf("%d 天 %d 小时", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%d 小时 %d 分钟", hours, minutes)
	}
	if minutes > 0 {
		return fmt.Sprintf("%d 分钟", minutes)
	}
	return fmt.Sprintf("%d 秒", seconds)
}

// registerAPIRoutes 注册 API 路由。
func registerAPIRoutes(router *gin.Engine, cfg *RouterConfig) {
	router.GET("/api/repo/:owner/:repo/branch", func(c *gin.Context) {
		owner := c.Param("owner")
		repo := c.Param("repo")
		if owner == "" || repo == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing owner or repo"})
			return
		}

		branch := ghproxyservice.GetDefaultBranchWithCache(owner, repo)
		c.JSON(http.StatusOK, gin.H{"branch": branch})
	})
}

// registerStaticRoutes 注册前端静态资源路由。
func registerStaticRoutes(router *gin.Engine, appCfg *config.AppConfig, staticFS StaticFileSystem) {
	if appCfg.Server.EnableFrontend {
		router.GET("/", func(c *gin.Context) {
			ServeEmbedFile(c, staticFS, "public/index.html")
		})
		router.GET("/public/*filepath", func(c *gin.Context) {
			filepath := strings.TrimPrefix(c.Param("filepath"), "/")
			ServeEmbedFile(c, staticFS, "public/"+filepath)
		})
		router.GET("/assets/*filepath", func(c *gin.Context) {
			filepath := c.Param("filepath")
			ServeEmbedFile(c, staticFS, "public/assets"+filepath)
		})

		registerFaviconRoutes(router, staticFS, true)
	} else {
		router.GET("/", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
		})
		router.GET("/public/*filepath", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
		})
		router.GET("/assets/*filepath", func(c *gin.Context) {
			c.Status(http.StatusNotFound)
		})

		registerFaviconRoutes(router, staticFS, false)
	}
}

// registerFaviconRoutes 注册 favicon 相关路由。
func registerFaviconRoutes(router *gin.Engine, staticFS StaticFileSystem, enabled bool) {
	favicons := []string{"/favicon.ico", "/favicon.svg"}

	for _, path := range favicons {
		if enabled {
			router.GET(path, func(c *gin.Context) {
				SetCORSSettings(c)
				filename := "public" + c.Request.URL.Path
				ServeEmbedFile(c, staticFS, filename)
			})
			router.OPTIONS(path, func(c *gin.Context) {
				SetCORSSettings(c)
				c.Status(http.StatusNoContent)
			})
		} else {
			router.GET(path, func(c *gin.Context) {
				c.Status(http.StatusNotFound)
			})
		}
	}
}
