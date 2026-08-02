# Github-Proxy

专注于 GitHub 资源加速的轻量级反向代理工具，内置 Vue 3 前端界面，支持仓库搜索、Release 浏览、README 预览、Git 加速等。

## 部署

### 二进制

从 [Releases](https://github.com/josephxzy/github-proxy/releases) 下载对应平台的二进制，设置环境变量后直接运行即可。

### 构建

```bash
# Linux / macOS
./build.sh v1.2.0

# Windows
.\build.ps1 -Version v1.2.0
```

服务默认监听 `0.0.0.0:5000`。

### 反向代理

生产环境建议前置 Nginx / Caddy（详见[使用说明](docs/使用说明.md#反向代理https)）：

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

## 配置

所有配置项均可通过环境变量设置。

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `5000` |
| `ENABLE_FRONTEND` | 是否启用 Web 前端 | `true` |
| `GITHUB_TOKEN` | 服务器 PAT（API 限流兜底） | 空 |
| `TOKEN_WHITELIST` | Token 白名单，匹配则不限速 | 空 |
| `DOWNLOAD_RATE` | 单用户下载限速（字节/秒） | `0` |
| `GLOBAL_RATE` | 全局限速（字节/秒） | `0` |
| `IP_REQUEST_LIMIT` | IP 请求限制（次/小时） | `0` |
| `BUFFER_SIZE` | 水位线缓冲区（字节） | `8388608` |
| `MAX_FILE_SIZE` | 单文件最大大小（字节） | `2147483648` |
| `API_SEARCH_HOURLY` | 搜索 API 每小时限额 | `1200` |
| `API_RELEASE_HOURLY` | Release API 每小时限额 | `3333` |
| `API_REPO_HOURLY` | Repo API 每小时限额 | `3333` |
| `API_OTHER_HOURLY` | 其他 API 每小时限额 | `3333` |
| `ACCESS_PROXY` | 上游代理地址 | 空 |
| `REPO_WHITELIST` | 仓库白名单（逗号分隔） | 空 |
| `REPO_BLACKLIST` | 仓库黑名单（逗号分隔） | 空 |

完整配置参考 `src/config.toml`。

## 文档

| 类型 | 位置 | 说明 |
|:---|:---|:---|
| 用户文档站 | [site](site) | React + Vite 构建，部署于 GitHub Pages |
| 设计文档 | [docs/design](docs/design) | 系统设计、模块接口、领域建模 |
| 项目 Wiki | [docs/wiki](docs/wiki) | 长期架构规则、工作流边界、调试经验 |
| 发布历史 | [docs/releases](docs/releases) | 用户可见版本更新说明 |

完整文档管理约定见 [docs/README.md](docs/README.md)。

## License

MIT
