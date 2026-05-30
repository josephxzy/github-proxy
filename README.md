# Github-Proxy

专注于 GitHub 资源加速的轻量级反向代理工具。支持 Raw、Blob、Archive、Release、Gist 等全类型资源加速下载，内置 Vue 3 前端界面。

## 特性

- 🚀 全类型资源加速（Raw / Blob / Archive / Release / Gist）
- 📦 Releases 浏览与搜索，一键下载 ZIP
- 📊 下载进度条 + 断点续传（Range 探测协议）
- 🔗 脚本自动替换（`.sh` / `.ps1` 内 GitHub 链接自动替换为代理地址）
- 🌐 多节点支持与测速排序
- 🔒 仓库黑白名单 + 用户认证
- ⚡ API 分级限流 + IP 频率限制
- 🤝 可选加入分布式公益加速网络

## 快速开始

### Docker（推荐）

```bash
docker run -d \
  --name github-proxy \
  -p 5000:5000 \
  -v $(pwd)/config.toml:/app/config.toml \
  ghcr.io/AlistairBlake/github-proxy:latest
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
enableFrontend = true
githubToken = ""           # 可选，提升 API 限流 60→5000 次/小时

[rateLimit]
requestLimit = 500         # IP 请求频率限制
periodHours = 3.0
apiSearchHourly = 1200     # 各类 API 每小时限额
apiReleaseHourly = 3333
apiRepoHourly = 3333
apiOtherHourly = 3333

[access]
whiteList = []              # 仓库白名单
blackList = []              # 仓库黑名单
proxy = ""                  # 上游代理地址

[nodeRegistry]
urls = []                   # 调度中心地址（留空不加入公益网络）

[authUsers]
users = []                  # 认证用户 "用户名:密码"，留空不启用
```

| 环境变量 | 说明 | 默认值 |
|---------|------|--------|
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `5000` |
| `GITHUB_TOKEN` | GitHub PAT（提升 API 限流） | 空 |
| `REQUEST_LIMIT` | IP 频率限制 | `500` |
| `ACCESS_PROXY` | 上游代理地址 | 空 |
| `REPO_WHITELIST` | 仓库白名单（逗号分隔） | 空 |
| `REPO_BLACKLIST` | 仓库黑名单（逗号分隔） | 空 |
| `AUTH_USERS` | 认证用户列表（逗号分隔） | 空 |

## 使用方式

### 加速文件下载

在页面输入框粘贴 GitHub 链接即可：

| 类型 | 示例 |
|------|------|
| Raw 文件 | `https://raw.githubusercontent.com/user/repo/main/file.txt` |
| Blob 页面 | `https://github.com/user/repo/blob/main/file.txt` |
| Archive | `https://github.com/user/repo/archive/refs/heads/main.zip` |
| Release | `https://github.com/user/repo/releases/download/v1.0/file.zip` |

### 加速 Git Clone

```bash
git clone https://your-proxy.com/https://github.com/user/repo.git
```

### 用户认证

配置 `authUsers.users` 后，通过路径前缀认证：

```bash
# 认证用户（不限速）
git clone https://your-proxy.com/user1:pass1/https://github.com/user/repo.git
```

### 脚本自动替换

下载 `.sh` / `.ps1` 脚本时，内部所有 `github.com` / `githubusercontent.com` 链接会自动替换为当前代理地址。支持 gzip 压缩脚本，上限 10MB。

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

## 公益加速网络（可选）

多个实例通过统一的调度中心联合形成分布式加速网络。加入方法：

```toml
[nodeRegistry]
urls = ["https://registry.example.com"]
publicUrl = "https://hub.example.com"
```

将 `urls` 留空即可独立运行，不参与共享计划。退出只需清空 `urls` 并重启。

## API 端点

| 端点 | 方法 | 说明 |
|------|------|------|
| `/` | GET | 前端界面 |
| `/ready` | GET | 服务就绪检查 |
| `/api/nodes` | GET | 节点列表 |
| `/{github_url}` | GET/POST | 代理请求（核心） |

## 技术栈

- **后端**: Go 1.23+ / Gin / go-toml/v2
- **前端**: Vue 3 / Vite 5 / TailwindCSS 3

## License

MIT
