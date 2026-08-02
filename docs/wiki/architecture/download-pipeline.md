# 下载管道与水位线反压

## 背景

GitHub 服务器带宽极高（可达 100MB/s+），而用户下载速度通常只有 1-10MB/s。如果代理无限制地从 GitHub 读取数据，内核缓冲区会迅速填满，TCP 窗口归零，最终导致连接中断。操作系统内核不会帮我们做应用层限速——必须在 Go 层实现反压。

## 决策

**生产-消费者 + 环形缓冲区 + 水位线暂停/恢复**。从 GitHub 全速读取（生产者），写入真环形缓冲区；80% 时暂停读取，20% 时恢复。消费者从缓冲区读取，经限速器写入客户端。真环形缓冲区在满时阻塞写入，绝不短写丢数据。

## 当前规则

- 使用 `WaterlineBuffer`（真环形缓冲区）连接生产者和消费者，禁止使用线性缓冲区。
- 高水位 = 80%，低水位 = 20%。达到高水位暂停 `resp.Body.Read()`，降到低水位恢复。
- 缓冲区满时 `Write()` 必须阻塞等待，禁止短写。
- 每个下载连接有独立的 producer goroutine + WaterlineBuffer，互不干扰。
- 每 64KB flush 一次到客户端，数据立即可达。
- Content-Length 探测失败时走 chunked 传输，绝不填入估算值。

## 示例

**推荐做法**：
- 设置 `BUFFER_SIZE=8388608`（8MB），适配 100Mbps 带宽
- 高带宽服务器（500Mbps+）调大至 16MB 或 32MB
- 使用 `WaterlineBuffer.Read/Write` 的真环形实现

**禁止做法**：
- 使用线性缓冲区 `copy(dst[pos:], src)` 在满时短写
- 在高水位时继续调用 `resp.Body.Read()`
- 探测不到 Content-Length 时填入估算值

## 失败模式

| 症状 | 原因 | 排查 |
|:---|:---|:---|
| 下载末尾报网络错误 | 线性缓冲区满时短写丢数据 | 确认使用真环形缓冲区 |
| 进度条消失 | Content-Length 探测失败 | 检查 Range bytes=0-0 预检日志 |
| 下载速度忽快忽慢 | 缓冲区过小导致高频暂停/恢复 | 调大 `BUFFER_SIZE` |
| 客户端收不到数据 | 64KB flush 未执行 | 检查 `response_writer.go` 的 flush 逻辑 |

## 相关模块

- `src/handlers/waterline.go` — 真环形缓冲区
- `src/handlers/response_writer.go` — producer/consumer 双 goroutine
- `src/handlers/proxy_download.go` — 下载请求处理
- `src/pkg/network/` — `SO_RCVBUF` 设置

## 来源文档

- 下载与断点续传：`../design/download.md`
- 限速与稳定性设计：`../design/rate-limit.md`