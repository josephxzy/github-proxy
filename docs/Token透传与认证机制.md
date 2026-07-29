# Token 透传与认证机制

github-proxy 统一从三个来源提取 token，合并为一份后应用到所有出站请求。

## 1. Token 提取（唯一入口）

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

## 2. Token 应用

提取到的 token 存入 Gin Context（`c.Set("userToken", token)`），然后在下游统一应用：

```go
// proxy_download.go / proxy_api.go
// 对所有出站请求设置 Authorization: token <extracted_token>
req.Header.Set("Authorization", "token "+token)
```

**一个 token，一种格式，所有请求**。不管原来从哪个来源提取的，最终都以 `Authorization: token <value>` 发给 GitHub。

## 3. 服务器 GitHub Token（`GITHUB_TOKEN` / `githubToken`）

**仅用于兜底**：当请求是 GitHub API（`api.github.com`）且用户未提供有效 token 时，用服务器 token 提升限流。对文件下载和 Git 操作**从不生效**。

### 请求类型与 token 行为

```
API 请求：
  无用户 token        → Authorization: token <server>
  有用户 token 有效    → Authorization: token <user>
  有用户 token 无效    → 重试用 server token + 前端警告

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

**关键差异**：

- **API 请求**：服务器 token 兜底。用户 token 失效时自动回退，保证公开仓库的搜索和浏览不受影响。
- **文件下载**：服务器 token 不参与。公开文件无需认证，私有文件必须用户提供有效 token。用户 token 失效时 GitHub 直接拒绝，不会重试——因为如果是公开文件本就不需要 token，如果是私有文件服务器 token 也访问不了。
- **Git 操作**：同上。服务器 token 不参与，认证完全由 git 凭据决定。

配置方式：`config.toml` → `githubToken` 或环境变量 `GITHUB_TOKEN`。

## 4. Token 白名单（`TOKEN_WHITELIST`）

提取到的 token 与白名单比对。命中 → 免限速。未命中或无 token → 受限速。

| Token 状态 | 限速 |
|:---|:---|
| 未传入 | 限速 |
| 传入，不在白名单 | 限速 |
| 传入，在白名单 | 不限速 |

配置方式：`config.toml` → `[tokenWhiteList].tokens` 或环境变量 `TOKEN_WHITELIST=ghp_xxx,ghp_yyy`。

## 5. 场景速查

| 场景 | Token 来源 | 生效形式 | 白名单 |
|:---|:---|:---|:---|
| 前端浏览 Releases | `X-GitHub-Token` 头 | `Authorization: token <user>` | ✅ |
| 新窗口下载文件 | `?token=` 参数 | `Authorization: token <user>` | ✅ |
| git clone/push 私有仓库 | `Authorization: Basic`（从 URL 提取） | `Authorization: token <user>` | ✅ |
| 公开仓库 API 请求 | 无用户 token + 服务器 `githubToken` | `Authorization: token <server>` | — |
| 用户 token 失效 | 服务器 `githubToken` 兜底重试 | `Authorization: token <server>` | — |
| 公开仓库下载 | 无 token，GitHub 允许匿名 | 无 Authorization | — |

## 6. 常见问题

**Q: 前端设了 token，git push 还需要再设吗？**

需要。前端的 token 存在浏览器的 localStorage 中，只有浏览器发出的请求才会携带。git 命令从终端发起，完全读不到浏览器的数据，必须在 remote URL 里单独嵌入 token。

两者互不干扰——一个在浏览器，一个在终端，是两条独立的请求链路。

**Q: 如果我提供的 token 已过期或无效，会发生什么？**

代理检测到 GitHub 返回 401/403 后，会自动用服务器 token 重试 API 请求，保证基本功能不受影响。同时响应头中会标记 `X-Token-Status: invalid`，前端在搜索框下方显示黄色警告"Token 已失效，已自动切换为服务器限流模式"。此时应重新设置有效 Token。

**Q: 为什么 git 凭据被统一为 `token <value>` 格式而不是保留 `Basic` 格式？**

GitHub 的所有端点（API、Git HTTP、下载）都接受 `Authorization: token <value>` 格式。统一为一种格式简化了实现。

**Q: Windows 上 git push 时弹出了凭据输入窗口，兼容吗？**

完全兼容。Windows 的 Git Credential Manager 弹出的窗口中输入 GitHub 用户名和 PAT，git 会以 `Authorization: Basic` 发给代理。代理从 Basic Auth 中提取 PAT，后续流程与前端传入 token 完全一致——白名单检查、限速豁免均生效。

首次成功认证后 GCM 会缓存凭据，后续不再弹窗。
