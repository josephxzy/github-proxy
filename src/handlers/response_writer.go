package handlers

import (
	"bufio"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	streamBufferSize = 128 * 1024 // 流式传输缓冲区: 128KB
	flushThreshold   = 64 * 1024  // 每 64KB 刷新一次
)

// streamToClient 流式传输函数，直接读取并写入客户端。
// 去掉了 io.Pipe 管道层，避免同步管道（内部仅 64KB 缓冲）
// 在被填满后导致的管道阻塞和背压问题。
//
// 工作流程:
//  1. 从上游（GitHub）读取数据到缓冲区
//  2. 写入 Gin ResponseWriter
//  3. 每达到 flushThreshold 字节就 Flush 一次，确保数据及时推送给客户端
//  4. 循环直到上游数据读完
//
// 相比原双 goroutine + io.Pipe 架构:
//   - 减少了一层数据拷贝（io.Pipe → bufio.Writer）
//   - 去掉了 sync.WaitGroup 的同步开销
//   - 去掉了 256KB 双层缓冲，降低内存占用
//   - 客户端慢速读取时会轻微影响上游读取速度，但实际影响极小
func streamToClient(c *gin.Context, body io.Reader) int64 {
	buf := bufio.NewReaderSize(body, streamBufferSize)

	w := &flushingWriter{
		writer:        c.Writer,
		flusher:       c.Writer.(http.Flusher),
		flushInterval: flushThreshold,
	}

	written, _ := io.Copy(w, buf)
	w.Flush()

	return written
}

// flushingWriter 是一个包装的 Writer，在写入一定量数据后自动 Flush。
// 确保数据及时推送给客户端，避免 Gin 内部缓冲导致的延迟。
type flushingWriter struct {
	writer        gin.ResponseWriter
	flusher       http.Flusher
	flushInterval int64
	written       int64
	flushed       int64
}

func (w *flushingWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		w.written += int64(n)
		if w.written-w.flushed >= w.flushInterval {
			w.flusher.Flush()
			w.flushed = w.written
		}
	}
	return n, err
}

func (w *flushingWriter) Flush() {
	w.flusher.Flush()
	w.flushed = w.written
}

type countWriter struct {
	writer  io.Writer
	written *int64
}

func (w *countWriter) Write(p []byte) (int, error) {
	n, err := w.writer.Write(p)
	if n > 0 {
		*w.written += int64(n)
	}
	return n, err
}
