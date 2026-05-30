package utils

import (
	"fmt"
	"time"
)

// FormatDuration 将时间间隔格式化为人类可读的字符串。
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d秒", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d分钟%d秒", int(d.Minutes()), int(d.Seconds())%60)
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d小时%d分钟", int(d.Hours()), int(d.Minutes())%60)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%d天%d小时", days, hours)
	}
}

// GetUptimeInfo 获取服务运行时间信息
func GetUptimeInfo(startTime time.Time) (time.Duration, float64, string) {
	uptime := time.Since(startTime)
	return uptime, uptime.Seconds(), FormatDuration(uptime)
}
