//go:build windows

package server

import (
	"net"
	"syscall"
	"testing"
)

// getSndBuf 读取连接当前的 SO_SNDBUF 实际值。
func getSndBuf(t *testing.T, tc *net.TCPConn) int {
	t.Helper()
	raw, err := tc.SyscallConn()
	if err != nil {
		t.Fatalf("SyscallConn 失败: %v", err)
	}
	var v int
	raw.Control(func(fd uintptr) {
		v, err = syscall.GetsockoptInt(syscall.Handle(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF)
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
		v, err = syscall.GetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY)
	})
	if err != nil {
		t.Fatalf("GetsockoptInt(TCP_NODELAY) 失败: %v", err)
	}
	return v
}
