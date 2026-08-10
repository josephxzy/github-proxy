# 常见故障模式

## 背景

github-proxy 作为代理，同时与客户端、GitHub API、操作系统内核三方交互。故障往往跨越多个层面，需要系统性的排查路径。

## 常见模式

### 1. 下载末尾报网络错误

**症状**：Chrome/Edge 下载到 99% 时提示"网络错误"。

**根因**：线性缓冲区满时短写，数据丢失。

**排查**：确认 `internal/waterline/waterline.go` 使用真环形缓冲区实现。搜索 `copy(b.data[b.writePos:], data)` 残留代码。

**不应使用的短期手段**：重试下载——数据丢失是确定性的，重试无效。

### 2. Nginx 反向代理后无进度条

**症状**：浏览器下载不显示进度条，断点续传失败。

**根因**：Nginx 开启了 gzip 或 brotli 压缩，或未关闭缓冲。

**修复**：在 Nginx location 块中添加：
```nginx
gzip off;
brotli off;
zstd off;
proxy_buffering off;
```

### 3. 内存持续增长

**症状**：长期运行后内存占用持续上升。

**根因**：IP 限流的 cleanup goroutine 未启动，IP 记录只增不减。

**排查**：检查 `internal/handlers/ip_limiter.go` 中 `StartCleanup` 是否在服务启动时调用。

### 4. 下载速度忽快忽慢

**症状**：下载速度在 0 和配置值之间剧烈波动。

**根因**：`BUFFER_SIZE` 过小，导致高频暂停/恢复。

**排查**：调大 `BUFFER_SIZE`（8MB → 16MB → 32MB），观察振荡周期是否变长。

### 5. 私有仓库 API 请求返回 403

**症状**：前端设了 token 但搜索私有仓库仍然 403。

**根因**：token 已过期或权限不足。检查 `X-Token-Status: invalid` 响应头。

**修复**：重新生成 GitHub PAT，确保包含 `repo` 权限。

## 相关模块

- `internal/waterline/waterline.go`
- `internal/handlers/ip_limiter.go`
- `internal/handlers/github.go`
- `internal/handlers/proxy_download.go`