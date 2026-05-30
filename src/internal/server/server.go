package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github-proxy/config"
)

// Server HTTP 服务器封装。
// 封装 Go 标准库的 http.Server，提供简洁的启动和关闭接口
type Server struct {
	httpServer *http.Server      // 底层的 HTTP 服务器实例
	config     *config.AppConfig // 应用配置引用
}

// NewServer 创建新的 HTTP 服务器实例。
//
// 参数:
//   - cfg: 应用配置（包含监听地址、端口等）
//   - router: HTTP 处理器（通常是 Gin 路由引擎）
//
// 超时配置：
//   - ReadTimeout: 5分钟 - 读取请求头的超时时间（适应大文件上传场景）
//   - WriteTimeout: 30分钟 - 写入响应的超时时间（适应大文件下载场景）
//   - IdleTimeout: 10分钟 - 空闲连接的超时时间
func NewServer(cfg *config.AppConfig, router http.Handler) *Server {
	// 组合监听地址和端口
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	return &Server{
		httpServer: &http.Server{
			Addr:         addr,               // 监听地址
			Handler:      router,             // 请求处理器
			ReadTimeout:  5 * time.Minute,    // 读取超时
			WriteTimeout: 30 * time.Minute,   // 写入超时（大文件下载需要较长时间）
			IdleTimeout:  10 * time.Minute,   // 空闲超时
		},
		config: cfg,
	}
}

// Start 启动 HTTP 服务器（阻塞调用）。
// 此函数会阻塞当前 goroutine，直到服务器停止或发生错误
// 通常在 main 函数的最后调用
func (s *Server) Start() error {
	fmt.Printf("Server starting on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown 优雅关闭服务器。
// 使用 context 实现超时控制，确保正在处理的请求能够完成
// 应该在收到中断信号（如 SIGTERM）时调用
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Addr 返回服务器监听地址。
// 格式为 "host:port"，如 "0.0.0.0:5000"
func (s *Server) Addr() string {
	return s.httpServer.Addr
}
