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

### 6. Git 推送被拒：PAT 权限不足 / 凭据过期（Windows）

**症状**：`git push` 被 GitHub 拒绝，常见两类报错：
- `refusing to allow a Personal Access Token to create or update workflow .github/workflows/... without workflow scope` —— 推送含 workflow 文件变更时缺 `workflow` scope
- `remote: Support for password authentication was removed` / 403 —— 凭据过期或用了旧 token

**根因**：Git for Windows 的 `credential.helper=manager`（GCM）缓存了旧 PAT。关键认知：
- **git 推送只用 GCM 的 `git:https://github.com` 条目**，与 GitHub CLI 的 `gh:https://github.com` 条目（username=`gh-token`）互不相通；
  只更新凭据管理器里的 `gh-token` 对 git push 无效。
- PAT 缺 `workflow` scope 时，任何涉及 `.github/workflows/` 的推送都会被拒（细粒度 token 需单独授权 Workflows 权限）。

**排查**：
```bash
# 查看 git 实际使用的凭据来源（只显示 username，勿泄露 password）
printf "protocol=https\nhost=github.com\n\n" | git credential fill | grep username
```

**修复**：
```bash
# 1. 清除旧凭据，强制重新认证
printf "protocol=https\nhost=github.com\n\n" | git credential reject
# 或 cmdkey /delete:git:https://github.com

# 2. 重新 push，GCM 会弹出登录窗口，输入带完整权限的 PAT
git push
```

**不应使用的短期手段**：反复重试 push（凭据未变结果不变）；只改凭据管理器里的 `gh-token` 条目。

### 7. 不限速仍卡速（300KB/s 平台期 / 忽满忽低）

**症状**：`downloadBytesPerSec`/`globalBytesPerSec` 已设为 0（不限速），但下载速度长期卡在 ~300KB/s，之后可能突然跑满，再跌到几十 KB/s。

**根因**：TCP 传输瓶颈，需要区分**上游**（VPS→GitHub）与**下游**（VPS→客户端）两条腿：
- 上游腿：GitHub 不同主机从国内/香港路由质量差异巨大（实测：release CDN ~2.5MB/s 稳定，codeload ~300KB~1.4MB 抖动，github.com 延迟 ~1.9s）。archive / git 慢速通常就是上游主机拥塞。
- 下游腿：跨境链路（RTT 200ms+）下 TCP 拥塞控制的典型波形——慢启动窗口不足（~300KB/s）→ cwnd 指数增长灌满链路（突然满速）→ bufferbloat 丢包 → RTO 坍缩到 1 段（~20KB/s）。这是 CUBIC 在丢包路径上的标准行为，与代理代码无关。

**排查**（在 VPS 上交错测，隔离两条腿）：
```bash
# ① 上游直连 codeload（archive 实际上游）
curl -o /dev/null -w "codeload: %{speed_download} B/s\n" --max-time 30 \
  "https://codeload.github.com/golang/go/zip/refs/heads/master"
# ② 上游直连 release CDN（对照）
curl -L -o /dev/null -w "release-cdn: %{speed_download} B/s\n" --max-time 30 \
  "https://github.com/josephxzy/github-proxy/releases/download/v1.3.7/github-proxy-linux-amd64"
# ③ 回环走代理（隔离客户端腿：git 流不限速，用白名单 token 或设限速为 0）
cd /tmp && time git clone "http://TOKEN@127.0.0.1:5001/golang/go.git" x
```

**修复**（部署层，代理代码不再做套接字调优）：
- 上游腿：`config.toml` 配 `proxy` 让 VPS→GitHub 走优质中转线路；或换 CN2 GIA / CMIN2 等对 GitHub 优化线路的 VPS
- 下游腿（VPS 是发送端）：开 BBR——`sysctl -w net.core.default_qdisc=fq && sysctl -w net.ipv4.tcp_congestion_control=bbr`，能显著消除 CUBIC 的坍缩波形
- git clone：v1.3.8 起一律不限速（限速会拉长传输、放大断流窗口，而 git-upload-pack 无续传能力）

**不应使用的短期手段**：只调大 `BUFFER_SIZE`——水位线缓冲只影响上游预取节奏，治不了 TCP 拥塞波形。

### 8. 测试套件中未认证下载被悄悄限速（全局限速器单例污染）

**症状**：`go test ./...` 中排在 `TestGitCloneWhitelistThrottleExempt`（`GLOBAL_RATE=2000`）之后的下载用例变慢或超时。

**根因**：`ratelimit.GetGlobalLimiter()` 曾用 `sync.Once` 把首次调用时的配置速率永久固化。测试切换环境变量重载配置后，单例仍按旧速率限速，后续每个未认证下载都被 2KB/s 卡死。

**修复**：`GetGlobalLimiter()` 改为互斥锁懒初始化，并随 `config.GetConfig()` 的当前速率变化自动重建限速器。

## 相关模块

- `internal/waterline/waterline.go`
- `internal/handlers/ip_limiter.go`
- `internal/handlers/github.go`
- `internal/handlers/proxy_download.go`
- `internal/server/server.go`
- `internal/ratelimit/ratelimit.go`