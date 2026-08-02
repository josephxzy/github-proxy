# 限速与稳定性设计

## 架构全景

系统分为四层：用户 → Go 程序 → 操作系统内核 → GitHub。

```
用户A (限速 2MB/s) ──┐
用户B (限速 5MB/s) ──┤
用户C (无限制)     ──┼──→ [Go 程序] ──→ [内核 SO_RCVBUF 4MB] ──→ GitHub
                      │    ▲
                单流  │
                漏桶    │
                     ┌────┴────┐
                     │ global  │
                     │ 令牌桶   │
                     └─────────┘
```

核心矛盾：GitHub 灌数据的速度远超用户消费速度。内核不会帮我们限速——必须在 Go 层完成。

## 双层限速

`rateLimitedWriter.Write()` 对每次写入依次过两道限速：

```go
func (w *rateLimitedWriter) Write(p []byte) (int, error) {
    if w.user != nil   { w.user.wait(len(p)) }   // ① 单流漏桶
    if w.global != nil { w.global.wait(len(p)) } // ② global 令牌桶
    return w.writer.Write(p)
}
```

| | 单流限速器 | global 限速器 |
|:---|:---|:---|
| 算法 | 漏桶（独立，每请求一个） | 共享令牌桶（全局单例） |
| 作用 | 限制单个下载流的峰值带宽 | 限制所有未认证用户的总带宽 |
| 资源 | 每个下载流独占一个 | 所有下载流竞争同一个令牌池 |
| 白名单 | 跳过 | 跳过 |

> **注意**：单流限速器按请求独立创建（每连接一个）。同名用户 N 个并发下载将获得 N × `downloadBytesPerSec` 的带宽。全局令牌桶提供真正的跨用户总带宽上限。

### 为什么不是简单的延迟相加？

单流漏桶执行 `wait()` 期间，global 令牌桶在后台持续累积令牌。单流延迟结束后，global 通常已有足够令牌，无需额外等待。只有全局带宽真正被打满时，global 令牌池才会耗尽。

```
场景1：1 个用户，单流=5MB/s，global=50MB/s
  wait(32KB) = 0.006s → global 令牌充足 → 速率 5MB/s ✓

场景2：20 个用户各 1 流，单流=5MB/s，global=50MB/s
  每人 global.wait 追加延迟 → 有效速率 ≈ 2.5MB/s × 20 = 50MB/s ✓

场景3：1 个用户 4 个并发下载，单流=5MB/s（无 global 限制）
  4 流各自独立限速 → 总速率最高可达 20MB/s
```

### 暂停恢复行为

单流漏桶**没有空闲重置逻辑**。暂停再久，`nextTime` 始终保持着上次写入时设置的值：

```
暂停 10s → nextTime 停留在 10s 前 →
恢复后首 chunk：sleep 为负 → 不 sleep，立即写出（仅 32KB 突发）
后续所有 chunk：sleep ≈ d × (已累积 chunk 数) → 严格按 rate 限速
```

> 删除了一秒空闲重置（旧代码 `if nextTime.Before(now.Add(-1s)) { nextTime = now }`），原因：重置导致暂停期间所有积压数据在恢复瞬间涌出，等同未限速。

## 水位线反压

```
GitHub ──→ SO_RCVBUF(4MB) ──→ WaterlineBuffer(8MB) ──→ 限速器 ──→ 客户端
                                   80% 暂停 / 20% 恢复
```

生产者（从 GitHub 全速读取）→ 写入环形缓冲区。缓冲区 80% 时暂停读取，20% 时恢复。消费者从缓冲区读取，经限速器写入客户端。

### 完整流序

**阶段一：正常下载中**

```
内部缓冲区 ████████░░░░░░░░░░  35%
       │
       ├─→ 从内核全速或自然速度读取（50MB/s）
       └─→ 多个用户 token bucket 控制写入速度（合计 10MB/s）
       
净流入 = 50 - 10 = 40MB/s → 缓冲区快速增长
但每次只读 32KB，读完立刻检查水位，涨到 80% 立即暂停
```

**阶段二：触达高水位 → 暂停读取**

```
内部缓冲区 ████████████████░░  80%  → 触发高水位
       │
       └─→ Go 程序停止调用 resp.Body.Read()
           但内核缓冲区中还有已接收的数据
           GitHub 也还不知道要暂停——仍有飞行数据在路上

内核层：
        GitHub ──飞行数据(2.5MB)──→ [内核缓冲区 ████████░░]
        
        内核缓冲区继续向内部缓冲区吐数据（Go 还没来取）
        同时 GitHub 的飞行数据还在往内核缓冲区灌

Go 层：
        停止 Read() 的瞬间，内部缓冲区可能微涨到 85%~90%
        这多出来的 5%~10% 就是内核缓冲区里积压的数据
```

**阶段三：用户消费，缓冲区下降**

```
内部缓冲区 ██████████░░░░░░░░  50%
       │
       └─→ 用户 token bucket 持续消费（合计 10MB/s）
           此时没有新数据进来，缓冲区只出不进
```

**阶段四：触达低水位 → 恢复读取**

```
内部缓冲区 ████░░░░░░░░░░░░░░  20%  → 触发低水位
       │
       └─→ Go 程序恢复调用 resp.Body.Read()

恢复瞬间：
        内核缓冲区可能已经排空或接近排空
        TCP 窗口重新打开 → 通知 GitHub 可以继续发送
        
        GitHub 数据需要 RTT(50ms) 才能到达
        在这 50ms 空窗期内，用户仍在消费
        内部缓冲区可能从 20% 继续降到 ~15%
        
        RTT 过后 → GitHub 数据到达 → 缓冲区开始重新上涨
```

### 完整振荡曲线

```
 80%  ┤         ╱╲          ╱╲
      ┤        ╱  ╲        ╱  ╲
      ┤       ╱    ╲      ╱    ╲       ← 暂停读取,
 50%  ┤──────╱      ╲────╱      ╲────    缓冲区下降
      ┤     ╱        ╲  ╱        ╲
      ┤    ╱          ╲╱          ╲
 20%  ┤───╱            ╲            ╲── ← 恢复读取,
      ┤  ╱              ╲            ╲    GitHub 数据到达
      ┤ ╱ 读取(自然速度)  ╲ 暂停       ╲
      ┤╱  写入(10MB/s)    ╲ 写入(10MB/s)╲
     ─┴────────────────────────────────────→ 时间
        填充             排空           填充
```

### 多文件并发

每个文件有自己的 HTTP 连接、独立的内核缓冲区、独立的 producer goroutine + WaterlineBuffer。

```
File A ──→ [缓冲A + 水位A] ──┐
File B ──→ [缓冲B + 水位B] ──┤
File C ──→ [缓冲C + 水位C] ──┼──→ [用户X bucket] ──→ 用户X
                              ├──→ [用户Y bucket] ──→ 用户Y
                              └──→ [用户Z 不限速]  ──→ 用户Z
```

| 场景 | File A | File B | 结果 |
|:---|:---|:---|:---|
| 两个都快 | 缓冲→80% 暂停 | 缓冲→80% 暂停 | 各自独立暂停/恢复 |
| A 快 B 慢 | 缓冲→80% 暂停 | 缓冲低位，持续读 | B 不受 A 暂停影响 |
| A 暂停中 B 启动 | 保持暂停 | 从 0% 开始填充 | B 正常启动 |
| A GitHub 慢(3MB/s) | 缓冲低位 | — | 水位线永不触发 |

实际使用中常出现：从 GitHub 同时下多个文件，一个 50MB/s，另一个仅 3MB/s。水位线完美适配——快文件高频暂停，慢文件自然流畅，互不干扰。

### 缓冲区大小选择

| 服务器带宽 | 飞行数据 | **推荐** | 理由 |
|:---|:---|:---|:---|
| 100Mbps（~12MB/s） | 0.6MB | **8MB** | 填充 0.32s，不震荡 |
| 500Mbps（~60MB/s） | 3MB | **16MB** | 填充 0.2s |
| 1Gbps（~125MB/s） | 6.25MB | **32MB** | 填充 0.15s |

缓冲区过小 → 填充太快 → 高频暂停/恢复 → 有效吞吐下降。调大让每次振荡周期至少几百毫秒。

### 真环形缓冲区

```go
// Write — 满时阻塞，绝不短写丢数据
func (b *WaterlineBuffer) Write(data []byte) int {
    for total < len(data) {
        for b.count == b.capacity { b.cond.Wait() }  // 满则等
        // 写入，指针取模环绕
    }
    if b.usage() >= 0.8 { b.paused = true }
}

// Read — 关闭且读空时返回 io.EOF
func (b *WaterlineBuffer) Read(p []byte) (int, error) {
    for b.count == 0 && !b.closed { b.cond.Wait() }
    if b.count == 0 { return 0, io.EOF }
    // 读取，指针取模环绕
    return n, nil
}
```

> 早期的线性缓冲区实现 `copy(b.data[b.writePos:], data)` 在写满时静默丢数据，是"下载末尾网络错误"的根源。真环形 + 满时阻塞彻底解决。

## 完整管道

```
resp.Body ──→ producer goroutine ──→ WaterlineBuffer ──→ rateLimitedWriter ──→ 客户端
  (GitHub)     全速读取，受水位     80%/20% 暂停/恢复     单流 + global
               线暂停控制                                  
```

| 文件 | 职责 |
|:---|:---|
| `handlers/waterline.go` | 真环形缓冲区，80%/20% 暂停/恢复 |
| `handlers/response_writer.go` | producer/consumer 双 goroutine |
| `handlers/rate_limit.go` | 单流漏桶 + global 共享令牌桶 |
| `pkg/network/http_client_*.go` | `SO_RCVBUF = 4MB`（每连接） |

## 配置

| 配置 | 环境变量 | 说明 |
|:---|:---|:---|
| `downloadBytesPerSec` | `DOWNLOAD_RATE` | 单用户下载上限（B/s），0=不限 |
| `globalBytesPerSec` | `GLOBAL_RATE` | 全局共享带宽上限（B/s），0=不限 |
| `bufferSize` | `BUFFER_SIZE` | 水位线缓冲区大小，默认 8MB |

白名单用户（token 在 `TOKEN_WHITELIST` 中）跳过所有限速器，直接透传写入。