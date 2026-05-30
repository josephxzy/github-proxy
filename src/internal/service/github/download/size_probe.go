package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github-proxy/config"
	"github-proxy/pkg/network"
)

// PrefetchContentLength 文件总大小探测的调度器。
// 实现三级探测策略，按优先级依次尝试，任一成功即返回：
//
//	→ RangeProbeSize()     ── HTTP Range 协议探测（适用于 Release Asset / Raw）
//	→ QueryArchiveSizeFromAPI() ── GitHub API 查询（适用于 Archive/codeload）
//	→ 返回 0                ── 全部失败（降级到 chunked 传输）
func PrefetchContentLength(ctx context.Context, url string, headers http.Header) int64 {
	if size := RangeProbeSize(ctx, url, headers); size > 0 {
		return size
	}
	if IsArchiveURL(url) {
		return QueryArchiveSizeFromAPI(ctx, url)
	}
	return 0
}

// RangeProbeSize 使用 HTTP Range 协议探测文件大小。
// 发送 Range: bytes=0-0 请求，期望服务器返回 206 Partial Content，
// 并在 Content-Range 响应头中包含文件总大小。
func RangeProbeSize(ctx context.Context, url string, headers http.Header) int64 {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0
	}
	req.Header.Set("Range", "bytes=0-0")
	for key, values := range headers {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Del("Host")
	ApplyGitHubToken(req, url)

	resp, err := network.GetGlobalHTTPClient().Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusPartialContent:
		if cr := resp.Header.Get("Content-Range"); cr != "" {
			if parts := strings.Split(cr, "/"); len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil && total > 0 {
					return total
				}
			}
		}
	case http.StatusOK:
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if size, err := strconv.ParseInt(cl, 10, 64); err == nil && size > 0 {
				return size
			}
		}
	}
	return 0
}

// IsArchiveURL 判断目标 URL 是否为 GitHub Archive 下载链接。
func IsArchiveURL(url string) bool {
	return strings.Contains(url, "/archive/") || strings.Contains(url, "codeload.github.com")
}

// QueryArchiveSizeFromAPI 通过 GitHub Release API 查询 Archive 的文件大小。
func QueryArchiveSizeFromAPI(ctx context.Context, rawURL string) int64 {
	matches := MatchURL(rawURL)
	if len(matches) < 2 {
		return 0
	}
	owner, repo := matches[0], matches[1]

	tag := ExtractTagFromArchiveURL(rawURL)
	if tag == "" {
		return 0
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/tags/%s", owner, repo, tag)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return 0
	}
	cfg := config.GetConfig()
	if cfg.Server.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+cfg.Server.GitHubToken)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := network.GetGlobalHTTPClient().Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0
	}

	type releaseAsset struct {
		Size int64 `json:"size"`
	}
	type releaseResponse struct {
		Assets []releaseAsset `json:"assets"`
	}

	var r releaseResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return 0
	}

	var total int64
	for _, asset := range r.Assets {
		total += asset.Size
	}
	return total
}

// ExtractTagFromArchiveURL 从 GitHub Archive 下载 URL 中提取 tag 或 branch 名称。
func ExtractTagFromArchiveURL(url string) string {
	re := regexp.MustCompile(`/(?:tags|heads)/([^/.]+)`)
	if matches := re.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	re = regexp.MustCompile(`/tar\.zip/([^/.]+)`)
	if matches := re.FindStringSubmatch(url); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
