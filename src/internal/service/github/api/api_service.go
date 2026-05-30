package github

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github-proxy/pkg/network"
)

// APIService GitHub API 代理服务类
// **专门处理 GitHub JSON API 请求**：
//   - Release 列表查询 (api.github.com/repos/.../releases)
//   - Tag 信息获取
//   - 其他 REST API 端点
//
// 特点（与 DownloadService 对比）：
//
//	特性                 APIService	              DownloadService
//	Range 预检	            不执行                      执行
//	内容类型检查              跳过	                     执行
//	文件大小限制	          跳过	                     执行
//	带宽限制	              无                    有（未认证用户）
//	脚本 URL 替换	        不执行                  执行 (.sh/.ps1)
//	传输方式	        io.Copy 直接转发	     streamWithLimit 流式传输
//	Token 认证	         Release API	            仅下载不需要
type APIService struct {
}

// NewAPIService 创建 API 代理服务实例。
func NewAPIService() *APIService {
	return &APIService{}
}

// APIRequest API 请求参数。
type APIRequest struct {
	Context context.Context // 请求上下文（用于超时和取消）
	Method  string          // HTTP 方法
	URL     string          // 目标 URL
	Headers http.Header     // 请求头
	Body    io.Reader       // 请求体
}

// APIResult API 请求结果。
type APIResult struct {
	StatusCode int         // HTTP 状态码
	Headers    http.Header // 响应头
	Body       io.Reader   // 响应体
	Error      error       // 错误信息
}

// Execute 执行 GitHub API 代理请求。
// 处理流程：
//  1. 检查 API 速率限制队列
//  2. 创建 HTTP 请求并应用认证
//  3. 发送请求到 GitHub
//  4. 清理安全响应头
//  5. 返回结果
func (s *APIService) Execute(req *APIRequest) *APIResult {
	result := &APIResult{}

	// 步骤1：检查速率限制
	if err := CheckAPIQueue(req.Context, req.URL); err != nil {
		result.Error = fmt.Errorf("queue wait failed: %v", err)
		result.StatusCode = http.StatusGatewayTimeout
		return result
	}

	// 步骤2：创建请求
	httpReq, err := s.createRequest(req)
	if err != nil {
		result.Error = fmt.Errorf("create request failed: %v", err)
		result.StatusCode = http.StatusInternalServerError
		return result
	}

	// 步骤3-4：执行请求并处理响应
	httpResp, err := s.doRequest(httpReq, req.Context)
	if err != nil {
		if req.Context.Err() != nil {
			result.Error = req.Context.Err()
			return result
		}
		result.Error = fmt.Errorf("request failed: %v", err)
		result.StatusCode = http.StatusInternalServerError
		return result
	}
	defer httpResp.Body.Close()

	// 清理安全相关的响应头
	network.CleanSecurityHeaders(httpResp.Header)

	// 设置返回值
	result.StatusCode = httpResp.StatusCode
	result.Headers = httpResp.Header
	result.Body = httpResp.Body

	return result
}

// createRequest 创建 HTTP 请求并应用认证头。
func (s *APIService) createRequest(req *APIRequest) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(req.Context, req.Method, req.URL, req.Body)
	if err != nil {
		return nil, err
	}

	// 复制所有请求头
	for key, values := range req.Headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
	httpReq.Header.Del("Host")

	// 应用 GitHub Token 认证
	s.applyAuth(httpReq, req.URL)

	return httpReq, nil
}

// doRequest 执行 HTTP 请求并处理重定向。
func (s *APIService) doRequest(httpReq *http.Request, ctx context.Context) (*http.Response, error) {
	client := network.GetGlobalHTTPClient()
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	// 处理重定向
	if location := resp.Header.Get("Location"); location != "" {
		if _, needRedirect := network.HandleRedirectLocation(nil, location, MatchURL); needRedirect {
			resp.Body.Close()
			redirectReq, _ := http.NewRequestWithContext(ctx, httpReq.Method, location, httpReq.Body)
			s.applyAuth(redirectReq, location)
			return client.Do(redirectReq)
		}
	}

	return resp, nil
}

// applyAuth 应用 GitHub Token 认证（仅 Release API）。
func (s *APIService) applyAuth(req *http.Request, url string) {
	ApplyGitHubToken(req, url)
}

// IsAPIURL 判断是否是 GitHub API URL。
func (s *APIService) IsAPIURL(url string) bool {
	return IsGitHubAPIURL(url)
}
