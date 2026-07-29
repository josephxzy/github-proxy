package handlers

import (
	"bufio"
	"io"
	"net/http"

	"github-proxy/config"

	"github.com/gin-gonic/gin"
)

const (
	streamBufferSize = 128 * 1024 // 流式传输缓冲区: 128KB
	flushThreshold   = 64 * 1024  // 每 64KB 刷新一次
)

func streamToClient(c *gin.Context, body io.Reader, w io.Writer) int64 {
	buf := bufio.NewReaderSize(body, streamBufferSize)
	written, _ := io.Copy(w, buf)
	if fw, ok := w.(*flushingWriter); ok {
		fw.Flush()
	}
	return written
}

// streamToClientWithWaterline 带水位线反压的流式传输。
// 生产端从 GitHub 全速读取，受水位线暂停控制。
// 消费端通过 rateLimitedWriter 限速写入客户端。
func streamToClientWithWaterline(c *gin.Context, body io.Reader, rlw *rateLimitedWriter) int64 {
	cfg := config.GetConfig()
	bufCap := cfg.Server.BufferSize
	if bufCap <= 0 {
		bufCap = 8 * 1024 * 1024
	}
	wb := newWaterlineBuffer(int(bufCap))

	var written int64
	producerDone := make(chan struct{})

	// 生产者：从 GitHub 读取 → 写入水位线缓冲区
	go func() {
		defer close(producerDone)
		chunk := make([]byte, 32*1024)
		for {
			wb.WaitUnpaused()
			n, err := body.Read(chunk)
			if n > 0 {
				wb.Write(chunk[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// 消费者：从缓冲区读取 → 限速写入客户端
	out := make([]byte, 32*1024)
	for {
		n := wb.Read(out)
		if n > 0 {
			wn, _ := rlw.Write(out[:n])
			written += int64(wn)
		}
		select {
		case <-producerDone:
			// 消费完缓冲区残留数据
			for {
				n := wb.Read(out)
				if n == 0 {
					break
				}
				wn, _ := rlw.Write(out[:n])
				written += int64(wn)
			}
			return written
		default:
		}
	}
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
