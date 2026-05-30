package github

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github-proxy/config"
	"github-proxy/pkg/network"
)

// DownloadService GitHub 文件下载服务。
// **专门处理文件下载类请求**：
//   - releases/archive/blob/raw/gist 文件下载
//   - Range 预检（获取总大小用于进度条）
//   - 断点续传支持
//   - 脚本文件 URL 替换 (.sh/.ps1)
//   - 安全检查（Content-Type/文件大小）
type DownloadService struct {
	config *config.AppConfig
}

// NewDownloadService 创建文件下载服务实例。
func NewDownloadService(cfg *config.AppConfig) *DownloadService {
	return &DownloadService{
		config: cfg,
	}
}

// DownloadRequest 下载请求参数。
type DownloadRequest struct {
	Context       context.Context
	Method        string
	URL           string
	Headers       http.Header
	Body          io.Reader
	IsRange       bool
	Authenticated bool
}

// DownloadResult 下载结果。
type DownloadResult struct {
	StatusCode int
	Headers    http.Header
	Body       io.Reader
	Error      error
	Latency    int
}

// Execute 执行文件下载的核心流程。
func (s *DownloadService) Execute(req *DownloadRequest) *DownloadResult {
	result := &DownloadResult{}
	startTime := time.Now()

	httpResp, err := s.doGitHubRequest(req)
	if err != nil {
		if req.Context.Err() != nil {
			result.Error = req.Context.Err()
			return result
		}
		latency := int(time.Since(startTime).Milliseconds())
		result.Error = fmt.Errorf("server error %v (latency=%dms)", err, latency)
		result.StatusCode = http.StatusInternalServerError
		result.Latency = latency
		return result
	}
	defer httpResp.Body.Close()

	result.Latency = int(time.Since(startTime).Milliseconds())

	if !s.validateResponse(httpResp, req, result) {
		return result
	}

	network.CleanSecurityHeaders(httpResp.Header)

	result.StatusCode = httpResp.StatusCode
	result.Headers = httpResp.Header
	result.Body = httpResp.Body

	return result
}

// doGitHubRequest 执行 GitHub 下载请求。
func (s *DownloadService) doGitHubRequest(req *DownloadRequest) (*http.Response, error) {
	httpReq, err := http.NewRequestWithContext(req.Context, req.Method, req.URL, req.Body)
	if err != nil {
		return nil, err
	}

	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
	httpReq.Header.Del("Host")
	ApplyGitHubToken(httpReq, req.URL)

	client := network.GetGlobalHTTPClient()
	return client.Do(httpReq)
}

// validateResponse 验证下载响应的安全性。
func (s *DownloadService) validateResponse(resp *http.Response, req *DownloadRequest, result *DownloadResult) bool {
	contentType := resp.Header.Get("Content-Type")

	if req.Method == "GET" && network.IsBlockedContentType(contentType) {
		result.Error = fmt.Errorf("content type not allowed: %s", contentType)
		result.StatusCode = http.StatusForbidden
		return false
	}

	if exceeded, msg := network.CheckFileSize(resp.Header.Get("Content-Length"), s.config.Server.FileSize); exceeded {
		result.Error = fmt.Errorf(msg)
		result.StatusCode = http.StatusRequestEntityTooLarge
		return false
	}

	return true
}

// IsScriptURL 判断是否为脚本文件。
func (s *DownloadService) IsScriptURL(url string) bool {
	return IsScriptURL(url)
}

// ProcessScript 处理脚本文件（URL 替换）。
func (s *DownloadService) ProcessScript(body io.Reader, isGzipCompressed bool, realHost string) (io.Reader, int64, error) {
	return ProcessSmart(body, isGzipCompressed, realHost)
}

// CalculateContentLength 计算 Content-Length。
func (s *DownloadService) CalculateContentLength(resp *http.Response, preflightSize int64, isRangeRequest bool) int64 {
	var knownSize int64

	if isRangeRequest {
		if cl := resp.Header.Get("Content-Length"); cl != "" {
			if size, err := strconv.ParseInt(cl, 10, 64); err == nil && size > 0 {
				knownSize = size
			}
		}
	} else if preflightSize > 0 {
		knownSize = preflightSize
	} else {
		contentLength := resp.Header.Get("Content-Length")
		if contentLength != "" {
			if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil && size > 0 {
				knownSize = size
			}
		} else if resp.ContentLength > 0 {
			knownSize = resp.ContentLength
		} else if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
			if parts := strings.Split(contentRange, "/"); len(parts) == 2 {
				if total, err := strconv.ParseInt(parts[1], 10, 64); err == nil && total > 0 {
					knownSize = total
				}
			}
		}
	}

	return knownSize
}
