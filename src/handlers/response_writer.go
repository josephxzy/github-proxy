package handlers

import (
	"io"
	"net/http"

	"github-proxy/config"

	"github.com/gin-gonic/gin"
)

const (
	flushThreshold = 64 * 1024 // 每 64KB 刷新一次
)

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
	ctx := c.Request.Context()

	// 客户端断开监听：一旦 ctx 取消，立刻关闭缓冲区，
	// 把阻塞在 Read 上的消费者和阻塞在 WaitUnpaused/Read 上的生产者同时唤醒。
	go func() {
		<-ctx.Done()
		wb.Close()
	}()

	go func() {
		defer close(producerDone)
		defer wb.Close()
		chunk := make([]byte, 32*1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			wb.WaitUnpaused()
			if wb.IsClosed() {
				return
			}
			n, err := body.Read(chunk)
			if n > 0 {
				wb.Write(chunk[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	out := make([]byte, 32*1024)
	defer wb.Close()
	for {
		n, rerr := wb.Read(out)
		if n > 0 {
			wn, err := rlw.Write(out[:n])
			if err != nil {
				written += int64(wn)
				return written
			}
			written += int64(wn)
		}
		select {
		case <-producerDone:
			for {
				n, rerr := wb.Read(out)
				if n > 0 {
					wn, err := rlw.Write(out[:n])
					if err != nil {
						written += int64(wn)
						return written
					}
					written += int64(wn)
				}
				if rerr == io.EOF {
					break
				}
			}
			rlw.Flush()
			return written
		case <-ctx.Done():
			return written
		default:
		}
		if rerr == io.EOF {
			rlw.Flush()
			return written
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
