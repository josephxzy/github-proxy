# 初始项目搭建

## 背景

2025 年，基于 Gin + Vue 3 构建 GitHub 资源加速反向代理的初始版本。

## 完成内容

- Go 后端：Gin 框架，完整代理管道（下载、API、Git 加速）
- Vue 3 前端：仓库搜索、Release 浏览、README 预览、Token 设置
- 双层限速：单流漏桶 + global 共享令牌桶
- 水位线反压：真环形缓冲区，80%/20% 暂停/恢复
- Token 三源提取：`X-GitHub-Token` 头 / `?token=` 参数 / `Authorization: Basic`
- 断点续传：Range 预检 + 透传
- 脚本 URL 替换：`.sh` / `.ps1` 内 GitHub 链接自动代理
- IP 三层限流：固定窗口计数器，IPv6 /64 归一化
- 构建脚本：`build.sh`（Linux/macOS）和 `build.ps1`（Windows）
- 文档站：Astro + Starlight，12 个文档页面

## 架构决策

- 使用 Gin 作为 HTTP 框架
- 前端嵌入 Go 二进制，通过 `embed` 包打包
- 配置通过 `config.toml` 和环境变量双通道
- 限速和水位线在 Go 层实现，不依赖操作系统

## 相关模块

- 全部 `src/` 目录