package handlers

import (
	"sync"
)

// WaterlineBuffer 带水位线的环形缓冲区。
// 生产者（从 GitHub 读取）→ 写入缓冲区 → 消费者（写入客户端）
// 水位达到 80% 时暂停生产，降到 20% 时恢复。
//
// 关键不变式：Write 必须完整写入全部数据，绝不短写、绝不丢数据。
// 读/写指针按 capacity 取模环绕（真环形），无需等待"完全读空"才复位，
// 因此在持续反压（消费者长期慢于生产者）下也不会发生数据丢失。
type WaterlineBuffer struct {
	mu       sync.Mutex
	cond     *sync.Cond
	data     []byte
	writePos int // 下一个写入位置（对 capacity 取模环绕）
	readPos  int // 下一个读取位置（对 capacity 取模环绕）
	count    int // 当前缓冲的字节数
	capacity int
	paused   bool
	closed   bool
}

func newWaterlineBuffer(capacity int) *WaterlineBuffer {
	b := &WaterlineBuffer{
		data:     make([]byte, capacity),
		capacity: capacity,
	}
	b.cond = sync.NewCond(&b.mu)
	return b
}

func (b *WaterlineBuffer) usage() float64 {
	return float64(b.count) / float64(b.capacity)
}

// Write 完整写入 data，返回 len(data)。
// 缓冲区满时阻塞等待消费者腾出空间，保证不短写、不丢数据。
func (b *WaterlineBuffer) Write(data []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	total := 0
	for total < len(data) {
		for b.count == b.capacity && !b.closed {
			b.cond.Wait()
		}
		if b.closed {
			// 生产者自身关闭缓冲区后不应再写入（防御性分支，正常不会到达）
			return total
		}
		n := len(data) - total
		if space := b.capacity - b.count; n > space {
			n = space
		}
		if tail := b.capacity - b.writePos; n > tail {
			n = tail
		}
		copy(b.data[b.writePos:b.writePos+n], data[total:total+n])
		b.writePos = (b.writePos + n) % b.capacity
		b.count += n
		total += n
		b.cond.Broadcast()
	}
	if b.usage() >= 0.8 && !b.paused {
		b.paused = true
	}
	return total
}

// Read 读取最多 len(p) 字节；无数据时阻塞，直到有数据或缓冲区关闭。
// 缓冲区关闭且读空后返回 0（io.EOF 语义）。
func (b *WaterlineBuffer) Read(p []byte) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.count == 0 && !b.closed {
		b.cond.Wait()
	}
	if b.count == 0 {
		return 0
	}
	n := len(p)
	if n > b.count {
		n = b.count
	}
	if tail := b.capacity - b.readPos; n > tail {
		n = tail
	}
	copy(p, b.data[b.readPos:b.readPos+n])
	b.readPos = (b.readPos + n) % b.capacity
	b.count -= n
	if b.usage() < 0.2 && b.paused {
		b.paused = false
	}
	b.cond.Broadcast() // 唤醒可能阻塞在"缓冲区满"的生产者
	return n
}

// Close 关闭缓冲区，唤醒所有等待的 reader/writer。
func (b *WaterlineBuffer) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closed = true
	b.cond.Broadcast()
}

func (b *WaterlineBuffer) WaitUnpaused() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for b.paused && !b.closed {
		b.cond.Wait()
	}
}
