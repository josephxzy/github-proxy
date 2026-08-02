# IP 三层限流体系

## 背景

仅有带宽限速不足以防止单 IP 高频刷代理。一个未认证 IP 可以发起大量小请求（搜索、列表、API），虽然不占带宽但会消耗 GitHub API 配额。需要在请求次数维度增加一层防护。

## 决策

**三层限流体系：IP 计数 → Token 白名单 → 带宽限速**。请求进入时先过 IP 限流（次数/小时），再检查 Token 白名单（免限速），最后过带宽限速（单流 + global）。Token 在白名单中的 IP 跳过 IP 计数和带宽限速。

## 当前规则

- 使用固定窗口计数器算法，每 IP 每小时限制 `IP_REQUEST_LIMIT` 次请求。
- IPv4 原样，IPv6 取前 64 位（同 /64 子网共享配额）。
- Token 在白名单中的 IP 直接放行，不计数。
- 前端路径（`/`, `/assets/*`, `/public/*`, `/ready`, `/favicon.ico`）不计入。
- 后台 goroutine 每 20 分钟清理超过 2 小时无访问的 IP 记录。
- 超限返回 429 Too Many Requests。
- Token 提取逻辑必须提前到中间件层，让 IP 限流和 `GitHubProxyHandler` 都能访问 `authenticated` 标志。

## 示例

**推荐做法**：
- 设置 `IP_REQUEST_LIMIT=500`（每小时每 IP 500 次请求）
- 将可信用户 token 加入 `TOKEN_WHITELIST` 跳过 IP 限流
- 前端路径纳入 `isFrontendPath` 排除列表

**禁止做法**：
- 将 IP 限流放在 `GitHubProxyHandler` 内部（无法在中间件层访问 `authenticated`）
- 不设置清理 goroutine（内存泄漏）
- 不归一化 IPv6（同子网用户共享配额但不公平）

## 失败模式

| 症状 | 原因 | 排查 |
|:---|:---|:---|
| 正常用户被 429 | 共享 IP 或 NAT 网络 | 调大 `IP_REQUEST_LIMIT` 或加入白名单 |
| 内存持续增长 | cleanup goroutine 未启动 | 检查 `StartCleanup` 是否在启动时调用 |
| 白名单用户仍被 429 | token 提取中间件未在 IP 限流前执行 | 检查中间件注册顺序 |
| IPv6 用户限流不准 | 未归一化，每 IP 独立计数 | 确认 `normalizeIP` 实现 |

## 相关模块

- `src/handlers/ip_limiter.go` — IP 限流中间件
- `src/handlers/github.go` — token 提取逻辑
- `src/config/config.go` — `IP_REQUEST_LIMIT`

## 来源文档

- IP 限流设计：`../design/ip-limit.md`