# Github-Proxy

> **项目迁移通知**：原仓库 `AlistairBlake/github-proxy` 已弃用，项目已迁移至 `josephxzy/github-proxy`，请更新您的远程地址和书签。

专注于 GitHub 资源加速的轻量级反向代理工具。支持 Raw、Blob、Archive、Release、Gist 等全类型资源加速下载，内置 Vue 3 前端界面。

## 特性

- 🚀 全类型资源加速（Raw / Blob / Archive / Release / Gist）
- 📦 Releases 浏览与搜索，一键下载 ZIP
- 📊 下载进度条 + 断点续传（Range 探测协议）
- 🔗 脚本自动替换（`.sh` / `.ps1` 内 GitHub 链接自动替换为代理地址）
- 🔒 仓库黑白名单 + 用户认证
- 🔑 私有仓库访问（前端设置 GitHub Token）
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
./build.sh v1.0.0

# Windows
.\build.ps1 -Version v1.0.0
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
githubToken = ""            # 可选，提升 API 限流 60→5000 次/小时

[rateLimit]
apiSearchHourly = 1200     # 各类 API 每小时限额
apiReleaseHourly = 3333
apiRepoHourly = 3333
apiOtherHourly = 3333

[access]
whiteList = []              # 仓库白名单
blackList = []              # 仓库黑名单
proxy = ""                  # 上游代理地址

[authUsers]
users = []                  # 认证用户 "用户名:密码"，留空不启用
```

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `5000` |
| `ENABLE_FRONTEND` | 是否启用 Web 前端 | `true` |
| `GITHUB_TOKEN` | GitHub PAT（提升 API 限流） | 空 |
| `MAX_FILE_SIZE` | 单文件最大大小（字节） | `2147483648` (2GB) |
| `API_SEARCH_HOURLY` | 搜索 API 每小时限额 | `1200` |
| `API_RELEASE_HOURLY` | Release API 每小时限额 | `3333` |
| `API_REPO_HOURLY` | Repo API 每小时限额 | `3333` |
| `API_OTHER_HOURLY` | 其他 API 每小时限额 | `3333` |
| `ACCESS_PROXY` | 上游代理地址 | 空 |
| `REPO_WHITELIST` | 仓库白名单（逗号分隔） | 空 |
| `REPO_BLACKLIST` | 仓库黑名单（逗号分隔） | 空 |
| `AUTH_USERS` | 认证用户列表（逗号分隔） | 空 |

## 使用方式

### URL 格式

代理接受三种等价的 URL 格式，以 Release 下载为例：

| 格式 | 示例 |
|------|------|
| 完整 URL | `https://hub.xzyuse.site/https://github.com/user/repo/releases/download/v1.0/file.zip` |
| 省略协议 | `https://hub.xzyuse.site/github.com/user/repo/releases/download/v1.0/file.zip` |
| 短路径 | `https://hub.xzyuse.site/user/repo/releases/download/v1.0/file.zip` |

> 短路径格式要求第一个路径段不含 `.`，以此区分 `github.com` 域名和 `user/repo` 短写。

### 网页端加速下载

在页面输入框粘贴 GitHub 链接即可：

| 类型 | 示例 |
|------|------|
| Raw 文件 | `https://raw.githubusercontent.com/user/repo/main/file.txt` |
| Blob 页面 | `https://github.com/user/repo/blob/main/file.txt` |
| Archive | `https://github.com/user/repo/archive/refs/heads/main.zip` |
| Release | `https://github.com/user/repo/releases/download/v1.0/file.zip` |

### 加速 Git Clone / Fetch

```bash
# 三种等价写法
git clone https://hub.xzyuse.site/https://github.com/user/repo.git
git clone https://hub.xzyuse.site/github.com/user/repo.git
git clone https://hub.xzyuse.site/user/repo.git
```

### Git Push（私有仓库）

Push 需要 GitHub 认证。直接将凭据嵌入 URL 即可，代理会透传 `Authorization` 头给 GitHub：

```bash
git remote set-url origin https://用户名:ghp_xxx@hub.xzyuse.site/https://github.com/user/repo.git
git push
```

Push 走代理的话，Git 会提示需要访问 `hub.xzyuse.site` 的凭据（因为它不认识这个 Host）。输入你的 GitHub 用户名 + Token 即可通过。

如果希望 push 走直连、clone/fetch 走代理：

```bash
git config --global url.https://github.com/.pushInsteadOf https://hub.xzyuse.site/https://github.com/
```

### 私有仓库网页访问

页面搜索框下方有「私有仓库访问（设置 GitHub Token）」折叠面板，填入你的 GitHub Personal Access Token 后：

- 搜索、Release 列表等 API 请求会携带 `X-GitHub-Token` 请求头
- 文件下载链接会自动拼接 `?token=` 查询参数
- Token 保存在浏览器 localStorage 中，刷新不丢失

Token 需具备 `repo` 权限。获取方式：https://github.com/settings/tokens

### 三种认证 / Token 机制

| 机制 | 配置位置 | 格式 | 作用 |
|------|---------|------|------|
| 服务器 PAT | `config.toml` → `githubToken` | — | 提高 Release API 限流（60→5000 次/小时），仅对 api.github.com/repos/*/releases 生效 |
| 代理自身认证 | `config.toml` → `[authUsers].users` | URL 路径 `user:pass/` | 控制谁能使用代理站（认证用户不限速） |
| 用户 GitHub Token | 前端输入或 URL 嵌入 | `?token=` / `X-GitHub-Token` 头 / `user:token@` | 以你自己的身份访问 GitHub（私有仓库、Push 认证） |

示例——三种机制可同时使用互不冲突：

```bash
# 同时满足：代理站需要身份 + Git 操作需要 Token
git clone https://proxyUser:proxyPass@hub.xzyuse.site/https://githubUser:ghp_xxx@github.com/user/private-repo.git
```

### 脚本自动替换

下载 `.sh` / `.ps1` 脚本时，内部所有 `github.com` / `githubusercontent.com` 链接会自动替换为当前代理地址。支持 gzip 压缩脚本，上限 10MB。

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

所有代理请求支持以下方式传递用户 GitHub Token：

| 方式 | 示例 | 适用场景 |
|------|------|---------|
| HTTP 头 `X-GitHub-Token` | `curl -H "X-GitHub-Token: ghp_xxx"` | 同源 API 请求 |
| 查询参数 `?token=` | `/user/repo/archive/main.zip?token=ghp_xxx` | 新窗口下载 |
| HTTP Basic Auth | `https://user:ghp_xxx@proxy.com/https://github.com/...` | Git 操作 |

## 技术栈

- **后端**: Go 1.23+ / Gin / go-toml/v2
- **前端**: Vue 3 / Vite 5 / TailwindCSS 3

## License

MIT
