# 构建与测试

## 构建

```bash
# Linux / macOS
./build.sh v1.2.0

# Windows
.\build.ps1 -Version v1.2.0
```

构建脚本编译 Go 二进制并打包前端资源，输出到 `build/` 目录。

## 测试

```bash
# 运行所有测试（在仓库根目录下）
go test ./...

# 运行特定包的测试
go test ./internal/handlers/

# 带详细输出
go test -v ./internal/handlers/

# 静态检查
go vet ./...
```

## Race 检测

CI（`.github/workflows/ci.yml`）已启用 `go test -race`（Ubuntu 自带 gcc，无需额外配置）。

本机（Windows）运行 race 需要 C 编译器：Go 的 race detector 在 Windows/amd64 上依赖 cgo，
需安装 MSYS2 UCRT64 环境（`C:\msys64\ucrt64`，内含 gcc）。注意 **PATH 必须包含
`ucrt64/bin`**——gcc 的子进程（`cc1`/`as`/`ld`）与依赖 DLL 从这里加载，否则 cgo 编译
`runtime/cgo` 报 `exit status 2`。可在 VS Code 中用 MSYS2 UCRT64 终端
（`C:\msys64\msys2_shell.cmd -defterm -here -no-start -ucrt64`），或手动导出：

```bash
# Windows + MSYS2 UCRT64（git bash）
export PATH="/c/msys64/ucrt64/bin:$PATH"
CGO_ENABLED=1 CC="C:/msys64/ucrt64/bin/gcc.exe" go test -race -count=1 ./...
```

## 测试覆盖

### 既有测试

- `internal/waterline/waterline_test.go` —— 水位线环形缓冲区（写入、读取、暂停/恢复、满时阻塞、关闭后 EOF）
- `internal/network/reconnecting_reader_test.go` —— 断线自动重连（偏移校验、重连上限）
- `internal/github/download/helpers_test.go` —— URL 匹配 / 短路径规范化
- `internal/github/download/auth_whitelist_test.go` —— git Basic 认证 Token 提取
- `internal/handlers/github_test.go` —— 仓库黑白名单在 handler 真实输入下的行为
- `internal/handlers/git_auth_whitelist_test.go` —— git clone 白名单链路（authenticated 标记）
- `internal/service/access_control_service_test.go` —— 仓库访问控制

### 代码质量重构新增测试

- `internal/config/config_test.go` —— 默认值、深拷贝、环境变量覆盖（含非法值忽略）
- `internal/ratelimit/ratelimit_test.go` —— 单用户/全局限速时序、Writer 透传与限速、Flush
- `internal/network/response_test.go` —— 内容类型黑名单、安全头清理、文件大小限制、真实 Host、Content-Range 解析、重定向处理
- `internal/github/download/script_processor_test.go` —— 脚本 URL 替换、gzip 解压、超限拒绝
- `internal/github/api/api_queue_test.go` —— API 限速器分类、白名单豁免、默认值回退、窗口重置
- `internal/handlers/ip_limiter_test.go` —— IP 归一化（IPv4/IPv6 /64）、固定窗口、清理、中间件 429
- `internal/handlers/upstream_test.go` —— 公共上游请求构造（头复制、Host 移除、Token 应用）
- `internal/server/server_test.go` —— uptime 格式化、MIME 检测、静态文件服务、/ready 路由

## 测试位置约定

测试文件遵循 Go 标准工具链约定：**与被测包位于同一目录**（如 `internal/config/config_test.go` 对应 `internal/config` 包）。
原因：

- `go test ./...` 只会编译并运行各包目录内的 `_test.go`，放在其他目录（如 `test/`）的测试文件不会被任何包编译，测试将全部失效。
- 现有测试为白盒测试，直接访问包内私有符号（如 `setConfig`、`wait`、`parseContentRange`、`checkRepoAccess`、`normalizeIP`、`formatUptime`、`transformURL` 等），跨目录黑盒测试无法覆盖这些路径。

因此**不设集中测试目录**。若未来出现可复用的共享测试辅助（如内存静态文件系统、请求头构造），可提取到 `internal/testutil` 独立包，由各包测试导入。

## 代码规范

- 使用 Go 标准库 `testing` 包
- 文件名以 `_test.go` 结尾
- 函数名以 `Test` 开头
- 表驱动测试优先于重复代码
- 涉及时间的限速测试使用宽松断言（±30%）避免 CI 抖动

## 源码结构

| 目录 | 职责 |
|:---|:---|
| `cmd/github-proxy/main.go` | 入口，启动 Gin 服务 |
| `internal/handlers/` | 代理请求流水线（下载、API、IP 限流、流式输出），**实际的业务逻辑所在** |
| `internal/github/` | GitHub 领域工具函数（URL 规范化、Token 提取、脚本处理、API 限速、默认分支） |
| `internal/service/` | 有状态服务（Token 白名单、仓库访问控制） |
| `internal/config/` | 配置加载与解析 |
| `internal/ratelimit/` | 双层限速（单用户漏桶 + 全局令牌桶） |
| `internal/waterline/` | 水位线反压环形缓冲区 |
| `internal/server/` | HTTP 服务器与路由 |
| `internal/network/` | 网络层（HTTP 客户端、重连读取器、响应工具） |
| `web/` | Vue 3 前端（嵌入构建） |
| `cmd/github-proxy/public/` | 静态资源 |

> 架构说明：本项目**不存在独立的服务编排层**——handler 直接使用
> `internal/github` 的包级工具函数完成请求流水线。
> 详见 [代码分层约定](../wiki/architecture/code-layering.md)。
