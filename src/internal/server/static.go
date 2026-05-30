package server

import (
	"embed"
	"io/fs"
	"strings"

	"github.com/gin-gonic/gin"
)

// StaticFileSystem 静态文件系统接口。
// 定义了从嵌入式文件系统读取文件的最小接口
// 支持使用 embed.FS 或其他实现了此接口的文件系统
type StaticFileSystem interface {
	ReadFile(name string) ([]byte, error)       // 读取文件内容
	ReadDir(name string) ([]fs.DirEntry, error) // 读取目录内容
}

// ServeEmbedFile 从文件系统中提供静态文件。
// 根据文件扩展名自动设置正确的 Content-Type。
//
// 参数:
//   - c: Gin 上下文
//   - filesystem: 静态文件系统实现
//   - filename: 要提供的文件路径（相对于嵌入的根目录）
func ServeEmbedFile(c *gin.Context, filesystem StaticFileSystem, filename string) {
	// 从文件系统中读取文件内容
	data, err := filesystem.ReadFile(filename)
	if err != nil {
		// 文件不存在或读取失败，返回 404
		c.Status(404)
		return
	}

	// 根据文件扩展名检测 MIME 类型
	contentType := detectContentType(filename)

	switch {
	case strings.HasSuffix(filename, ".html"):
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
	case strings.HasSuffix(filename, ".js"), strings.HasSuffix(filename, ".css"):
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	default:
		c.Header("Cache-Control", "public, max-age=86400")
	}

	// 将文件内容写入响应（状态码 200）
	c.Data(200, contentType, data)
}

// detectContentType 根据文件扩展名返回 MIME 类型。
// 支持常见的 Web 资源格式，包括图标、脚本、样式表、字体等
func detectContentType(filename string) string {
	switch {
	case strings.HasSuffix(filename, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(filename, ".js"):
		return "application/javascript"
	case strings.HasSuffix(filename, ".css"):
		return "text/css"
	case strings.HasSuffix(filename, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(filename, ".json"):
		return "application/json"
	case strings.HasSuffix(filename, ".png"):
		return "image/png"
	case strings.HasSuffix(filename, ".jpg"), strings.HasSuffix(filename, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(filename, ".gif"):
		return "image/gif"
	case strings.HasSuffix(filename, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(filename, ".woff"):
		return "font/woff"
	case strings.HasSuffix(filename, ".ttf"):
		return "font/ttf"
	default:
		// 默认返回 HTML 类型
		return "text/html; charset=utf-8"
	}
}

// SetCORSSettings 设置跨域响应头。
// 允许来自任何源的 GET 和 OPTIONS 请求
// 主要用于 favicon 等可能被跨域引用的资源
func SetCORSSettings(c *gin.Context) {
	c.Header("Access-Control-Allow-Origin", "*")             // 允许所有来源
	c.Header("Access-Control-Allow-Methods", "GET, OPTIONS") // 允许的方法
	c.Header("Access-Control-Allow-Headers", "*")            // 允许的请求头
}

// EmbedFSWrapper embed.FS 包装器实现 StaticFileSystem 接口。
// 将 Go 的嵌入文件系统适配到我们的接口
// 使得可以使用 //go:embed 嵌入的前端资源
type EmbedFSWrapper struct {
	FS embed.FS // 底层的 embed.FS 实例
}

// ReadFile 实现 StaticFileSystem 接口的 ReadFile 方法
func (e *EmbedFSWrapper) ReadFile(name string) ([]byte, error) {
	return e.FS.ReadFile(name)
}

// ReadDir 实现 StaticFileSystem 接口的 ReadDir 方法
func (e *EmbedFSWrapper) ReadDir(name string) ([]fs.DirEntry, error) {
	return e.FS.ReadDir(name)
}
