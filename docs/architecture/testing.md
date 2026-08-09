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
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./src/handlers/

# 带详细输出
go test -v ./src/handlers/
```

当前测试覆盖：

- `internal/waterline/waterline_test.go` —— 水位线环形缓冲区的单元测试（写入、读取、暂停/恢复、满时阻塞、关闭后读取 EOF 等场景）
- `pkg/network/reconnecting_reader_test.go` —— ReconnectingReader 断线重连的单元测试（偏移校验、重连上限等）
- `internal/service/github/download/helpers_test.go` —— URL 匹配等工具函数的单元测试

## 代码规范

- 使用 Go 标准库 `testing` 包
- 文件名以 `_test.go` 结尾
- 函数名以 `Test` 开头
- 表驱动测试优先于重复代码

## 源码结构

| 目录 | 职责 |
|:---|:---|
| `src/main.go` | 入口，启动 Gin 服务 |
| `src/handlers/` | 核心代理逻辑（下载、API、限速、水位线、IP 限流） |
| `src/config/` | 配置加载与解析 |
| `src/frontend/` | Vue 3 前端（嵌入构建） |
| `src/internal/` | 内部工具 |
| `src/pkg/` | 可复用公共库（HTTP 客户端等） |
| `src/public/` | 静态资源 |