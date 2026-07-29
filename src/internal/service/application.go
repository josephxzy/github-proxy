package service

import (
	"context"

	"github-proxy/config"
	ghproxygithub "github-proxy/internal/service/github"
)

// Application 应用程序核心类
// **统一管理所有服务实例**，作为依赖注入容器：
//
// 基础设施层：
//	  - TokenWhiteListService: Token 白名单（不限速）
//	  - AccessControlService: 仓库权限控制（黑白名单）
//
//	GitHub 领域层 (github package)：
//	  - ProxyService: 代理编排层（路由分发）
//	    ├─ DownloadService: 文件下载服务
//	    └─ APIService: API 代理服务
//
// 所有 Handler 和 Server 通过 Application 获取所需的服务实例。
type Application struct {
	Config          *config.AppConfig
	TokenWhiteList  *TokenWhiteListService
	AccessCtrl      *AccessControlService
	URLNormalizer   *ghproxygithub.URLNormalizer
	Proxy           *ProxyService
}

// NewApplication 创建应用程序实例并初始化所有服务。
func NewApplication(cfg *config.AppConfig) *Application {
	app := &Application{
		Config: cfg,
		TokenWhiteList: NewTokenWhiteListService(cfg.TokenWhiteList.Tokens),
		AccessCtrl: NewAccessControlService(
			cfg.Access.WhiteList,
			cfg.Access.BlackList,
		),
		URLNormalizer: ghproxygithub.NewURLNormalizer(),
		Proxy:         NewProxyService(cfg),
	}

	return app
}

// Start 启动所有需要后台运行的服务。
func (app *Application) Start(ctx context.Context) error {
	cfg := config.GetConfig()
	ghproxygithub.InitGlobalAPILimiters(
		cfg.RateLimit.APISearchHourly,
		cfg.RateLimit.APIReleaseHourly,
		cfg.RateLimit.APIRepoHourly,
		cfg.RateLimit.APIOtherHourly,
	)

	return nil
}

// Stop 优雅停止所有服务。
func (app *Application) Stop() {
}

// GetDownloadService 获取文件下载服务。
func (app *Application) GetDownloadService() *ghproxygithub.DownloadService {
	return app.Proxy.GetDownloadService()
}

// GetAPIService 获取 API 代理服务。
func (app *Application) GetAPIService() *ghproxygithub.APIService {
	return app.Proxy.GetAPIService()
}

// GetURLNormalizer 获取 URL 规范化器。
func (app *Application) GetURLNormalizer() *ghproxygithub.URLNormalizer {
	return app.URLNormalizer
}
