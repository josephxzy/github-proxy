# 代码分层约定

## 背景

历史上 `internal/service/` 下存在一个 `ProxyService` 编排层（含 `DownloadService` / `APIService` 实例），
意图是 handler → 编排层 → 领域服务 的分层。但实际请求流水线从未经过它：
`handlers.GitHubProxyHandler` 直接调用 `internal/service/github` 的包级工具函数。
该编排层及其 3 个文件属于死代码，已在 2025 年代码质量重构中删除
（见 [重构检查点](../checkpoints/code-quality-refactor.md)）。

## 决策

**不设独立的"服务编排层"**。请求处理流水线整体位于 `handlers` 包，
依赖 `internal/service/github` 提供的纯函数工具层：

```
HTTP 请求
   │
   ▼
handlers（流水线：URL 规范化 → 黑白名单 → 下载/API 分支 →
   │        预检 → 安全检查 → 脚本替换 → 限速 → 水位线 → 流式输出）
   │
   ▼
internal/service/github（纯函数：MatchURL / ExtractToken / ProcessSmart /
   │                     PrefetchContentLength / API 限速 / 默认分支缓存）
   │
   ▼
pkg/network（HTTP 客户端、ReconnectingReader、响应工具）
```

## 理由

- 流水线的核心状态（Gin Context、响应写入、限速器、水位线）与 HTTP 层强耦合，
  强行抽象成"服务对象"只会产生包裹层，无法真正复用。
- 工具层全部是无状态纯函数，可单测、可被未来任何调用方复用。
- 之前的分层对象只有 `ProxyService` 一个调用方（main 的依赖注入），删除后
  减少了"文档说的架构"与"代码实际的架构"之间的偏差。

## 长期规则

- 新增代理逻辑时，优先作为 `handlers` 包内的函数或 `internal/service/github`
  的纯函数实现；**不要**再引入"服务对象编排层"。
- 有状态、需要生命周期管理的服务（Token 白名单、仓库访问控制、URL 规范化器）
  由 `internal/service.Application` 统一持有，handler 通过注入访问。
- 全局单例（`config.GetConfig()`、`network.GetGlobalHTTPClient()`、全局 API 限速器）
  属于已知的隐式依赖，目前刻意保留；如未来引入依赖注入再行评估。

## 相关模块

- `internal/handlers/github.go` — 入口分流
- `internal/handlers/proxy_download.go` / `proxy_api.go` — 下载 / API 流水线
- `internal/github/pkg.go` — 工具层统一导出
- `internal/service/application.go` — 有状态服务容器

## 来源文档

- 重构检查点：`../checkpoints/code-quality-refactor.md`
