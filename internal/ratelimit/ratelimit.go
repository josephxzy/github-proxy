// Package ratelimit 提供带宽限速基础设施：
//
//	UserLimiter  单用户漏桶限速器（按字节计时的顺序等待）
//	GlobalLimiter 全局共享令牌桶限速器（所有未认证用户竞争同一令牌池）
//	Writer       组合两者并写入底层 io.Writer 的限速写包装器
//
// 与 handlers 包解耦：本包不依赖 gin，仅依赖配置。
package ratelimit

import (
	"io"
	"math"
	"sync"
	"time"

	"github-proxy/internal/config"
)

var (
	globalLimiter     *GlobalLimiter
	globalLimiterOnce sync.Once
)

// initGlobalLimiter 根据当前配置初始化全局限速器（只执行一次）。
// 使用 sync.Once 保证并发安全，首次调用 GetGlobalLimiter 时触发。
func initGlobalLimiter() {
	cfg := config.GetConfig()
	globalLimiter = NewGlobalLimiter(cfg.RateLimit.GlobalBytesPerSec)
}

// GetGlobalLimiter 获取全局限速器实例（懒初始化，线程安全）。
func GetGlobalLimiter() *GlobalLimiter {
	globalLimiterOnce.Do(initGlobalLimiter)
	return globalLimiter
}

// UserLimiter 单用户漏桶限速器（独立，每请求一个）。
// 按字节计算等待时间，控制单个未认证用户的下载带宽。
// bytesPerSec<=0 时创建空限速器（perByte=0，wait 直接返回，不限速）。
type UserLimiter struct {
	perByte  time.Duration
	mu       sync.Mutex
	nextTime time.Time
}

// NewUserLimiter 创建单用户限速器。
// bytesPerSec<=0 表示不限速。
func NewUserLimiter(bytesPerSec int64) *UserLimiter {
	if bytesPerSec <= 0 {
		return &UserLimiter{}
	}
	return &UserLimiter{
		perByte:  time.Second / time.Duration(bytesPerSec),
		nextTime: time.Now(), // 初始化基准时间，避免首次 wait 因零值时间戳不等待
	}
}

// wait 阻塞 n 字节所需的限速等待时间。
// 采用漏桶算法：每字节固定间隔 perByte，多请求之间按 nextTime 累积排队。
func (l *UserLimiter) wait(n int) {
	if l.perByte == 0 {
		return
	}
	d := time.Duration(n) * l.perByte

	l.mu.Lock()
	now := time.Now()
	sleep := l.nextTime.Sub(now) + d
	// nextTime 推进到"本次预计完成时刻"：有积压（nextTime>now）时按
	// now+sleep 推进，保留排队语义，避免并发写入超出速率。
	l.nextTime = now.Add(sleep)
	l.mu.Unlock()

	if sleep > 0 {
		time.Sleep(sleep)
	}
}

// GlobalLimiter 共享令牌桶，所有未认证用户竞争同一个令牌池。
// 与 per-user 限速器独立工作，不产生延迟叠加。
type GlobalLimiter struct {
	mu       sync.Mutex
	rate     float64
	tokens   float64
	lastTime time.Time
}

// NewGlobalLimiter 创建全局限速器（令牌桶，容量为 1/10 秒的令牌量）。
func NewGlobalLimiter(bytesPerSec int64) *GlobalLimiter {
	return &GlobalLimiter{
		rate:     float64(bytesPerSec),
		tokens:   float64(bytesPerSec),
		lastTime: time.Now(),
	}
}

// wait 阻塞 n 字节所需的令牌等待时间。
// 令牌桶按时间速率补充令牌，桶容量限制为 1/10 秒的令牌量（突发缓冲）。
// 注意：与 per-user 限速器独立工作，全局限速不产生延迟叠加。
func (g *GlobalLimiter) wait(n int) {
	if g.rate == 0 {
		return
	}

	g.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(g.lastTime).Seconds()
	g.tokens += elapsed * g.rate
	if g.tokens > g.rate/10 {
		g.tokens = g.rate / 10
	}
	g.lastTime = now

	need := float64(n)
	if g.tokens >= need {
		g.tokens -= need
		g.mu.Unlock()
		return
	}

	deficit := need - g.tokens
	g.tokens = 0
	sleepTime := time.Duration(math.Ceil(deficit / g.rate * float64(time.Second)))
	g.lastTime = now.Add(sleepTime)
	g.mu.Unlock()

	if sleepTime > 0 {
		time.Sleep(sleepTime)
	}
}

// Writer 组合单用户与全局限速的写包装器。
// writer 为实际写入对象（通常是带 Flush 的 flushingWriter）。
// user/global 为 nil 时跳过对应的限速逻辑（如白名单用户 user 为 nil）。
type Writer struct {
	writer io.Writer
	user   *UserLimiter
	global *GlobalLimiter
}

// NewWriter 创建限速写包装器。
func NewWriter(w io.Writer, user *UserLimiter, global *GlobalLimiter) *Writer {
	return &Writer{writer: w, user: user, global: global}
}

// Write 先按字节数等待限速，再写入底层 writer。
func (w *Writer) Write(p []byte) (int, error) {
	n := len(p)
	if w.user != nil {
		w.user.wait(n)
	}
	if w.global != nil {
		w.global.wait(n)
	}
	return w.writer.Write(p)
}

// Flush 刷新底层 writer（若其支持 Flush）。
func (w *Writer) Flush() {
	if fw, ok := w.writer.(interface{ Flush() }); ok {
		fw.Flush()
	}
}
