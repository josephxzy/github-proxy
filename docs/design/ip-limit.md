# IP 限流设计

## 目标

对未认证 IP 按小时限制请求次数，防止单 IP 高频刷代理。与现有 Token 白名单和带宽限速互补。

## 三层限流体系

```
请求进入
  ├─ IP 限流（次数/小时）    ← 新增，本次实现
  ├─ Token 白名单（带宽免限） ← 已有
  └─ 带宽限速（单流 + global） ← 已有
```

| 层 | 维度 | 粒度 | 白名单豁免 |
|:---|:---|:---|:---|
| IP 限流 | 请求次数 | 每小时每 IP | Token 在白名单 → 跳过 |
| 带宽限速 | 下载速度 | 单流 + global | Token 在白名单 → 跳过 |

## 实现方案

### 新增文件

| 文件 | 职责 |
|:---|:---|
| `handlers/ip_limiter.go` | IP 限流中间件 + 计数逻辑 |

### IP 限流器

```go
type IPRateLimiter struct {
    mu           sync.Mutex
    entries      map[string]*ipEntry
    limit        int           // 每小时每 IP 最大请求数
    windowSize   time.Duration // 统计窗口（1 小时）
    cleanupAge   time.Duration // 清理阈值（2 小时无访问则删除）
}

type ipEntry struct {
    count      int
    windowStart time.Time
}
```

**算法：固定窗口计数器**

1. 请求到达时，取出 IP 对应的 entry
2. 如果 `now - windowStart >= windowSize`：重置 `count = 0, windowStart = now`
3. `count++`
4. 如果 `count > limit`：返回 429 Too Many Requests
5. 否则放行

**IP 归一化**：IPv4 原样；IPv6 取前 64 位（同 /64 子网共享配额）。

### 中间件

```go
func (l *IPRateLimiter) Middleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. 白名单 token 用户直接放行
        if getAuthFromContext(c) {
            c.Next()
            return
        }

        // 2. 前端路径不计入
        if isFrontendPath(c.Request.URL.Path) {
            c.Next()
            return
        }

        // 3. 获取 IP
        ip := normalizeIP(c.ClientIP())

        // 4. 检查
        if !l.allow(ip) {
            c.AbortWithStatus(http.StatusTooManyRequests)
            return
        }

        c.Next()
    }
}
```

### 前端路径排除

```go
func isFrontendPath(path string) bool {
    return path == "/" ||
           path == "/ready" ||
           path == "/favicon.ico" ||
           path == "/favicon.svg" ||
           strings.HasPrefix(path, "/assets/") ||
           strings.HasPrefix(path, "/public/")
}
```

### 清理

```go
func (l *IPRateLimiter) cleanup() {
    l.mu.Lock()
    defer l.mu.Unlock()
    now := time.Now()
    for ip, entry := range l.entries {
        if now.Sub(entry.windowStart) > l.cleanupAge {
            delete(l.entries, ip)
        }
    }
}
```

后台 goroutine 每 20 分钟执行一次。

### 配置

```toml
[rateLimit]
ipRequestLimit = 500    # 每 IP 每小时最大请求数，0=不限
```

环境变量：`IP_REQUEST_LIMIT=500`

## 注册位置

`handlers/github.go` 的 `GitHubProxyHandler` **之前**，作为 Gin 中间件注册在路由上：

```go
// server/router.go
limiter := handlers.NewIPRateLimiter(cfg)
router.Use(limiter.Middleware())
```

> 注意：必须放在 `GitHubProxyHandler` 之前执行，因为它需要在 Gin context 中已有 `authenticated` 标志。而 `authenticated` 是在 `GitHubProxyHandler` 内部设置的。所以要么把 IP 限流放在 `NoRoute` handler 内部，要么把 token 提取提前到中间件层。

**解决方案**：把 token 提取逻辑从 `GitHubProxyHandler` 中抽出来，作为独立中间件提前执行。这样 IP 限流和 `GitHubProxyHandler` 都能访问到 `authenticated` 标志。

## 改造范围

| 文件 | 改动 |
|:---|:---|
| `handlers/ip_limiter.go` | **新建** — IP 限流器 |
| `handlers/github.go` | 拆分：token 提取提前到中间件 |
| `server/router.go` | 注册 token 中间件 + IP 限流中间件 |
| `config/config.go` | 新增 `IPRequestLimit` |
| `config.toml` | 新增配置项 |

## 边界情况

| 场景 | 行为 |
|:---|:---|
| IP 在白名单 token 用户 | 跳过 IP 计数 |
| IP 超过限额 | 429，等待窗口重置 |
| IP 长时间不活跃 | cleanup 删除，下次请求重新计数 |
| 窗口切换 | 下一个小时重置计数 |
| IPv6 | 归一化到 /64，同子网共享 |
| 前端页面 | 不计入（`/`, `/assets/*`, `/ready` 等） |