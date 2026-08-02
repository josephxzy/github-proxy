# 下载与断点续传

## 请求处理流程

```
客户端请求
  ↓
proxyDownloadRequest
  ├─ 创建转发请求 + 复制请求头 + 修正 Content-Length
  ├─ 应用 token
  ├─ [可选] 并发 Range 预检（获取文件总大小）
  ├─ 发送请求到 GitHub
  ├─ 安全检查（Content-Type 过滤 + 文件大小上限）
  └─ 分发
       ├─ .sh / .ps1 → handleScriptResponse（URL 替换）
       └─ 其他       → handleNormalResponse → 流式输出
```

核心代码：`handlers/proxy_download.go`。

## 下载模式

### 标准模式

```
1. 并发执行两个请求：
   - Range bytes=0-0 → Content-Range: bytes 0-0/总大小 → 得到总大小
   - 实际下载请求 → 等待响应
2. 两个请求并行，预检结果设置 Content-Length → 浏览器显示进度条
3. 流式输出到客户端（水位线 + 限速）
```

### 断点续传

客户端带 `Range` 头时，代理跳过预检，透传 Range 到 GitHub：

```
客户端 Range: bytes=1024-
     ↓
代理检测 Range → isRangeRequest=true → 跳过预检 → 透传到 GitHub
     ↓
GitHub 返回 206 Partial Content
  Content-Range: bytes 1024-9999/10000
  Content-Length: 8976（本次传输剩余字节数）
     ↓
代理: Status=206, Content-Length=8976（透传，不覆盖）
      流式输出 8976 字节
     ↓
客户端追加到已有 1024 字节 → 完整 10000 字节文件
```

| | 首次下载（200） | 断点续传（206） |
|:---|:---|:---|
| Range 预检 | 执行（并发） | 跳过 |
| 发给 GitHub 的 Range | 不发送 | 透传客户端的 Range |
| Content-Length | 总大小（预检或响应） | 不设置（透传剩余大小） |

**关键规则**：`206 → 完全不碰 Content-Length，GitHub 给什么就透传什么`。覆盖会导致字节数不符，下载工具判定连接中断。

## 文件大小探测

### 实测结果

以下是在代理服务器上实际请求 GitHub 各类型资源的结果：

| 资源类型 | Range: bytes=0-0 | 能否获取 | 进度条 |
|:---|:---|:---|:---|
| Release Asset | 206, Content-Range: bytes 0-0/13377735 | 是 | 有 |
| Raw 文件 | 206, Content-Range: bytes 0-0/7712 | 是 | 有 |
| Archive (codeload) | 200, 无 Content-Range | 否 | 无 |
| Archive (github.com) | 302 → codeload, 无 | 否 | 无 |
| Blob 页面 | 无 Content-Range | 否 | 无 |

> Archive 经过 302 重定向后落在 codeload.github.com。codeload 为 chunked 编码，不提供 Content-Length，不支持 Range。GitHub API 的 `asset.size` 是用户上传附件大小，与源码压缩包完全无关。

### 核心原则

**错误的 Content-Length 比没有 Content-Length 更糟。** 探测不到就走 chunked 传输——无进度条，但数据完整可靠。

```go
func PrefetchContentLength(ctx, url, headers) int64 {
    return RangeProbeSize(ctx, url, headers) // 失败返回 0 → chunked
}
```

## Content-Length 设置策略

| 场景 | Content-Length 来源 |
|:---|:---|
| Release Asset（响应带 CL） | GitHub 响应的 Content-Length |
| 响应无 CL 但预检成功 | 预检获得的总大小 |
| 断点续传（206） | 不覆盖，透传 GitHub 返回的剩余大小 |
| 全失败（chunked） | 不设置（绝不填估算值） |

设置 CL 的同时删除从 GitHub 复制过来的 `Transfer-Encoding` 头，避免 CL 与 TE 并存导致客户端无法判断响应边界。

## 安全检查

**Content-Type 过滤**：阻止网页被代理，仅对 GET 检查。`text/html` 等 → 403。

**文件大小限制**：`Content-Length > fileSize`（默认 2GB）→ 413。配置：`MAX_FILE_SIZE`。

## 脚本处理

`.sh` / `.ps1` 内所有 `github.com` / `githubusercontent.com` 链接自动替换为代理地址。自动检测 gzip，上限 10MB。

```
原：curl -L https://github.com/user/repo/releases/download/v1.0/tool.sh
后：curl -L https://hub.example.com/https://github.com/user/repo/releases/download/v1.0/tool.sh
```

## 流式输出

`handlers/response_writer.go` 负责写入客户端。每 64KB flush，数据立即可达。配合水位线反压和双层限速（详见 [限速设计文档](#/docs/rate-limit)）。

## POST 请求（Git Push）

```go
if c.Request.ContentLength > 0 {
    req.ContentLength = c.Request.ContentLength  // 避免 Go 回退为 chunked
}
```