package main

// 版本信息变量
// 这些变量会在编译时通过 ldflags 注入实际的值：
//
//	-ldflags "-X main.Version=1.0.0 -X main.GitCommit=abc123 -X main.BuildTime=2024-01-01"
var (
	// Version 应用版本号，默认为 "dev" 表示开发版本
	Version = "dev"
	// GitCommit Git 提交哈希值，用于追踪具体版本的代码
	GitCommit = "unknown"
	// BuildTime 构建时间戳，记录二进制文件的编译时间
	BuildTime = "unknown"
	// 项目名称
	ProjectName = "Github-Proxy"
	// 项目仓库地址
	ProjectURL = "https://github.com/AlistairBlake/github-proxy"
)
