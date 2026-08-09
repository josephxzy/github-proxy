package handlers

import (
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github-proxy/config"
	"github-proxy/internal/ratelimit"
	"github-proxy/internal/waterline"

	"github.com/gin-gonic/gin"
)

const (
	flushThreshold = 64 * 1024 // 每 64KB 刷新一次
)

// streamToClientWithWaterline 带水位线反压的流式传输。
// 生产端从 GitHub 全速读取，受水位线暂停控制。
// 消费端通过 ratelimit.Writer 限速写入客户端。
// upstream 为上游响应体（io.Closer）：客户端停滞超过 downloadIdleTimeout
// 时主动关闭它释放 GitHub 连接资源，浏览器连接保留，恢复后由
// ReconnectingReader 自动重连续传。
func streamToClientWithWaterline(c *gin.Context, body io.Reader, rlw *ratelimit.Writer, upstream io.Closer) int64 {
	cfg := config.GetConfig()
	bufCap := cfg.Server.BufferSize
	if bufCap <= 0 {
		bufCap = 8 * 1024 * 1024
	}
	wb := waterline.NewWaterlineBuffer(int(bufCap))

	var written int64
	producerDone := make(chan struct{})
	stopped := make(chan struct{})
	defer close(stopped)
	ctx := c.Request.Context()

	// 客户端断开监听：一旦 ctx 取消，立刻关闭缓冲区，
	// 把阻塞在 Read 上的消费者和阻塞在 WaitUnpaused/Read 上的生产者同时唤醒。
	go func() {
		<-ctx.Done()
		wb.Close()
	}()

	// 客户端停滞监听：超过 downloadIdleTimeout 无写入进展（浏览器暂停且未恢复），
	// 主动关闭上游 GitHub 连接释放资源。浏览器连接不受影响，
	// 用户恢复时生产者 Read 失败 → ReconnectingReader 自动以 Range 重连续传。
	var lastWrite atomic.Int64
	lastWrite.Store(time.Now().UnixNano())
	if idleTimeout := time.Duration(cfg.Server.DownloadIdleTimeout) * time.Second; idleTimeout > 0 && upstream != nil {
		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-stopped:
					return
				case <-ctx.Done():
					return
				case <-ticker.C:
					if time.Since(time.Unix(0, lastWrite.Load())) > idleTimeout {
						upstream.Close()
						return
					}
				}
			}
		}()
	}

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
			lastWrite.Store(time.Now().UnixNano())
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
					lastWrite.Store(time.Now().UnixNano())
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
