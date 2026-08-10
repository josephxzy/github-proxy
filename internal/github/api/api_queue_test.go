package api

import (
	"context"
	"testing"
	"time"
)

// TestPickLimiter 按 URL 自动选择对应的限制器。
func TestPickLimiter(t *testing.T) {
	qs := NewAPIScopedLimiters(100, 200, 300, 400)

	tests := []struct {
		url  string
		name string
	}{
		{"https://api.github.com/search/repositories?q=proxy", "search"},
		{"https://api.github.com/repos/o/r/releases", "release"},
		{"https://api.github.com/repos/o/r", "repo"},
		{"https://api.github.com/rate_limit", "default"},
	}
	for _, tt := range tests {
		if got := qs.PickLimiter(tt.url); got.name != tt.name {
			t.Errorf("PickLimiter(%q).name = %q, want %q", tt.url, got.name, tt.name)
		}
	}
}

// TestCheckAPIQueueExempt 白名单用户（authenticated=true）应豁免限速。
func TestCheckAPIQueueExempt(t *testing.T) {
	InitGlobalAPILimiters(1, 1, 1, 1) // 上限 1 次/小时

	// 白名单用户即使上限已满也应立即通过
	if err := CheckAPIQueue(context.Background(), "https://api.github.com/search/x", true); err != nil {
		t.Errorf("白名单用户不应被限速: %v", err)
	}
	if err := CheckAPIQueue(context.Background(), "https://api.github.com/search/x", true); err != nil {
		t.Errorf("白名单用户第二次也不应被限速: %v", err)
	}
}

// TestCheckAPIQueueNilLimiter 限速器未初始化时应直接放行。
func TestCheckAPIQueueNilLimiter(t *testing.T) {
	GlobalAPILimiters = nil
	if err := CheckAPIQueue(context.Background(), "https://api.github.com/x", false); err != nil {
		t.Errorf("未初始化的限速器应放行: %v", err)
	}
}

// TestInitGlobalAPILimitersDefaults 非正数配置应回退到安全默认值。
func TestInitGlobalAPILimitersDefaults(t *testing.T) {
	InitGlobalAPILimiters(0, -1, 0, 5)

	if GlobalAPILimiters == nil {
		t.Fatal("InitGlobalAPILimiters 后 GlobalAPILimiters 不应为 nil")
	}
	if GlobalAPILimiters.searchLimiter.maxPerHour != 1200 {
		t.Errorf("search 默认值 = %d, want 1200", GlobalAPILimiters.searchLimiter.maxPerHour)
	}
	if GlobalAPILimiters.releaseLimiter.maxPerHour != 3333 {
		t.Errorf("release 默认值 = %d, want 3333", GlobalAPILimiters.releaseLimiter.maxPerHour)
	}
	if GlobalAPILimiters.defaultLimiter.maxPerHour != 5 {
		t.Errorf("default 应保留合法配置 = %d, want 5", GlobalAPILimiters.defaultLimiter.maxPerHour)
	}
}

// TestHourlyRateLimiterWindowReset 时间窗口过期后计数应重置。
func TestHourlyRateLimiterWindowReset(t *testing.T) {
	l := NewHourlyRateLimiter(1, "test")

	if err := l.Acquire(context.Background()); err != nil {
		t.Fatalf("首次 Acquire 失败: %v", err)
	}

	// 窗口未过期：第二次应阻塞等待（用带超时的 context 验证返回取消错误）
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := l.Acquire(ctx); err == nil {
		t.Error("窗口内第二次 Acquire 应阻塞直到超时")
	}

	// 把窗口起点拨回 2 小时前，模拟窗口过期，应能重新获取
	l.mu.Lock()
	l.windowStart = time.Now().Add(-2 * time.Hour)
	l.mu.Unlock()

	if err := l.Acquire(context.Background()); err != nil {
		t.Errorf("窗口过期后 Acquire 失败: %v", err)
	}
}
