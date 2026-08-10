package main

import (
	"fmt"

	"github-proxy/internal/config"
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
	fmt.Printf(" 前端: %s\n", boolStr(cfg.Server.EnableFrontend, "已启用", "未启用"))
	fmt.Printf(" 代理: %s\n", cfg.Access.Proxy)
	fmt.Printf(" Token: %s\n", boolStr(cfg.Server.GitHubToken != "", "已配置", "未配置"))

	// 显示 Token 白名单数量
	whiteList := "未启用"
	if n := len(cfg.TokenWhiteList.Tokens); n > 0 {
		whiteList = fmt.Sprintf("%d个", n)
	}
	fmt.Printf(" 白名单: %s\n", whiteList)
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
