package service

import (
	"context"
	"io"
	"net/http"
	"time"

	"github-proxy/config"
	ghproxyservice "github-proxy/internal/service/github"
)

// ProxyService GitHub 代理服务**编排层**。
// **不直接处理业务逻辑，仅负责路由分发**：
//
//	客户端请求:
//	    │
//	    ├─ IsAPIURL? ── yes ──→ github.APIService.Execute()
//	    │                          （Release 列表、JSON API）
//	    │
//	    └─ no ───────────────→ github.DownloadService.Execute()
//	                            （文件下载、Range 等）
//
// 职责:
//   - 请求路由分发（API vs 下载）
//   - 统一错误处理和日志
//   - 提供对底层服务的统一访问接口
type ProxyService struct {
	config      *config.AppConfig
	downloadSvc *ghproxyservice.DownloadService
	apiSvc      *ghproxyservice.APIService
}

// NewProxyService 创建代理服务编排层实例。
func NewProxyService(
	cfg *config.AppConfig,
) *ProxyService {
	return &ProxyService{
		config:      cfg,
		downloadSvc: ghproxyservice.NewDownloadService(cfg),
		apiSvc:      ghproxyservice.NewAPIService(),
	}
}

// ProxyRequest 统一的代理请求参数。
type ProxyRequest struct {
	Context       context.Context
	Method        string
	URL           string
	Headers       http.Header
	Body          io.Reader
	IsRange       bool
	Authenticated bool
}

// ProxyRequestResult 统一的代理请求结果。
type ProxyRequestResult struct {
	StatusCode int
	Headers    http.Header
	Body       io.Reader
	Error      error
	Latency    int
	IsAPI      bool // 标识走的是哪条路径
}

// Execute 执行代理请求并自动路由到正确的服务。
func (s *ProxyService) Execute(req *ProxyRequest) *ProxyRequestResult {
	result := &ProxyRequestResult{}
	startTime := time.Now()

	if s.isAPIRequest(req.URL) {
		result.IsAPI = true
		apiResult := s.apiSvc.Execute(&ghproxyservice.APIRequest{
			Context: req.Context,
			Method:  req.Method,
			URL:     req.URL,
			Headers: req.Headers,
			Body:    req.Body,
		})

		result.StatusCode = apiResult.StatusCode
		result.Headers = apiResult.Headers
		result.Body = apiResult.Body
		result.Error = apiResult.Error
	} else {
		result.IsAPI = false
		downloadResult := s.downloadSvc.Execute(&ghproxyservice.DownloadRequest{
			Context:       req.Context,
			Method:        req.Method,
			URL:           req.URL,
			Headers:       req.Headers,
			Body:          req.Body,
			IsRange:       req.IsRange,
			Authenticated: req.Authenticated,
		})

		result.StatusCode = downloadResult.StatusCode
		result.Headers = downloadResult.Headers
		result.Body = downloadResult.Body
		result.Error = downloadResult.Error
		result.Latency = downloadResult.Latency
	}

	result.Latency = int(time.Since(startTime).Milliseconds())
	return result
}

// isAPIRequest 判断是否是 API 请求。
func (s *ProxyService) isAPIRequest(url string) bool {
	return ghproxyservice.IsAPIRequest(url)
}

// GetDownloadService 获取下载服务实例。
func (s *ProxyService) GetDownloadService() *ghproxyservice.DownloadService {
	return s.downloadSvc
}

// GetAPIService 获取 API 服务实例。
func (s *ProxyService) GetAPIService() *ghproxyservice.APIService {
	return s.apiSvc
}

// GetConfig 返回应用配置。
func (s *ProxyService) GetConfig() *config.AppConfig {
	return s.config
}
