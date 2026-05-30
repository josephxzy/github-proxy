// Package main 是 github-proxy 项目的入口包
// github-proxy 是一个 GitHub 代理服务，用于加速 GitHub 资源的访问和下载
// 支持功能包括：
//   - GitHub API 代理（搜索、发布版本、仓库信息等）
//   - GitHub 文件/资源下载代理
//   - Docker 镜像拉取代理
//   - 访问控制和速率限制
//   - 多节点注册和负载均衡
//   - Web 前端界面
package main

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github-proxy/config"
	ghproxyhandlers "github-proxy/handlers"
	"github-proxy/internal/server"
	"github-proxy/internal/service"
	"github-proxy/pkg/network"
)

// staticFiles 嵌入的静态文件系统，包含前端构建产物
// 使用 Go 的 embed 包将前端静态资源编译进二进制文件
//
//go:embed public/*
var staticFiles embed.FS

// serviceStartTime 记录服务的启动时间，用于计算服务运行时长
var serviceStartTime = time.Now()

// main 应用程序的主入口函数
// 执行流程：
//  1. 加载配置文件（config.toml 或环境变量）
//  2. 打印服务启动横幅信息
//  3. 初始化 HTTP 客户端连接池
//  4. 创建并启动应用服务（包括节点注册等后台服务）
//  5. 配置并构建 HTTP 路由器
//  6. 启动 HTTP 服务器监听请求
func main() {
	// 步骤1：加载配置
	if err := config.LoadConfig(); err != nil {
		fmt.Printf("配置加载失败: %v\n", err)
		return
	}

	// 获取配置实例
	cfg := config.GetConfig()

	// 步骤2：打印启动横幅，显示版本信息和关键配置
	printBanner(cfg)

	// 步骤3：初始化全局 HTTP 客户端（包含连接池和超时设置）
	network.InitHTTPClients()

	// 步骤4：创建应用实例，初始化所有核心服务
	app := service.NewApplication(cfg)

	// 将应用实例注入到处理器中，使处理器可以访问服务层
	ghproxyhandlers.SetApplication(app)

	// 启动应用的后台服务（节点注册、健康检查等）
	ctx := context.Background()
	if err := app.Start(ctx); err != nil {
		fmt.Printf("服务启动失败: %v\n", err)
		return
	}
	// 确保程序退出时优雅停止所有服务
	defer app.Stop()

	// 步骤5：构建路由器配置
	routerCfg := &server.RouterConfig{
		AppConfig:        cfg,                                     // 应用配置
		Version:          Version,                                 // 版本号
		BuildTime:        BuildTime,                               // 构建时间
		ServiceStartTime: serviceStartTime,                        // 服务启动时间
		StaticFS:         &server.EmbedFSWrapper{FS: staticFiles}, // 静态文件系统
		NodeRegistry:     app.NodeRegistry,                        // 节点注册中心
	}

	// 构建路由器，注册所有路由规则
	router := server.BuildRouter(routerCfg)

	// 初始化网络流量监控
	server.InitNetworkMonitor()

	// 步骤6：创建并启动 HTTP 服务器
	srv := server.NewServer(cfg, router)

	if err := srv.Start(); err != nil {
		fmt.Printf("启动服务失败: %v\n", err)
	}
}
