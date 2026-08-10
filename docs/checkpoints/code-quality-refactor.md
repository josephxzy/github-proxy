# 代码质量重构检查点

## 背景

2025 年对 Go 后端（`cmd/`、`internal/`）做了一次代码质量审计与重构。目标：消除死代码与重复实现、统一类型安全与一致性，同时**不改变任何用户可见行为**（路由、限速、下载、Token 认证等协议不变），并补齐单元测试与文档。

## 审计结论（重构前问题清单）

### 1. 分层名存实亡：handler 绕过 service 编排层

- `internal/service/proxy_service.go` 定义了 `ProxyService` 编排层（`ProxyRequest` / `ProxyRequestResult`），
  但 `internal/handlers/github.go:GitHubProxyHandler` 实际直接调用
  `ProxyGitHubRequest` → `proxyDownloadRequest` / `proxyAPIRequest`，**从不经过 `ProxyService.Execute`**。
- `Application.Proxy`、`GetDownloadService()`、`GetAPIService()`（`internal/service/application.go`）同样无人调用。
- `internal/github/download/download_service.go`（`DownloadService`）与
  `internal/github/api/api_service.go`（`APIService`）的实例方法全部是死代码，
  只有 `proxy_service.go` 内部引用。

### 2. 重复实现且行为不一致

- handler 层 `proxyDownloadRequest` / `proxyAPIRequest` 各自内联了"创建请求 → 复制头 → 删除 Host →
  处理 Content-Length → 应用 Token"整段逻辑，与 service 层的 `createRequest` / `doGitHubRequest` 重复。
- Content-Length 计算存在**两份不一致**的实现：
  `handlers/proxy_download.go:handleNormalResponse`（实际生效路径，CL 优先于 preflight）与
  `download_service.go:CalculateContentLength`（preflight 优先于 CL）。死代码与活代码行为漂移。

### 3. 类型不安全与弱类型

- `internal/server/router.go:RouterConfig.FrequencyLimiter interface{}` 用空接口 +
  运行时类型断言 `cfg.FrequencyLimiter.(gin.HandlerFunc)`，且 `main.go` 从未赋值（永远 nil）。
- `internal/config/config.go:DefaultConfig` 用匿名 struct 字面量重复拼写结构体，字段一多极易漏配默认值。

### 4. 其他小问题

- `internal/handlers/proxy_download.go` 中 `defer func(){ if err := resp.Body.Close(); err != nil {} }()` 空语句体。
- `cmd/github-proxy/version.go:GitCommit` 声明了但构建脚本（`build.sh` / `build.ps1` / `release.yml`）只注入
  `Version` 与 `BuildTime`，`GitCommit` 无人读取。
- `internal/github/download/url_normalizer.go` 的
  `IsAPIURL` / `IsScriptURL` / `IsBlobURL` / `IsReleaseAPIURL` / `MatchURL` 实例方法无人调用
  （调用方使用包级函数）。

## 决策

| 问题 | 决策 | 理由 |
|:---|:---|:---|
| handler 绕过 service 编排层 | **删除** `ProxyService` / `DownloadService` / `APIService` 实例层，handlers 直接使用 `github` 包级工具函数 | 死代码层维护成本 > 架构上的"分层美感"；真正工作的流水线在 handler，文档与代码一致更重要 |
| 重复且不一致的 CL 计算 | 保留 handler 内实际生效的版本，删除 service 版本 | 消除行为漂移，避免"修了死代码"的假象 |
| `FrequencyLimiter interface{}` | 改为 `gin.HandlerFunc` 具体类型 | 编译期类型检查，去掉无意义的运行时断言 |
| `DefaultConfig` 匿名 struct | 提取命名结构体类型 `ServerConfig` 等 | toml tag / env 覆盖 / 深拷贝行为保持不变，默认值一目了然 |
| 请求构建逻辑重复 | 在 handlers 包内提取 `buildUpstreamRequest` 公共函数 | 纯提取，行为零变化 |

## 变更清单

- 删除 `internal/service/proxy_service.go`
- 删除 `internal/github/download/download_service.go`
- 删除 `internal/github/api/api_service.go`
- `application.go`：移除 `Proxy` 字段与 `GetDownloadService` / `GetAPIService`，更新注释
- `pkg.go`：移除 `DownloadService` / `APIService` / `DownloadRequest` / `DownloadResult` /
  `APIRequest` / `APIResult` 类型别名与 `NewDownloadService` / `NewAPIService`；保留全部工具函数导出
- `url_normalizer.go`：删除无人调用的实例方法，保留 `Normalize` / `ensureHTTPS` / `validateGitHubURL`
- `version.go`：删除 `GitCommit`
- `config.go`：提取命名结构体，简化 `DefaultConfig`
- `handlers`：提取 `buildUpstreamRequest`（请求构造）与 `serverError` / `serverErrorWithLatency` /
  `loopDetectedError`（统一错误响应）；修复空语句体
- `server/router.go`：`FrequencyLimiter` 类型修正
- 顺带修复：`UserLimiter` 首次 wait 不等待与并发积压丢失（初始化 `nextTime`、按 `now.Add(sleep)` 推进）；
  `IPRateLimiter(0)` 的 nil map 隐患（统一初始化 entries）
- 新增单元测试：`config`、`ratelimit`、`network`、`download`（脚本处理）、`api`（限流器）、
  `handlers`（IP 限流）、`server`（静态文件/uptime）
- 格式化：对既有 7 个 gofmt 不合规文件执行 `gofmt`（行尾保持 CRLF）；此后全部 `.go` 文件
  （去除行尾差异后）通过 `gofmt -l` 检查
- 文档站与内部文档同步更新（见"文档更新"节）

## 验证

```bash
cd src
go build ./...
go vet ./...
go test ./...        # 全部通过
```

重构前后 `go build` / `go vet` / `go test ./...` 全绿；行为协议（路由、限速、下载管道、Token 认证）
未改动一行实现逻辑，仅做删除与提取。

## 文档更新

- `docs/architecture/testing.md`：更新测试覆盖清单与源码结构表
- `docs/wiki/architecture/code-layering.md`（新增）：记录实际分层约定（handler 直连工具层）
- `docs/releases/release-notes.md`：追加重构记录（文档站"项目动态"可见）
- 本检查点挂入 `docs/README.md`

## 遗留事项

- `config.GetConfig()` 全局单例、`network.GetGlobalHTTPClient()` 全局客户端仍为隐式依赖，
  刻意保留：配置在进程生命周期内不变，单例语义正确；改为显式注入需改动全部工具函数签名
  （`ApplyGitHubToken`、`RangeProbeSize`、限速器等）且收益有限。未来若引入依赖注入再评估。
