package handlers

import (
	"io"
	"math"
	"sync"
	"time"

	"github-proxy/config"
)

var (
	globalLimiter     *globalRateLimiter
	globalLimiterOnce sync.Once
)

func initGlobalLimiter() {
	cfg := config.GetConfig()
	globalLimiter = newGlobalRateLimiter(cfg.RateLimit.GlobalBytesPerSec)
}

func getGlobalLimiter() *globalRateLimiter {
	globalLimiterOnce.Do(initGlobalLimiter)
	return globalLimiter
}

// rateLimiter 单用户漏桶限速器（独立，每请求一个）。
type rateLimiter struct {
	perByte  time.Duration
	mu       sync.Mutex
	nextTime time.Time
}

func newRateLimiter(bytesPerSec int64) *rateLimiter {
	if bytesPerSec <= 0 {
		return &rateLimiter{}
	}
	return &rateLimiter{
		perByte: time.Second / time.Duration(bytesPerSec),
	}
}

func (r *rateLimiter) wait(n int) {
	if r.perByte == 0 {
		return
	}
	d := time.Duration(n) * r.perByte

	r.mu.Lock()
	now := time.Now()
	sleep := r.nextTime.Sub(now) + d
	r.nextTime = now.Add(d)
	r.mu.Unlock()

	if sleep > 0 {
		time.Sleep(sleep)
	}
}

// globalRateLimiter 共享令牌桶，所有未认证用户竞争同一个令牌池。
// 与 per-user 限速器独立工作，不产生延迟叠加。
type globalRateLimiter struct {
	mu       sync.Mutex
	rate     float64
	tokens   float64
	lastTime time.Time
}

func newGlobalRateLimiter(bytesPerSec int64) *globalRateLimiter {
	return &globalRateLimiter{
		rate:     float64(bytesPerSec),
		tokens:   float64(bytesPerSec),
		lastTime: time.Now(),
	}
}

func (g *globalRateLimiter) wait(n int) {
	if g.rate == 0 {
		return
	}

	g.mu.Lock()
	now := time.Now()
	elapsed := now.Sub(g.lastTime).Seconds()
	g.tokens += elapsed * g.rate
	if g.tokens > g.rate {
		g.tokens = g.rate
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

type rateLimitedWriter struct {
	writer io.Writer
	user   *rateLimiter
	global *globalRateLimiter
}

func (w *rateLimitedWriter) Write(p []byte) (int, error) {
	n := len(p)
	if w.user != nil {
		w.user.wait(n)
	}
	if w.global != nil {
		w.global.wait(n)
	}
	return w.writer.Write(p)
}

func (w *rateLimitedWriter) Flush() {
	if fw, ok := w.writer.(interface{ Flush() }); ok {
		fw.Flush()
	}
}
