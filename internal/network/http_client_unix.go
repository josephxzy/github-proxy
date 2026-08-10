//go:build !windows

package network

import (
	"syscall"
)

// setSocketBuffer 设置 TCP 连接的接收缓冲区大小为 4MB。
// 作为 DialContext 的 Control 回调使用：大文件下载场景下，
// 增大内核接收缓冲区可减少 TCP 拥塞窗口限制造成的吞吐瓶颈。
func setSocketBuffer(network, address string, c syscall.RawConn) error {
	return c.Control(func(fd uintptr) {
		syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
	})
}
