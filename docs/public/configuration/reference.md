# 配置参考

所有配置项均可通过环境变量设置。完整配置参考 `src/config.toml`。

## 服务器配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `5000` |
| `ENABLE_FRONTEND` | 是否启用 Web 前端 | `true` |

## 认证配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `GITHUB_TOKEN` | 服务器 PAT（API 限流兜底） | 空 |
| `TOKEN_WHITELIST` | Token 白名单，匹配则不限速 | 空 |

## 限速配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `DOWNLOAD_RATE` | 单用户下载限速（字节/秒） | `0` |
| `GLOBAL_RATE` | 全局限速（字节/秒） | `0` |
| `IP_REQUEST_LIMIT` | IP 请求限制（次/小时） | `0` |

## 缓冲区配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `BUFFER_SIZE` | 水位线缓冲区（字节） | `8388608` (8MB) |
| `MAX_FILE_SIZE` | 单文件最大大小（字节） | `2147483648` (2GB) |

## API 限流配置

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `API_SEARCH_HOURLY` | 搜索 API 每小时限额 | `1200` |
| `API_RELEASE_HOURLY` | Release API 每小时限额 | `3333` |
| `API_REPO_HOURLY` | Repo API 每小时限额 | `3333` |
| `API_OTHER_HOURLY` | 其他 API 每小时限额 | `3333` |

## 访问控制

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `ACCESS_PROXY` | 上游代理地址 | 空 |
| `REPO_WHITELIST` | 仓库白名单（逗号分隔） | 空 |
| `REPO_BLACKLIST` | 仓库黑名单（逗号分隔） | 空 |

## config.toml 示例

```toml
[server]
host = "0.0.0.0"
port = 5000
enableFrontend = true

[rateLimit]
downloadBytesPerSec = 0
globalBytesPerSec = 0
ipRequestLimit = 0
bufferSize = 8388608
maxFileSize = 2147483648

[access]
accessProxy = ""
repoWhitelist = []
repoBlacklist = []

[tokenWhiteList]
tokens = []

[apiLimits]
searchHourly = 1200
releaseHourly = 3333
repoHourly = 3333
otherHourly = 3333
```