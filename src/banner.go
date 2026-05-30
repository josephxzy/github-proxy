package main

import (
	"fmt"
	"strings"

	"github-proxy/config"
)

// printBanner 打印服务启动时的横幅信息
// 显示项目名称、版本、构建时间以及关键的运行配置
// 用于在控制台输出友好的启动信息，便于运维人员确认配置正确性
func printBanner(cfg *config.AppConfig) {
	fmt.Println()
	fmt.Println("============================================")
	fmt.Printf(" 项目: %s\n", ProjectName)
	fmt.Printf(" 仓库: %s\n", ProjectURL)
	fmt.Printf(" 版本: %s\n", Version)
	fmt.Printf(" 构建: %s\n", BuildTime)
	fmt.Println("--------------------------------------------")
	fmt.Printf(" 监听: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	if cfg.Server.EnableH2C {
		fmt.Printf(" H2C: 已启用\n")
	}
	fmt.Printf(" 前端: %s\n", boolStr(cfg.Server.EnableFrontend, "已启用", "未启用"))
	fmt.Printf(" 代理: %s\n", cfg.Access.Proxy)
	fmt.Printf(" Token: %s\n", boolStr(cfg.Server.GitHubToken != "", "已配置", "未配置"))
	fmt.Printf(" 节点: %s\n", nodeRegistryStr(cfg.NodeRegistry.URLs))

	// 显示认证用户数量
	auth := "未启用"
	if n := len(cfg.AuthUsers.Users); n > 0 {
		auth = fmt.Sprintf("%d用户", n)
	}
	fmt.Printf(" 认证: %s\n", auth)
	fmt.Println("============================================")
	fmt.Println()
}

// boolStr 根据布尔条件返回不同的字符串
// 用于格式化显示开关类型的配置项
func boolStr(cond bool, yes, no string) string {
	if cond {
		return yes
	}
	return no
}

// nodeRegistryStr 格式化显示节点注册列表
// 如果没有节点则返回"未连接"，否则用逗号分隔显示所有节点 URL
func nodeRegistryStr(urls []string) string {
	if len(urls) == 0 {
		return "未连接"
	}
	return strings.Join(urls, ", ")
}
