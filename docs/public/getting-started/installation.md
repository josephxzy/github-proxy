# 安装部署

## 二进制部署

从 [Releases](https://github.com/josephxzy/github-proxy/releases) 下载对应平台的二进制文件，设置环境变量后直接运行。

```bash
# Linux / macOS
chmod +x github-proxy
./github-proxy

# Windows
github-proxy.exe
```

服务默认监听 `0.0.0.0:5000`。

## 源码构建

### 前置要求

- Go 1.23+（`go.mod` 声明 `go 1.23.0`）
- Node.js 18+

### 构建步骤

```bash
# Linux / macOS
./build.sh v1.2.0

# Windows
.\build.ps1 -Version v1.2.0
```

构建脚本编译 Go 二进制并打包前端资源，输出到 `build/` 目录。

## 环境变量

所有配置项均可通过环境变量设置，详见 [配置参考](#/docs/config-reference)。

最小配置：

```bash
export GITHUB_TOKEN=ghp_xxx  # 可选，提升 API 限流
export SERVER_PORT=5000       # 可选，默认 5000
```

## Nginx 反向代理

生产环境建议前置 Nginx / Caddy：

```nginx
location / {
    proxy_pass http://127.0.0.1:5000;
    gzip off;
    brotli off;
    zstd off;
    proxy_buffering off;
}
```

> 必须关闭压缩和缓冲，否则下载无进度条。