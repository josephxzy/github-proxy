# Token 认证体系

## 背景

github-proxy 作为反向代理，需要处理三种完全不同来源的 GitHub token：浏览器前端设置的 Personal Access Token、URL 参数中携带的 token、以及 Git 操作中嵌入的 Basic Auth 凭据。三种来源使用场景互斥，但最终都需要统一转换为 `Authorization: token <value>` 格式发给 GitHub。

## 决策

**单一入口，统一格式**。所有 token 提取集中在一个函数 `ExtractToken` 中，按优先级降序检查三个来源，取第一个非空的。下游所有请求统一使用 `Authorization: token <value>` 格式，不管原来源是什么。

## 当前规则

- Token 提取优先级：`X-GitHub-Token` 头 > `?token=` 参数 > `Authorization: Basic` 头。
- 提取后必须从 URL 中删除 `?token=` 参数，不发给 GitHub。
- 服务器 token（`GITHUB_TOKEN`）仅用于 GitHub API 请求的兜底，不对文件下载和 Git 操作生效。
- 用户 token 失效时，API 请求自动重试用服务器 token，同时前端显示警告。
- Token 白名单命中时跳过所有限速（单流限速、global 限速、IP 限流）。
- 前端 token（localStorage）和 Git token（remote URL）互不干扰，各自独立。

## 示例

**推荐做法**：
- 前端设置 token → 浏览器自动带 `X-GitHub-Token` 头 → 访问私有仓库
- 新窗口下载 → URL 带 `?token=ghp_xxx` → 代理提取后删除参数
- Git clone 私有仓库 → URL 嵌入 PAT → 代理从 Basic Auth 提取

**禁止做法**：
- 在服务端日志中打印 token 明文
- 将用户 token 持久化到服务端存储
- 将服务器 token 应用到文件下载请求

## 失败模式

| 症状 | 原因 | 排查 |
|:---|:---|:---|
| 私有仓库 404 | token 未提取或已过期 | 检查 `X-Token-Status` 响应头 |
| 前端搜索受限 | 服务器 token 未设置 | 检查 `GITHUB_TOKEN` 环境变量 |
| Git push 403 | remote URL 未嵌入 token | 检查 `git remote -v` 输出 |
| 前端 token 无效但无警告 | 前端未监听 `X-Token-Status` 头 | 检查浏览器 Network 面板 |

## 相关模块

- `src/handlers/github.go` — `ExtractToken` 函数
- `src/config/config.go` — `GITHUB_TOKEN` 配置
- `src/frontend/` — 前端 token 设置与 localStorage

## 来源文档

- Token 透传与认证：`../design/token.md`