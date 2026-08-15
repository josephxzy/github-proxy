//go:build !windows

package server

import (
	"net"
	"syscall"
	"testing"
)

// minSendBuffer Unix 下限：不设置 SO_SNDBUF，保持内核默认（Linux
// wmem_default 208KB，getsockopt 报告 425984）。64KB 仅作"未被意外
// 调小"的宽松断言。
func minSendBuffer() int { return 64 * 1024 }

// getSndBuf 读取连接当前的 SO_SNDBUF 实际值。
func getSndBuf(t *testing.T, tc *net.TCPConn) int {
	t.Helper()
	raw, err := tc.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn 失败: %v", err)
	}
	var v int
	raw.Control(func(fd uintptr) {
		v, err = syscall.GetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF)
	})
	if err != nil {
		t.Fatalf("GetsockoptInt(SO_SNDBUF) 失败: %v", err)
	}
	return v
}

// getNoDelay 读取连接当前的 TCP_NODELAY 值（1=已禁用 Nagle）。
func getNoDelay(t *testing.T, tc *net.TCPConn) int {
	t.Helper()
	raw, err := tc.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn 失败: %v", err)
	}
	var v int
	raw.Control(func(fd uintptr) {
		v, err = syscall.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY)
	})
	if err != nil {
		t.Fatalf("GetsockoptInt(TCP_NODELAY) 失败: %v", err)
	}
	return v
}
