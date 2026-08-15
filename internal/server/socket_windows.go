//go:build windows

package server

import "net"

// clientSocketSendBuffer 客户端（下游）TCP 发送缓冲区大小（Windows）。
//
// Windows 上被 accept 的连接默认 SO_SNDBUF 仅 64KB（实测）。大文件下载经
// 高延迟链路（如跨境 RTT 200ms+）时，吞吐被发送窗口卡死：
//
//	64KB / 200ms ≈ 300KB/s   ← "不限速仍卡 300KB" 的根因
//
// 放大到 4MB 后吞吐上限提升到 4MB/RTT（200ms 时约 20MB/s），与上游连接
// （http_client 中 SO_RCVBUF=4MB）对齐。Windows 对 SO_SNDBUF 的设定
// 原样生效（实测 getsockopt 返回 4194304），无钳制。
const clientSocketSendBuffer = 4 * 1024 * 1024

// tuneClientSocket 在连接建立时调整下游 TCP 参数（Windows 实现）：
//   - SO_SNDBUF：放大到 4MB，消除高延迟路径的发送窗口瓶颈
//   - TCP_NODELAY：禁用 Nagle，避免小包与延迟 ACK 交互造成额外停顿
//
// net/http 服务端默认对 accept 的连接两者都不设置，必须在此显式调优。
func tuneClientSocket(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	tc.SetWriteBuffer(clientSocketSendBuffer)
	tc.SetNoDelay(true)
}
