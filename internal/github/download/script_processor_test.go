package download

import (
	"bytes"
	"compress/gzip"
	"io"
	"strings"
	"testing"
)

// TestTransformURL 单个 GitHub URL 改写为代理地址。
func TestTransformURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		host string
		want string
	}{
		{"普通 https URL", "https://github.com/owner/repo/raw/main/a.sh", "proxy.example.com", "https://proxy.example.com/https://github.com/owner/repo/raw/main/a.sh"},
		{"http 升级为 https", "http://github.com/owner/repo/raw/main/a.sh", "proxy.example.com", "https://proxy.example.com/https://github.com/owner/repo/raw/main/a.sh"},
		{"无协议补全", "github.com/owner/repo/raw/main/a.sh", "proxy.example.com", "https://proxy.example.com/https://github.com/owner/repo/raw/main/a.sh"},
		{"协议相对 // 保留", "//github.com/owner/repo", "proxy.example.com", "https://proxy.example.com///github.com/owner/repo"},
		{"已含代理 host 原样返回", "https://proxy.example.com/https://github.com/owner/repo", "proxy.example.com", "https://proxy.example.com/https://github.com/owner/repo"},
		{"host 无 scheme 自动补全", "https://github.com/owner/repo", "proxy.example.com:5000", "https://proxy.example.com:5000/https://github.com/owner/repo"},
		{"host 尾部斜杠去除", "https://github.com/owner/repo", "https://proxy.example.com/", "https://proxy.example.com/https://github.com/owner/repo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := transformURL(tt.url, tt.host); got != tt.want {
				t.Errorf("transformURL(%q, %q) = %q, want %q", tt.url, tt.host, got, tt.want)
			}
		})
	}
}

// TestProcessGitHubURLs 内容中的 GitHub URL 整体替换，且保留前缀字符。
func TestProcessGitHubURLs(t *testing.T) {
	input := `curl -fsSL https://github.com/owner/repo/raw/main/install.sh | bash
export URL="http://raw.githubusercontent.com/owner/repo/main/x.sh"`
	host := "proxy.example.com"

	got := processGitHubURLs(input, host)

	if !strings.Contains(got, "https://proxy.example.com/https://github.com/owner/repo/raw/main/install.sh") {
		t.Errorf("未替换 github.com URL:\n%s", got)
	}
	if !strings.Contains(got, "https://proxy.example.com/https://raw.githubusercontent.com/owner/repo/main/x.sh") {
		t.Errorf("未替换 raw.githubusercontent.com URL:\n%s", got)
	}
	// 前缀字符（空格、引号）应保留
	if !strings.Contains(got, "https://proxy.example.com/https://") || !strings.Contains(got, `"https://proxy.example.com/`) {
		t.Errorf("前缀字符丢失:\n%s", got)
	}
}

// TestProcessSmartNoGithubURLs 内容不含 GitHub 域名时应原样返回（不做字符串处理）。
func TestProcessSmartNoGithubURLs(t *testing.T) {
	input := "#!/bin/sh\necho hello world\n"
	reader, size, err := ProcessSmart(strings.NewReader(input), false, "proxy.example.com")
	if err != nil {
		t.Fatalf("ProcessSmart 出错: %v", err)
	}
	data, _ := io.ReadAll(reader)
	if string(data) != input {
		t.Errorf("内容被意外修改:\n%s", data)
	}
	if size != int64(len(input)) {
		t.Errorf("size = %d, want %d", size, len(input))
	}
}

// TestProcessSmartReplaces 含 GitHub URL 的脚本被正确替换。
func TestProcessSmartReplaces(t *testing.T) {
	input := "curl -L https://github.com/owner/repo/archive/refs/heads/main.zip -o x.zip"
	reader, size, err := ProcessSmart(strings.NewReader(input), false, "proxy.example.com")
	if err != nil {
		t.Fatalf("ProcessSmart 出错: %v", err)
	}
	data, _ := io.ReadAll(reader)
	if !strings.Contains(string(data), "https://proxy.example.com/") {
		t.Errorf("URL 未替换:\n%s", data)
	}
	if size != int64(len(data)) {
		t.Errorf("size = %d, 内容长度 = %d", size, len(data))
	}
}

// TestProcessSmartGzip gzip 压缩的脚本输入应被解压、替换并返回解压后内容。
func TestProcessSmartGzip(t *testing.T) {
	raw := "curl -L https://github.com/owner/repo/raw/main/a.sh"
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write([]byte(raw)); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	reader, size, err := ProcessSmart(bytes.NewReader(buf.Bytes()), true, "proxy.example.com")
	if err != nil {
		t.Fatalf("ProcessSmart(gzip) 出错: %v", err)
	}
	data, _ := io.ReadAll(reader)
	if !strings.Contains(string(data), "https://proxy.example.com/https://github.com/owner/repo/raw/main/a.sh") {
		t.Errorf("gzip 内容未正确解压替换:\n%s", data)
	}
	if size != int64(len(data)) {
		t.Errorf("size = %d, 内容长度 = %d", size, len(data))
	}
}

// TestProcessSmartEmpty 空输入返回空输出。
func TestProcessSmartEmpty(t *testing.T) {
	reader, size, err := ProcessSmart(strings.NewReader(""), false, "proxy.example.com")
	if err != nil {
		t.Fatalf("ProcessSmart 出错: %v", err)
	}
	data, _ := io.ReadAll(reader)
	if len(data) != 0 || size != 0 {
		t.Errorf("空输入应返回空, got len=%d size=%d", len(data), size)
	}
}

// TestProcessSmartTooLarge 超过 MaxShellSize 的脚本应拒绝处理。
func TestProcessSmartTooLarge(t *testing.T) {
	big := strings.Repeat("x", MaxShellSize+1)
	_, _, err := ProcessSmart(strings.NewReader(big), false, "proxy.example.com")
	if err == nil {
		t.Error("超限脚本应返回错误")
	}
}

// TestProcessSmartTooLargeGzip 压缩后超限同样拒绝。
func TestProcessSmartTooLargeGzip(t *testing.T) {
	// 内容本身超过 MaxShellSize（每段 29 字节，段数略多于上限/29）
	big := strings.Repeat("https://github.com/owner/repo ", MaxShellSize/29+1)
	if int64(len(big)) <= MaxShellSize {
		t.Fatalf("测试数据未超限: len=%d", len(big))
	}
	_, _, err := ProcessSmart(strings.NewReader(big), false, "proxy.example.com")
	if err == nil {
		t.Error("超限脚本应返回错误")
	}
}
