# Token 透传与认证

github-proxy 统一从三个来源提取 token，合并为一份后应用到所有出站请求。

## Token 提取（唯一入口）

```go
func ExtractToken(r *http.Request, rawPath string) string {
    // 依次检查，取第一个非空的
    // 1. X-GitHub-Token 请求头（前端 fetch、curl）
    // 2. ?token= 查询参数（新窗口下载、分享链接）
    // 3. Authorization: Basic 头（Git 操作的嵌入凭据）
}
```

三种来源对应三种使用场景，一条请求只会命中其中一个——浏览器不会同时发请求头又嵌 Basic Auth，git 不会在 URL 里带 `?token=`。

之后 token 从 URL 中删除（`?token=` 参数不会发给 GitHub），但 Basic Auth 头仍保留在原始请求中（后面会被统一应用覆盖）。

### Basic Auth 解析

```go
func extractTokenFromBasicAuth(r *http.Request) string {
    // 解码 Basic Auth → base64(user:password)
    // 返回 password（如果非空），否则返回 user 作为回退
}
```

Git 在 URL 中嵌入凭据时，`https://ghp_xxx@host/repo.git` 会发送 `Basic base64(ghp_xxx:)`，代理从用户名回退中提取 Token。`https://user:ghp_xxx@host/repo.git` 会发送 `Basic base64(user:ghp_xxx)`，代理从密码字段提取。两种写法均可。

## Token 应用

提取到的 token 存入 Gin Context，下游统一应用：

```go
req.Header.Set("Authorization", "token "+token)
```

**一个 token，一种格式，所有请求**。不管原来源是什么，最终都以 `Authorization: token <value>` 发给 GitHub。

## 服务器 Token（`GITHUB_TOKEN`）

仅用于兜底：当请求是 GitHub API（`api.github.com`）且用户未提供有效 token 时，用服务器 token 提升限流。对文件下载和 Git 操作从不生效。

```
API 请求：
  无用户 token        → Authorization: token <server>
  有用户 token 且有效  → Authorization: token <user>
  有用户 token 但无效  → 重试用 server token + 前端警告

下载请求：
  有用户 token → Authorization: token <user>
  无用户 token → 无 Authorization（匿名）

Git 请求：
  有用户 token → Authorization: token <user>
  无用户 token → 无 Authorization（公开仓库）
```

| | API 请求 | 文件下载 | Git |
|:---|:---|:---|:---|
| 无用户 token | `token <server>` | 无（匿名） | 无（公开仓库） |
| 用户 token 有效 | `token <user>` | `token <user>` | `token <user>` |
| 用户 token 无效 | 重试 → `token <server>` + 前端警告 | 直接 401/403 | 直接 401/403 |

配置方式：`config.toml` → `githubToken` 或环境变量 `GITHUB_TOKEN`。

## Token 白名单

提取到的 token 与白名单比对。命中 → 免限速。未命中或无 token → 受限速。

| Token 状态 | 限速 |
|:---|:---|
| 未传入 | 限速 |
| 传入，不在白名单 | 限速 |
| 传入，在白名单 | 不限速 |

配置方式：`config.toml` → `[tokenWhiteList].tokens` 或 `TOKEN_WHITELIST=ghp_xxx,ghp_yyy`。

## 场景速查

| 场景 | Token 来源 | 生效形式 |
|:---|:---|:---|
| 前端浏览 Releases | `X-GitHub-Token` 头 | `Authorization: token <user>` |
| 新窗口下载文件 | `?token=` 参数 | `Authorization: token <user>` |
| git clone/push 私有仓库 | `Authorization: Basic`（从 URL 提取） | `Authorization: token <user>` |
| 公开仓库 API 请求 | 无用户 token + 服务器 token | `Authorization: token <server>` |
| 用户 token 失效 | 服务器 token 兜底重试 | `Authorization: token <server>` |
| 公开仓库下载 | 无 token | 无 Authorization |

## 常见问题

**前端设了 token，git push 还需要再设吗？**

需要。前端的 token 存在浏览器的 localStorage 中，只有浏览器发出的请求才会携带。git 命令从终端发起，完全读不到浏览器的数据，必须在 remote URL 里单独嵌入 token。两者互不干扰。

**如果 token 已过期或无效，会发生什么？**

代理检测到 GitHub 返回 401/403 后，自动用服务器 token 重试 API 请求。同时响应头标记 `X-Token-Status: invalid`，前端在搜索框下方显示警告"Token 已失效，已自动切换为服务器限流模式"。

**Windows 上 git push 时弹出了凭据输入窗口，兼容吗？**

完全兼容。Git Credential Manager 弹出的窗口中输入 GitHub 用户名和 PAT，git 以 `Authorization: Basic` 发给代理。代理从 Basic Auth 中提取 PAT，后续白名单检查、限速豁免均生效。首次认证后 GCM 缓存凭据，不再弹窗。