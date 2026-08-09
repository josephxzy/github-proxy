package network

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

// TestReconnectingReader_MidStreamDrop 模拟传输中途连接被掐断（非 EOF），
// 验证 ReconnectingReader 自动以 Range 重连并输出完整连续的字节流。
func TestReconnectingReader_MidStreamDrop(t *testing.T) {
	InitHTTPClients()

	data := make([]byte, 300*1024)
	for i := range data {
		data[i] = byte(i % 251)
	}

	var firstRange, secondRange string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rng := r.Header.Get("Range")
		if rng == "" {
			// 首次请求：写出前 64KB 后强制中断连接（模拟 GitHub 掐断空闲连接）
			firstRange = ""
			w.Header().Set("Content-Length", fmt.Sprint(len(data)))
			w.WriteHeader(http.StatusOK)
			w.Write(data[:64*1024])
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		// 重连请求：校验 Range 起始偏移并返回 206 剩余部分
		secondRange = rng
		start, err := strconv.ParseInt(strings.TrimSuffix(rng[6:], "-"), 10, 64)
		if err != nil {
			http.Error(w, "bad range", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(data)-1, len(data)))
		w.Header().Set("Content-Length", fmt.Sprint(len(data)-int(start)))
		w.WriteHeader(http.StatusPartialContent)
		w.Write(data[start:])
	}))
	defer srv.Close()

	// 首次请求（与真实链路一致，走全局客户端）
	resp, err := GetGlobalHTTPClient().Get(srv.URL)
	if err != nil {
		t.Fatalf("initial request failed: %v", err)
	}

	rr := NewReconnectingReader(context.Background(), srv.URL, resp.Request.Header, resp)
	got, err := io.ReadAll(rr)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}

	if firstRange != "" {
		t.Errorf("expected first request without Range, got %q", firstRange)
	}
	if secondRange != "bytes=65536-" {
		t.Errorf("expected reconnect Range bytes=65536-, got %q", secondRange)
	}
	if string(got) != string(data) {
		t.Errorf("output stream mismatch: got %d bytes, want %d", len(got), len(data))
	}
}

// TestReconnectingReader_ServerIgnoresRange 服务器忽略 Range（如 codeload 返回 200 全量），
// 重连必须放弃，输出原始错误，避免字节流重复。
func TestReconnectingReader_ServerIgnoresRange(t *testing.T) {
	InitHTTPClients()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "" {
			w.Header().Set("Content-Length", "1024")
			w.WriteHeader(http.StatusOK)
			w.Write(make([]byte, 512))
			if hj, ok := w.(http.Hijacker); ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
			}
			return
		}
		// 忽略 Range，返回 200 全量
		w.Header().Set("Content-Length", "1024")
		w.WriteHeader(http.StatusOK)
		w.Write(make([]byte, 1024))
	}))
	defer srv.Close()

	resp, err := GetGlobalHTTPClient().Get(srv.URL)
	if err != nil {
		t.Fatalf("initial request failed: %v", err)
	}

	rr := NewReconnectingReader(context.Background(), srv.URL, resp.Request.Header, resp)
	if _, err := io.ReadAll(rr); err == nil {
		t.Fatal("expected error when server ignores Range, got nil")
	}
}
