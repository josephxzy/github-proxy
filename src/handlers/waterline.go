package handlers

import (
	"sync"
)

// WaterlineBuffer 带水位线的环形缓冲区。
// 生产者（从 GitHub 读取）→ 写入缓冲区 → 消费者（写入客户端）
// 水位达到 80% 时暂停生产，降到 20% 时恢复。
type WaterlineBuffer struct {
	mu       sync.Mutex
	cond     *sync.Cond
	data     []byte
	writePos int
	readPos  int
	capacity int
	paused   bool
}

// newWaterlineBuffer 创建水位线缓冲区。
func newWaterlineBuffer(capacity int) *WaterlineBuffer {
	b := &WaterlineBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *WaterlineBuffer) usage() float64 {
	return float64(b.writePos-b.readPos) / float64(b.capacity)
}

func (b *WaterlineBuffer) Write(data []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := copy(b.data[b.writePos:], data)
	b.writePos += n
	b.cond.Signal()
	if b.usage() >= 0.8 && !b.paused {
		b.paused = true
	}
	return n
}

func (b *WaterlineBuffer) Read(p []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.writePos <= b.readPos {
		b.cond.Wait()
	}
	n := copy(p, b.data[b.readPos:b.writePos])
	b.readPos += n
	if b.usage() < 0.2 && b.paused {
		b.paused = false
		b.cond.Broadcast()
	}
	b.compact()
	return n
}

func (b *WaterlineBuffer) WaitUnpaused() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.paused {
		b.cond.Wait()
	}
}

func (b *WaterlineBuffer) compact() {
	if b.readPos > 0 && b.readPos == b.writePos {
		b.readPos = 0
		b.writePos = 0
	}
}
