//go:build !windows

package server

import "net"

// tuneClientSocket 在连接建立时调整下游 TCP 参数（Linux/Unix 实现）。
//
// 只禁用 Nagle（TCP_NODELAY），不设置 SO_SNDBUF：
//
//   - Linux 默认有 tcp_wmem 发送缓冲自动调优（上限 net.ipv4.tcp_wmem[2]，
//     默认 4MB），高 BDP 路径下会自动增长，无需干预；
//   - 手动 SetWriteBuffer(4MB) 反而有害：非特权进程的 SO_SNDBUF 被
//     net.core.wmem_max（默认 212992 = 208KB）钳制，且设定后关闭自动调优。
//     CI 实测（ubuntu-latest）：SetWriteBuffer(4MB) 后 getsockopt 仅返回
//     425984（= 2 × 208KB，内核翻倍计入），高延迟吞吐反而被锁死在
//     208KB/RTT，远低于自动调优可达的 4MB/RTT。
//
// 如需在 Linux 上进一步扩大发送缓冲，应由部署侧调高内核参数：
//
//	sysctl -w net.core.wmem_max=4194304
func tuneClientSocket(conn net.Conn) {
	tc, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	tc.SetNoDelay(true)
}
