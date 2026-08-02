# 下载请求处理链路

## 背景

github-proxy 的下载请求是整个系统最复杂的链路，涉及 token 提取、Range 预检、Content-Length 策略、安全检查、脚本处理、流式输出、限速和水位线反压等多个环节。每个环节都有明确的职责和边界。

## 决策

**管道式处理：提取 → 预检 → 转发 → 检查 → 分发 → 流式输出**。每个阶段独立且可测试，下游通过 `WaterlineBuffer` 和 `rateLimitedWriter` 实现反压和限速。

## 当前规则

- 标准下载：并发 Range 预检（`bytes=0-0`）获取文件大小 → 设置 Content-Length → 浏览器显示进度条。
- 断点续传：检测到客户端 `Range` 头 → 跳过预检 → 透传 Range 到 GitHub → 206 响应不覆盖 Content-Length。
- Content-Type 过滤：`text/html` 等 → 403，阻止网页被代理。
- 文件大小限制：`Content-Length > MAX_FILE_SIZE`（默认 2GB）→ 413。
- 脚本处理：`.sh` / `.ps1` 内所有 GitHub 链接自动替换为代理地址。
- 每 64KB flush 一次，配合水位线反压和双层限速。

## 示例

**推荐做法**：
- 设置 `MAX_FILE_SIZE=2147483648`（2GB）防止超大文件耗尽资源
- 确保 Nginx 前置代理关闭压缩和缓冲（`gzip off; proxy_buffering off;`）

**禁止做法**：
- 在 206 响应中覆盖 Content-Length
- 探测不到 Content-Length 时填入估算值
- 在 Nginx 中开启 gzip 或 brotli（会破坏进度条和断点续传）

## 失败模式

| 症状 | 原因 | 排查 |
|:---|:---|:---|
| 下载无进度条 | Content-Length 探测失败 | 检查是否为 Archive 类型（codeload 不支持 Range） |
| 下载中断无提示 | Nginx 开启了缓冲 | 检查 `proxy_buffering off` |
| 断点续传失败 | 206 响应中 Content-Length 被覆盖 | 检查 `isRangeRequest` 判断逻辑 |
| 脚本文件下载后链接未替换 | 文件超过 10MB 脚本处理上限 | 检查文件大小是否超限 |

## 相关模块

- `src/handlers/proxy_download.go` — 下载请求入口
- `src/handlers/response_writer.go` — 流式输出
- `src/handlers/waterline.go` — 水位线反压
- `src/handlers/rate_limit.go` — 双层限速

## 来源文档

- 下载与断点续传：`../design/download.md`
- 限速与稳定性设计：`../design/rate-limit.md`