package github

import (
	"context"
	"strings"
	"sync"
	"time"
)

// HourlyRateLimiter 小时级速率限制器。
// 按固定时间窗口（每小时）限制请求次数，避免触发上游限流。
//
// 工作原理：
//   - 维护一个时间窗口起始时间和窗口内请求数计数器
//   - 每次请求时检查当前窗口是否已满
//   - 如果已满，则阻塞等待直到下一个窗口开始
//   - 新窗口开始时重置计数器
type HourlyRateLimiter struct {
	mu           sync.Mutex // 互斥锁，保证并发安全
	windowStart  time.Time  // 当前时间窗口的起始时间
	requestCount int        // 当前窗口内的请求数
	maxPerHour   int        // 每小时最大允许请求数
	name         string     // 限制器名称（用于日志）
}

// NewHourlyRateLimiter 创建小时级速率限制器。
//
// 参数:
//   - maxPerHour: 每小时最大请求次数
//   - name: 限制器名称，仅用于日志标识
func NewHourlyRateLimiter(maxPerHour int, name string) *HourlyRateLimiter {
	return &HourlyRateLimiter{
		maxPerHour:  maxPerHour,
		name:        name,
		windowStart: time.Now(), // 初始化为当前时间
	}
}

// Acquire 阻塞等待直到获取执行权。
// 若当前窗口内请求数已达上限，则等待下一窗口。
//
// 行为说明：
//   - 如果当前窗口未满，立即增加计数并返回
//   - 如果当前窗口已满，计算等待时间并阻塞
//   - 支持通过 context 取消等待
//   - 超时或取消时返回 context 的错误
func (l *HourlyRateLimiter) Acquire(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	// 检查是否需要开启新窗口（距离上次窗口开始已超过1小时）
	if now.Sub(l.windowStart) >= time.Hour {
		l.windowStart = now
		l.requestCount = 0
	}

	// 如果当前窗口已满，需要等待
	for l.requestCount >= l.maxPerHour {
		waitUntil := l.windowStart.Add(time.Hour)
		waitDur := time.Until(waitUntil)

		// 边界情况：如果等待时间为0或负数，直接重置窗口
		if waitDur <= 0 {
			l.windowStart = time.Now()
			l.requestCount = 0
			break
		}

		// 释放锁并等待
		l.mu.Unlock()
		timer := time.NewTimer(waitDur)
		select {
		case <-timer.C:
			// 等待完成，重新获取锁
		case <-ctx.Done():
			// 上下文被取消
			timer.Stop()
			l.mu.Lock()
			return ctx.Err()
		}
		timer.Stop()
		l.mu.Lock()

		// 重新检查窗口状态（可能已经被其他 goroutine 更新）
		now = time.Now()
		if now.Sub(l.windowStart) >= time.Hour {
			l.windowStart = now
			l.requestCount = 0
		}
	}

	// 增加请求计数
	l.requestCount++
	return nil
}

// APIScopedLimiters 按 API 类型管理的多个速率限制器。
// 每个 API 类型持有一个独立限制器，互不影响。
//
// 设计目的：
// 不同类型的 API 有不同的速率限制需求：
// 不同类型的 API 有不同的速率限制需求：
//   - 搜索 API：较严格的限制（1200/小时）
//   - Release API：较宽松的限制（3333/小时）
//   - 仓库 API：标准限制（3333/小时）
//   - 其他 API：默认限制（3333/小时）
type APIScopedLimiters struct {
	searchLimiter  *HourlyRateLimiter // 搜索 API 限制器
	releaseLimiter *HourlyRateLimiter // 发布版本 API 限制器
	repoLimiter    *HourlyRateLimiter // 仓库 API 限制器
	defaultLimiter *HourlyRateLimiter // 默认 API 限制器
}

// NewAPIScopedLimiters 创建按类型划分的 API 限制器集合。
//
// 参数:
//   - searchPerHour: 搜索 API 每小时限制
//   - releasePerHour: Release API 每小时限制
//   - repoPerHour: 仓库 API 每小时限制
//   - defaultPerHour: 其他 API 每小时限制
func NewAPIScopedLimiters(searchPerHour, releasePerHour, repoPerHour, defaultPerHour int) *APIScopedLimiters {
	return &APIScopedLimiters{
		searchLimiter:  NewHourlyRateLimiter(searchPerHour, "search"),
		releaseLimiter: NewHourlyRateLimiter(releasePerHour, "release"),
		repoLimiter:    NewHourlyRateLimiter(repoPerHour, "repo"),
		defaultLimiter: NewHourlyRateLimiter(defaultPerHour, "default"),
	}
}

// PickLimiter 根据 URL 自动选择对应的限制器。
//
// 匹配规则：
//   - 包含 "/search" → searchLimiter
//   - 包含 "/releases" → releaseLimiter
//   - 包含 "/repos" → repoLimiter
//   - 其他 → defaultLimiter
func (qs *APIScopedLimiters) PickLimiter(url string) *HourlyRateLimiter {
	switch {
	case strings.Contains(url, "/search"):
		return qs.searchLimiter
	case strings.Contains(url, "/releases"):
		return qs.releaseLimiter
	case strings.Contains(url, "/repos"):
		return qs.repoLimiter
	default:
		return qs.defaultLimiter
	}
}

// Acquire 根据 URL 自动选择限制器并等待执行权。
func (qs *APIScopedLimiters) Acquire(ctx context.Context, url string) error {
	return qs.PickLimiter(url).Acquire(ctx)
}

// GlobalAPILimiters 全局 API 限制器实例。
// 在应用启动时初始化，供所有请求使用
var GlobalAPILimiters *APIScopedLimiters

// InitGlobalAPILimiters 初始化全局 API 限制器。
// 应在应用启动时调用一次。
//
// 参数:
//   - searchPerHour: 搜索 API 每小时最大请求次数
//   - releasePerHour: Release API 每小时最大请求次数
//   - repoPerHour: 仓库 API 每小时最大请求次数
//   - defaultPerHour: 其他 API 每小时最大请求次数
//
// 安全措施：
// 所有参数 <= 0 时都会使用默认值，防止配置错误导致无限速
func InitGlobalAPILimiters(searchPerHour, releasePerHour, repoPerHour, defaultPerHour int) {
	// 使用默认值防止无效配置
	if searchPerHour <= 0 {
		searchPerHour = 1200
	}
	if releasePerHour <= 0 {
		releasePerHour = 3333
	}
	if repoPerHour <= 0 {
		repoPerHour = 3333
	}
	if defaultPerHour <= 0 {
		defaultPerHour = 3333
	}

	GlobalAPILimiters = NewAPIScopedLimiters(
		searchPerHour,
		releasePerHour,
		repoPerHour,
		defaultPerHour,
	)
}

// CheckAPIQueue 检查是否需要等待 API 限制器。
// 若限制器尚未初始化，返回 nil（不限速）。
//
// 这是外部调用的主要入口函数，用于在发送 API 请求前进行限流检查
func CheckAPIQueue(ctx context.Context, url string) error {
	if GlobalAPILimiters == nil {
		return nil
	}
	return GlobalAPILimiters.Acquire(ctx, url)
}
