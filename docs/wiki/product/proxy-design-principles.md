# 代理核心原则

## 背景

github-proxy 的设计需要在一系列矛盾中做出权衡：速度 vs 稳定性、简单 vs 功能完整、安全 vs 易用。这些原则指导了所有架构决策。

## 决策

### 1. 错误的 Content-Length 比没有更糟

探测不到文件大小时走 chunked 传输——无进度条，但数据完整可靠。绝不填入估算值。

### 2. 一个 token，一种格式，所有请求

不管 token 来自 `X-GitHub-Token` 头、`?token=` 参数还是 `Authorization: Basic`，最终都统一为 `Authorization: token <value>`。下游不需要关心来源。

### 3. 内核不会帮我们限速

GitHub 灌数据的速度远超用户消费速度。TCP 流控只在网络层，不在应用层。必须在 Go 层实现完整的水位线反压和双层限速。

### 4. 缓冲区满时阻塞，绝不短写

线性缓冲区在满时短写丢数据，是"下载末尾网络错误"的根源。真环形缓冲区满时阻塞，确保数据完整性。

### 5. 前端 token 和 Git token 互不干扰

前端 token 存在浏览器的 localStorage 中，Git token 在 remote URL 中。两者完全独立，不需要也不应该相互感知。

### 6. 服务器 token 仅用于 API 兜底

服务器 token（`GITHUB_TOKEN`）只在 GitHub API 请求且用户未提供 token 时使用。对文件下载和 Git 操作从不生效。这是安全边界，防止服务器 token 泄露到下载日志。

## 当前规则

- 所有设计决策必须符合上述原则。
- 新增功能时，如果与原则冲突，需要重新评估。
- 原则修改需要记录到本 Wiki 页面。

## 相关模块

- 所有 `internal/handlers/` 模块
- `internal/config/config.go`