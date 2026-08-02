# 项目介绍

Github Proxy 是专注于 GitHub 资源加速的轻量级反向代理工具，内置 Vue 3 前端界面，支持仓库搜索、Release 浏览、README 预览和 Git 加速。

与通常的文件代理不同，本项目不仅提供 GitHub 文件的加速下载，还支持**中转上传**：通过本代理 clone 的仓库，Push 操作会经由代理服务器中转上传到 GitHub，从而获得更好的上传体验。无需单独配置上传代理，一次 clone 即可同时享受下载和上传的双向加速。

## 它解决什么问题

在国内访问 GitHub 时，经常会遇到下载速度慢、Git 操作卡顿的问题。Github Proxy 作为反向代理，将所有 GitHub 请求通过代理服务器转发，利用服务器的带宽优势加速访问。无论是 Pull 还是 Push，数据流都经过代理中转，双向受益。

## 核心特性

- **全类型加速** — Raw / Blob / Archive / Release / Gist 全覆盖
- **仓库搜索** — 支持排序、筛选，一键查看 ReadMe
- **Release 浏览** — 在线查看版本列表，下载资源
- **Git 加速** — Clone / Fetch / Push，短路径支持
- **私有仓库** — 前端设置 Token，代理透传认证
- **进度条** — Range 预检 + Content-Length，断点续传
- **双层限速** — 单流 + global，白名单免限速
- **IP 限流** — 每 IP 每小时请求上限，防滥用
- **智能反压** — 水位线缓冲区，TCP 窗口永不归零
- **Token 自动重试** — 用户 token 失效时回退服务器 token
- **脚本替换** — `.sh` / `.ps1` 内 GitHub 链接自动代理

## 快速体验

```bash
GITHUB_TOKEN=ghp_xxx ./github-proxy
```

访问 `http://localhost:5000` 即可使用。

## 适用场景

- 团队内部部署 GitHub 加速代理
- 个人开发者加速 GitHub 下载和 Git 操作
- CI/CD 环境中加速 GitHub 资源获取
- 配合 Nginx 提供公网 GitHub 加速服务