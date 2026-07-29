# Github-Proxy

> **项目迁移通知**：原仓库 `AlistairBlake/github-proxy` 已弃用，项目已迁移至 `josephxzy/github-proxy`，请更新您的远程地址和书签。

专注于 GitHub 资源加速的轻量级反向代理工具。支持 Raw、Blob、Archive、Release、Gist 等全类型资源加速下载，内置 Vue 3 前端界面。

## 特性

- 🚀 全类型资源加速（Raw / Blob / Archive / Release / Gist）
- 📦 Releases 浏览与搜索，一键下载 ZIP
- 🔍 仓库搜索（支持排序、范围筛选），一键查看 README
- 📖 README 在线预览，GitHub 链接自动替换为加速域名
- 📊 下载进度条 + 断点续传（Range 探测协议）
- 🔗 脚本自动替换（`.sh` / `.ps1` 内 GitHub 链接自动替换为代理地址）
- 🔒 仓库黑白名单
- 🔑 Token 白名单免限速 + 私有仓库访问
- ⚡ API 分级限流

## 快速开始

### Docker（推荐）

```bash
docker run -d \
  --name github-proxy \
  -p 5000:5000 \
  -v $(pwd)/config.toml:/app/config.toml \
  ghcr.io/josephxzy/github-proxy:latest
```

或使用 Docker Compose：

```bash
docker compose up -d --build
```

### 从源码构建

**一键脚本：**

```bash
# Linux / macOS
./build.sh v1.2.0

# Windows
.\build.ps1 -Version v1.2.0
```

**手动编译：**

```bash
# 1. 编译前端
cd src/frontend && npm install && npm run build && cd ../..

# 2. 编译后端
cd src && go build -o github-proxy .

# 3. 运行
./github-proxy
```

服务默认监听 `0.0.0.0:5000`，访问 `http://your-ip:5000` 即可使用。

## 配置

编辑 `config.toml` 或通过环境变量配置（环境变量优先级更高）：

```toml
[server]
host = "0.0.0.0"
port = 5000
fileSize = 2147483648       # 单文件大小限制（字节），默认 2GB
enableFrontend = true
githubToken = ""            # 可选，服务器 PAT（仅用于 API 限流，不影响下载）
bufferSize = 8388608         # 水位线缓冲区大小（字节），默认 8MB

[rateLimit]
apiSearchHourly = 1200     # 各类 API 每小时限额
apiReleaseHourly = 3333
apiRepoHourly = 3333
apiOtherHourly = 3333
downloadBytesPerSec = 0    # 单用户下载限速（字节/秒），0=不限速
globalBytesPerSec = 0      # 全局限速（字节/秒），所有未认证用户共享

[access]
whiteList = []              # 仓库白名单
blackList = []              # 仓库黑名单
proxy = ""                  # 上游代理地址

[tokenWhiteList]
tokens = []                 # Token 白名单，匹配则不限速
```

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `5000` |
| `ENABLE_FRONTEND` | 是否启用 Web 前端 | `true` |
| `GITHUB_TOKEN` | 服务器 PAT（仅用于 API 限流，不影响下载和 Git） | 空 |
| `BUFFER_SIZE` | 水位线缓冲区大小（字节），默认 8MB | `8388608` |
| `MAX_FILE_SIZE` | 单文件最大大小（字节） | `2147483648` (2GB) |
| `API_SEARCH_HOURLY` | 搜索 API 每小时限额 | `1200` |
| `API_RELEASE_HOURLY` | Release API 每小时限额 | `3333` |
| `API_REPO_HOURLY` | Repo API 每小时限额 | `3333` |
| `API_OTHER_HOURLY` | 其他 API 每小时限额 | `3333` |
| `DOWNLOAD_RATE` | 单用户下载限速（字节/秒），0=不限速 | `0` |
| `GLOBAL_RATE` | 全局限速（字节/秒），所有未认证用户共享 | `0` |
| `ACCESS_PROXY` | 上游代理地址 | 空 |
| `REPO_WHITELIST` | 仓库白名单（逗号分隔） | 空 |
| `REPO_BLACKLIST` | 仓库黑名单（逗号分隔） | 空 |
| `TOKEN_WHITELIST` | Token 白名单（逗号分隔） | 空 |

## 使用方式

### URL 格式

代理接受两种等价的 URL 格式，以 Release 下载为例：

| 格式 | 示例 |
|------|------|
| 完整 URL | `https://hub.xzyuse.site/https://github.com/user/repo/releases/download/v1.0/file.zip` |
| 短路径 | `https://hub.xzyuse.site/user/repo/releases/download/v1.0/file.zip` |

> **短路径限制**：短路径格式会被自动补全为 `https://github.com/` 前缀，因此仅适用于 `github.com` 域名下的资源。`raw.githubusercontent.com` 等子域名链接**必须**使用完整 URL 格式。

### 网页端加速下载

在页面输入框粘贴 GitHub 链接即可（* 表示不支持短路径格式）：

| 类型 | 示例 |
|------|------|
| Raw 文件 * | `https://raw.githubusercontent.com/user/repo/main/file.txt` |
| Blob 页面 | `https://github.com/user/repo/blob/main/file.txt` |
| Archive | `https://github.com/user/repo/archive/refs/heads/main.zip` |
| Release | `https://github.com/user/repo/releases/download/v1.0/file.zip` |

### 加速 Git Clone / Fetch / Push

```bash
# 两种等价写法
git clone https://hub.xzyuse.site/https://github.com/user/repo.git
git clone https://hub.xzyuse.site/user/repo.git
```

短路径格式同样可用于 git push 等操作。

### Git Push（私有仓库）

Push 需要 GitHub 认证。将凭据嵌入 remote URL 即可：

```bash
git remote set-url origin https://ghp_xxx@hub.xzyuse.site/https://github.com/user/repo.git
git push
```

也可用短路径：

```bash
git remote set-url origin https://ghp_xxx@hub.xzyuse.site/user/repo.git
git push
```

如果希望 push 走直连、clone/fetch 走代理：

```bash
git config --global url.https://github.com/.pushInsteadOf https://hub.xzyuse.site/https://github.com/
```

### 私有仓库网页访问

点击页面搜索框下方的「私有仓库」开关，在弹窗中填入你的 GitHub Personal Access Token 后：

- 搜索、Release 列表、README 预览等 API 请求会携带 `X-GitHub-Token` 请求头
- 文件下载链接会自动拼接 `?token=` 查询参数
- Token 保存在浏览器 localStorage 中，刷新不丢失

Token 需具备 `repo` 权限。获取方式：https://github.com/settings/tokens

### Token 白名单

在 `config.toml` 中配置 token 白名单，匹配的 token 免除限速：

```toml
[tokenWhiteList]
tokens = ["ghp_xxx", "ghp_yyy"]
```

也可通过环境变量：`TOKEN_WHITELIST=ghp_xxx,ghp_yyy`

用户通过前端输入框、`?token=` 参数或 git 嵌入凭据传入的 token，均在白名单匹配范围内。未传入 token、或 token 不在白名单中，均受限速。

### 脚本自动替换

下载 `.sh` / `.ps1` 脚本时，内部所有 `github.com` / `githubusercontent.com` 链接会自动替换为当前代理地址。支持 gzip 压缩脚本，上限 10MB。

### 仓库搜索与 README 预览

在首页搜索框输入关键词或 `user/repo` 格式，可搜索 GitHub 仓库。搜索结果支持按星标数、Fork 数、最近更新排序，以及按名称、描述、README 范围筛选。

每个仓库卡片提供 **下载 ZIP**、**Releases**、**README** 三个按钮。点击 **README** 按钮会弹窗展示该仓库的 README 文件，自动渲染 Markdown 内容，且 README 中所有 `github.com` / `raw.githubusercontent.com` / `user-images.githubusercontent.com` 链接会自动替换为当前加速站域名。

---

## 仓库黑白名单

| 格式 | 含义 | 示例 |
|------|------|------|
| `user/repo` | 精确匹配仓库 | `AlistairBlake/github-proxy` |
| `user` | 匹配用户所有仓库 | `AlistairBlake` |
| `user/*` | 通配符匹配 | `AlistairBlake/*` |
| `prefix*` | 前缀匹配 | `user/proj-*` |

白名单优先级高于黑名单。两者都为空时不做限制。

## 反向代理（HTTPS）

生产环境建议前置 Nginx / Caddy：

```nginx
server {
    listen 443 ssl;
    server_name hub.example.com;

    ssl_certificate     /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://127.0.0.1:5000;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # 关闭压缩以保留下载进度条的 Content-Length
        gzip off;
        proxy_buffering off;
    }
}
```

```caddy
hub.example.com {
    reverse_proxy 127.0.0.1:5000
}
```

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 前端界面 |
| `/ready` | GET | 服务就绪检查 |
| `/{github_url}` | GET/POST | 代理请求（核心） |

用户 GitHub Token 可通过请求头、查询参数或 Git 凭据传入。详见 [Token 透传文档](docs/Token透传与认证机制.md)。

## 技术栈

- **后端**: Go 1.23+ / Gin / go-toml/v2
- **前端**: Vue 3 / Vite 5 / TailwindCSS 3

## 深入阅读

- [Token 透传与认证机制](docs/Token透传与认证机制.md) — token 提取、应用、白名单的完整流程
- [限速与下载稳定性方案](docs/下载限速与稳定性设计.md) — 水位线反压 + per-user token bucket 的设计细节

## License

MIT
