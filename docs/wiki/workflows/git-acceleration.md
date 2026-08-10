# Git 加速工作流

## 背景

github-proxy 的核心场景之一是加速 Git 操作（clone、fetch、push）。用户通过修改 remote URL 将 Git 流量路由到代理，代理转发到 GitHub。关键挑战：Git 的认证方式（Basic Auth）与浏览器的认证方式（HTTP Header）完全不同。

## 决策

**代理从 Git 的 Basic Auth 中提取 PAT，统一转换为 `Authorization: token <value>` 格式**。用户无需额外配置，在 remote URL 中嵌入 PAT 即可同时完成认证和加速。

## 当前规则

- 使用短路径格式：`git clone https://hub.example.com/user/repo.git`
- 私有仓库在 URL 中嵌入 PAT：`https://ghp_xxx@hub.example.com/user/repo.git`
- 代理从 `Authorization: Basic` 中提取 PAT，转换为 `token` 格式发给 GitHub
- 前端 localStorage 中的 token 对 Git 操作无效（终端无法读取浏览器数据）
- Windows Git Credential Manager 弹出的凭据窗口兼容——首次认证后 GCM 缓存凭据

## 示例

**推荐做法**：
```bash
# 公开仓库
git clone https://hub.example.com/user/public-repo.git

# 私有仓库
git clone https://ghp_xxx@hub.example.com/user/private-repo.git
```

**禁止做法**：
- 在 URL 中同时嵌入 token 和设置 `X-GitHub-Token` 头（token 只会从一个来源提取）
- 依赖前端 token 来完成 Git 认证

## 失败模式

| 症状 | 原因 | 排查 |
|:---|:---|:---|
| Git clone 403 | 私有仓库未嵌入 token | 检查 URL 是否包含 `ghp_xxx@` |
| Git push 很慢 | 代理未正确配置 | 检查 `SERVER_HOST` 和 `SERVER_PORT` |
| Windows 弹凭据窗口 | 正常行为，GCM 自动处理 | 输入 GitHub 用户名 + PAT 即可 |

## 相关模块

- `internal/handlers/github.go` — `ExtractToken`、Basic Auth 解析
- `internal/handlers/proxy_download.go` — POST 请求处理（Git Push）

## 来源文档

- Token 透传与认证：`../design/token.md`