package network

import (
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/net"
)

// NetworkSpeed 实时网速信息
type NetworkSpeed struct {
	InterfaceName string  `json:"interfaceName"`
	UploadSpeed   float64 `json:"uploadSpeed"`   // bytes/sec
	DownloadSpeed float64 `json:"downloadSpeed"` // bytes/sec
	TotalSent     uint64  `json:"totalSent"`
	TotalRecv     uint64  `json:"totalRecv"`
	Timestamp     int64   `json:"timestamp"`
}

// Monitor 网络监控器
type Monitor struct {
	mu           sync.Mutex
	lastCounters map[string]*net.IOCountersStat
	currentSpeed *NetworkSpeed
	interval     time.Duration
	stopCh       chan struct{}
}

// NewMonitor 创建网络监控器
func NewMonitor(sampleInterval time.Duration) *Monitor {
	if sampleInterval == 0 {
		sampleInterval = time.Second
	}
	return &Monitor{
		lastCounters: make(map[string]*net.IOCountersStat),
		interval:     sampleInterval,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动监控
func (m *Monitor) Start() {
	go m.loop()
}

// Stop 停止监控
func (m *Monitor) Stop() {
	close(m.stopCh)
}

// GetSpeed 获取当前网速
func (m *Monitor) GetSpeed() *NetworkSpeed {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.currentSpeed
}

func (m *Monitor) loop() {
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.sample()
		case <-m.stopCh:
			return
		}
	}
}

func (m *Monitor) sample() {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return
	}

	now := time.Now().Unix()

	var bestIface string
	var maxTraffic uint64

	for _, curr := range counters {
		if skipInterface(curr.Name) {
			continue
		}

		prev, exists := m.lastCounters[curr.Name]
		if !exists {
			cp := curr
			m.lastCounters[curr.Name] = &cp
			continue
		}

		sentDelta := curr.BytesSent - prev.BytesSent
		recvDelta := curr.BytesRecv - prev.BytesRecv
		total := sentDelta + recvDelta

		if total > maxTraffic {
			maxTraffic = total
			bestIface = curr.Name
		}

		m.lastCounters[curr.Name] = &curr
	}

	if bestIface == "" {
		return
	}

	prev := m.lastCounters[bestIface]
	curr := findCounter(counters, bestIface)
	if prev == nil || curr == nil {
		return
	}

	upload := float64(curr.BytesSent-prev.BytesSent) / m.interval.Seconds()
	download := float64(curr.BytesRecv-prev.BytesRecv) / m.interval.Seconds()

	speed := &NetworkSpeed{
		InterfaceName: bestIface,
		UploadSpeed:   upload,
		DownloadSpeed: download,
		TotalSent:     curr.BytesSent,
		TotalRecv:     curr.BytesRecv,
		Timestamp:     now,
	}

	m.mu.Lock()
	m.currentSpeed = speed
	m.mu.Unlock()
}

func findCounter(counters []net.IOCountersStat, name string) *net.IOCountersStat {
	for i := range counters {
		if counters[i].Name == name {
			return &counters[i]
		}
	}
	return nil
}

func skipInterface(name string) bool {
	switch name {
	case "lo", "loopback", "vmnet", "veth", "docker", "br-", "virbr":
		return true
	default:
		return len(name) > 15
	}
}
