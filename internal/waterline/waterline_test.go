package waterline

import (
	"bytes"
	"crypto/rand"
	"io"
	"testing"
	"time"
)

// TestWaterlineBufferNoDataLossUnderBackpressure 复现修复前的 bug 场景：
// 生产者持续快于消费者（反压），缓冲区在高低水位间震荡且多次绕环。
// 旧实现（线性缓冲区 + 忽略短写）会在此场景下静默丢数据，
// 导致客户端收到的字节数少于 Content-Length（浏览器报"网络错误"）。
func TestWaterlineBufferNoDataLossUnderBackpressure(t *testing.T) {
	const (
		capacity  = 64 * 1024       // 小容量，加速绕环
		totalSize = 8 * 1024 * 1024 // 总数据量 = 128 倍容量，确保多次绕环
		chunkSize = 32 * 1024       // 与生产环境一致
	)

	// 生成确定性"随机"数据
	src := make([]byte, totalSize)
	if _, err := rand.Read(src); err != nil {
		t.Fatal(err)
	}

	wb := NewWaterlineBuffer(capacity)

	// 生产者：全速写入（模拟从 GitHub 读取）
	go func() {
		defer wb.Close()
		for off := 0; off < totalSize; off += chunkSize {
			wb.WaitUnpaused()
			end := off + chunkSize
			if end > totalSize {
				end = totalSize
			}
			n := wb.Write(src[off:end])
			if n != end-off {
				t.Errorf("Write 短写: 写入 %d, 期望 %d", n, end-off)
				return
			}
		}
	}()

	// 消费者：以小粒度限速读取（模拟慢速客户端），校验字节序列。
	// 关键点：消费端持续稳定消费但永不在中途把缓冲区读空，
	// 使缓冲区在 20%~80% 水位间震荡、读写指针持续右移绕环——
	// 这正是旧实现静默丢数据的场景。
	dst := make([]byte, 0, totalSize)
	out := make([]byte, 7*1024)
	for {
		n, err := wb.Read(out)
		if err == io.EOF {
			break
		}
		dst = append(dst, out[:n]...)
		// 模拟消费端限速，制造持续反压
		time.Sleep(20 * time.Microsecond)
	}

	if len(dst) != totalSize {
		t.Fatalf("数据丢失: 收到 %d 字节, 期望 %d 字节", len(dst), totalSize)
	}
	if !bytes.Equal(dst, src) {
		// 找到第一个不一致的位置，便于定位
		for i := range src {
			if dst[i] != src[i] {
				t.Fatalf("数据损坏: 偏移 %d 处不一致", i)
			}
		}
	}
}

// TestWaterlineBufferBlockingWrite 验证缓冲区满时 Write 阻塞而非短写。
func TestWaterlineBufferBlockingWrite(t *testing.T) {
	wb := NewWaterlineBuffer(100)

	// 写满缓冲区（Write 的调用方保证单次写入不超过容量）
	fill := make([]byte, 100)
	if n := wb.Write(fill); n != 100 {
		t.Fatalf("首次写入应为 100, 实际 %d", n)
	}

	// 再次写入应当阻塞（缓冲区已满），直到有消费者读取
	done := make(chan int, 1)
	go func() {
		done <- wb.Write([]byte{1, 2, 3})
	}()

	select {
	case <-done:
		t.Fatal("缓冲区满时 Write 未阻塞")
	case <-time.After(50 * time.Millisecond):
	}

	// 消费者读出 10 字节，生产者应完成写入
	p := make([]byte, 10)
	if n, _ := wb.Read(p); n != 10 {
		t.Fatalf("Read 应返回 10, 实际 %d", n)
	}
	select {
	case n := <-done:
		if n != 3 {
			t.Fatalf("Write 应完整写入 3 字节, 实际 %d", n)
		}
	case <-time.After(time.Second):
		t.Fatal("消费者读取后 Write 仍未完成")
	}
}

// TestWaterlineBufferCloseUnblocksReader 验证 Close 后读空返回 io.EOF。
func TestWaterlineBufferCloseUnblocksReader(t *testing.T) {
	wb := NewWaterlineBuffer(1024)
	wb.Write([]byte("hello"))
	wb.Close()

	got, err := io.ReadAll(wb)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hello" {
		t.Fatalf("Close 前应读出全部已缓冲数据, 实际 %q", got)
	}
	if n, err := wb.Read(make([]byte, 1)); n != 0 || err != io.EOF {
		t.Fatalf("Close 且读空后 Read 应返回 (0, io.EOF), 实际 (%d, %v)", n, err)
	}
}

// TestWaterlineBufferCloseUnblocksBlockedReader 验证消费者阻塞在 Read 时，
// 外部调用 Close 能立刻将其唤醒（模拟客户端断开）。
func TestWaterlineBufferCloseUnblocksBlockedReader(t *testing.T) {
	wb := NewWaterlineBuffer(1024)

	// 消费者：阻塞在 Read 上（缓冲区为空）
	done := make(chan int, 1)
	go func() {
		n, _ := wb.Read(make([]byte, 1024))
		done <- n
	}()

	// 确保消费者已经进入 Read
	time.Sleep(10 * time.Millisecond)

	// 模拟客户端断开：外部调用 Close 唤醒消费者
	wb.Close()

	select {
	case n := <-done:
		if n != 0 {
			t.Fatalf("Close 后 Read 应返回 0, 实际 %d", n)
		}
	case <-time.After(time.Second):
		t.Fatal("Close 未能在 1s 内唤醒阻塞的 Read")
	}
}

// TestWaterlineBufferCloseUnblocksBlockedWriter 验证生产者阻塞在 Write（缓冲区满）时，
// 外部调用 Close 能立刻将其唤醒。
func TestWaterlineBufferCloseUnblocksBlockedWriter(t *testing.T) {
	wb := NewWaterlineBuffer(100)

	// 先写满缓冲区
	if n := wb.Write(make([]byte, 100)); n != 100 {
		t.Fatalf("首次写入应为 100, 实际 %d", n)
	}

	// 生产者：再次写入会阻塞（缓冲区满）
	done := make(chan int, 1)
	go func() {
		done <- wb.Write(make([]byte, 10))
	}()

	// 确保生产者已经阻塞
	time.Sleep(10 * time.Millisecond)

	// 外部调用 Close 唤醒阻塞的 Write
	wb.Close()

	select {
	case n := <-done:
		if n == 10 {
			t.Fatal("Close 后满缓冲区的 Write 不应完整写入")
		}
	case <-time.After(time.Second):
		t.Fatal("Close 未能在 1s 内唤醒阻塞的 Write")
	}
}
