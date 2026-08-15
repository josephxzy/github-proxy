package ratelimit

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github-proxy/internal/config"
)

// TestUserLimiterDisabled 限速值为 0 时应完全不限速（wait 立即返回）。
func TestUserLimiterDisabled(t *testing.T) {
	l := NewUserLimiter(0)
	start := time.Now()
	l.wait(1024 * 1024) // 1MB
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("限速禁用时 wait(1MB) 耗时 %v，应接近 0", elapsed)
	}
}

// TestUserLimiterThrottles 验证漏桶限速的实际吞吐量近似等于配置值。
// 1000 字节/秒 → 写入 1000 字节应约耗时 1 秒（允许 30% 误差避免 flaky）。
func TestUserLimiterThrottles(t *testing.T) {
	l := NewUserLimiter(1000)
	start := time.Now()
	l.wait(1000)
	elapsed := time.Since(start)

	if elapsed < 700*time.Millisecond || elapsed > 1300*time.Millisecond {
		t.Errorf("wait(1000B) 耗时 %v，期望约 1s", elapsed)
	}
}

// TestUserLimiterCumulative 串行等待按字节数线性累积（2×500B @2000B/s ≈ 500ms）。
func TestUserLimiterCumulative(t *testing.T) {
	l := NewUserLimiter(2000) // 0.5ms/字节
	start := time.Now()
	l.wait(500)
	l.wait(500)
	elapsed := time.Since(start)

	// 两次共 1000 字节 → 约 500ms（允许 30% 误差避免 flaky）
	if elapsed < 350*time.Millisecond || elapsed > 800*time.Millisecond {
		t.Errorf("两次 wait(500B) 耗时 %v，期望约 500ms", elapsed)
	}
}

// TestGlobalLimiterDisabled 全局限速值为 0 时 wait 立即返回。
func TestGlobalLimiterDisabled(t *testing.T) {
	g := NewGlobalLimiter(0)
	start := time.Now()
	g.wait(1024 * 1024)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("全局限速禁用时 wait 耗时 %v，应接近 0", elapsed)
	}
}

// TestGlobalLimiterBurst 桶容量内（rate/10 的突发缓冲）的请求不应等待。
func TestGlobalLimiterBurst(t *testing.T) {
	g := NewGlobalLimiter(1000) // 容量 = 100 字节
	start := time.Now()
	g.wait(50)
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Errorf("桶容量内的 wait(50B) 耗时 %v，不应等待", elapsed)
	}
}

// TestGlobalLimiterThrottles 超出桶容量的部分应按速率补足令牌。
// 速率 1000 B/s，容量 100B：wait(1100) 需等待 (1100-100)/1000 ≈ 1s。
func TestGlobalLimiterThrottles(t *testing.T) {
	g := NewGlobalLimiter(1000)
	start := time.Now()
	g.wait(1100)
	elapsed := time.Since(start)

	if elapsed < 700*time.Millisecond || elapsed > 1500*time.Millisecond {
		t.Errorf("wait(1100B) 耗时 %v，期望约 1s", elapsed)
	}
}

// TestWriterPassThrough 无限速器（白名单场景：user/global 均为 nil）时直接透传。
func TestWriterPassThrough(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, nil, nil)

	n, err := w.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("Write = %d, %v, want 5, nil", n, err)
	}
	if buf.String() != "hello" {
		t.Errorf("内容被篡改: %q", buf.String())
	}
}

// TestWriterWithUserLimiter 带用户限速器时写入字节数完整、且发生限速等待。
func TestWriterWithUserLimiter(t *testing.T) {
	var buf bytes.Buffer
	w := NewWriter(&buf, NewUserLimiter(1000), nil)

	start := time.Now()
	n, err := w.Write(make([]byte, 1000))
	if err != nil || n != 1000 {
		t.Fatalf("Write = %d, %v, want 1000, nil", n, err)
	}
	if elapsed := time.Since(start); elapsed < 700*time.Millisecond {
		t.Errorf("限速写入耗时 %v，期望 ≥700ms", elapsed)
	}
	if buf.Len() != 1000 {
		t.Errorf("写入字节数 = %d, want 1000", buf.Len())
	}
}

// TestWriterFlush Flush 应转发给底层支持 Flush 的 writer。
type flushRecorder struct {
	io.Writer
	flushed bool
}

func (f *flushRecorder) Flush() { f.flushed = true }

func TestWriterFlush(t *testing.T) {
	rec := &flushRecorder{Writer: &bytes.Buffer{}}
	w := NewWriter(rec, nil, nil)
	w.Flush()
	if !rec.flushed {
		t.Error("Flush 未转发到底层 writer")
	}
}

// TestNewGlobalLimiterDefaults 创建不同速率的全局限制器应得到不同 rate。
func TestNewGlobalLimiterDefaults(t *testing.T) {
	g := NewGlobalLimiter(5000)
	if g.rate != 5000 {
		t.Errorf("rate = %v, want 5000", g.rate)
	}
	if g.tokens != 5000 {
		t.Errorf("初始 tokens = %v, want 5000", g.tokens)
	}
}

// TestGlobalLimiterTracksConfig 单例限速器必须跟随配置变化重建，
// 而不能用 sync.Once 把首次调用时的速率永久固化。
// 回归场景：测试套件中先以 GLOBAL_RATE=2000 初始化单例后，
// 后续不限速（0）配置的用例仍被 2KB/s 悄悄限速。
func TestGlobalLimiterTracksConfig(t *testing.T) {
	t.Setenv("GLOBAL_RATE", "2000")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	g1 := GetGlobalLimiter()
	if g1.rate != 2000 {
		t.Fatalf("首次 GetGlobalLimiter rate = %v, want 2000", g1.rate)
	}

	// 切换为不限速配置：单例必须重建为 rate=0
	t.Setenv("GLOBAL_RATE", "0")
	if err := config.LoadConfig(); err != nil {
		t.Fatalf("LoadConfig 失败: %v", err)
	}
	g2 := GetGlobalLimiter()
	if g2.rate != 0 {
		t.Fatalf("配置切换后 GetGlobalLimiter rate = %v, want 0（不限速）", g2.rate)
	}
	if g2 == g1 {
		t.Error("配置变化后应重建限速器实例")
	}

	// 重建后等待应零耗时
	start := time.Now()
	g2.wait(1024 * 1024)
	if elapsed := time.Since(start); elapsed > 50*time.Millisecond {
		t.Errorf("重建后 wait(1MB) 耗时 %v，应接近 0", elapsed)
	}
}
