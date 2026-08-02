# Github Proxy

轻量级 GitHub 资源加速反向代理。项目为**站长**提供安全、快速的 GitHub 加速体验，同时为**普通用户**提供最基本的文件下载服务。

内置 Vue 3 前端界面，支持仓库搜索、Release 浏览、README 预览与 Git 加速。

## 功能一览

- 全类型加速：Raw / Blob / Archive / Release / Gist 全覆盖
- Git 加速：Clone / Fetch / Push，短路径格式支持
- 私有仓库：Token 透传认证，白名单免限速
- 稳定性：双层限速 + IP 限流 + 水位线反压，下载不断流

## 快速开始

下载 [Releases](https://github.com/josephxzy/github-proxy/releases) 中的二进制，设置环境变量后直接运行，默认监听 `0.0.0.0:5000`。

```bash
# 设置 GitHub Token（可选，提升 API 限流）
export GITHUB_TOKEN=ghp_xxx

# 启动服务
./github-proxy
```

## 文档

完整的部署、配置、使用指南与设计文档，请访问文档站：**https://doc.proxy.xzyuse.site**

## 致谢

- 后台代码参考 [hubproxy](https://github.com/sky22333/hubproxy)
- 文档站样式借鉴自 [AI-Novel-Writing-Assistant](https://github.com/ExplosiveCoderflome/AI-Novel-Writing-Assistant)

## License

MIT
