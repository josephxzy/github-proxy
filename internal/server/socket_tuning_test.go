package server

import (
	"net"
	"testing"
)

// TestTuneClientSocket 验证下游连接调优确实生效：
// 1) TCP_NODELAY 已启用（Nagle 关闭）——所有平台
// 2) SO_SNDBUF 不低于平台下限——Windows 上被放大到 4MB（修复前默认
//    64KB，高延迟链路下吞吐被卡在 64KB/RTT ≈ 300KB/s）；Linux 上保持
//    内核默认/自动调优（手动设置会被 wmem_max 钳制且关闭自动调优，故不设）
func TestTuneClientSocket(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("监听失败: %v", err)
	}
	defer ln.Close()

	type result struct {
		sndBuf int
		delay  int
	}
	done := make(chan result, 1)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			done <- result{}
			return
		}
		defer conn.Close()
		tc := conn.(*net.TCPConn)
		tuneClientSocket(tc)
		done <- result{sndBuf: getSndBuf(t, tc), delay: getNoDelay(t, tc)}
	}()

	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	client.Close()

	r := <-done
	t.Logf("tuned socket: SO_SNDBUF=%d, TCP_NODELAY=%d", r.sndBuf, r.delay)
	if r.sndBuf < minSendBuffer() {
		t.Errorf("SO_SNDBUF = %d, 期望 >= %d（平台下限）", r.sndBuf, minSendBuffer())
	}
	if r.delay != 1 {
		t.Errorf("TCP_NODELAY = %d, 期望 1（已禁用 Nagle）", r.delay)
	}
}
