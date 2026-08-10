package service

import (
	"context"

	"github-proxy/internal/config"
	ghproxygithub "github-proxy/internal/github"
)

// Application 应用程序核心类，作为轻量级依赖容器：
//
//	TokenWhiteListService: Token 白名单（命中则豁免限速）
//	AccessControlService: 仓库权限控制（黑白名单）
//	URLNormalizer:        下载 URL 规范化（补全协议、blob→raw 改写）
//
// 说明：本项目不存在"编排层"——请求处理流水线（下载 / API 代理、预检、
// 断点续传、脚本替换、限速、水位线反压）全部实现在 handlers 包内，
// 直接使用 github 包提供的工具函数。Application 只承载 handler 需要的
// 少量有状态服务，避免全局变量散落。
type Application struct {
	Config         *config.AppConfig
	TokenWhiteList *TokenWhiteListService
	AccessCtrl     *AccessControlService
	URLNormalizer  *ghproxygithub.URLNormalizer
}

// NewApplication 创建应用程序实例并初始化所有服务。
func NewApplication(cfg *config.AppConfig) *Application {
	return &Application{
		Config:         cfg,
		TokenWhiteList: NewTokenWhiteListService(cfg.TokenWhiteList.Tokens),
		AccessCtrl: NewAccessControlService(
			cfg.Access.WhiteList,
			cfg.Access.BlackList,
		),
		URLNormalizer: ghproxygithub.NewURLNormalizer(),
	}
}

// Start 启动所有需要后台运行的服务。
func (app *Application) Start(ctx context.Context) error {
	ghproxygithub.InitGlobalAPILimiters(
		app.Config.RateLimit.APISearchHourly,
		app.Config.RateLimit.APIReleaseHourly,
		app.Config.RateLimit.APIRepoHourly,
		app.Config.RateLimit.APIOtherHourly,
	)

	return nil
}

// Stop 优雅停止所有服务。
// 当前所有服务均为无状态或自管理生命周期（如限速器懒初始化），
// 暂无需要显式关闭的后台资源，保留该入口供未来扩展。
func (app *Application) Stop() {
}

// GetURLNormalizer 获取 URL 规范化器。
func (app *Application) GetURLNormalizer() *ghproxygithub.URLNormalizer {
	return app.URLNormalizer
}
