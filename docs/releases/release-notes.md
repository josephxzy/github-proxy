# Release Notes

完整的用户可见版本更新说明与发布历史。

## v1.3.4

- 修复：git clone / push 白名单豁免失效（Web 前端豁免正常但 git 场景被限速）
  - 根因一：git 客户端首次请求从不携带凭据（即使 URL 内嵌 token），代理对公共仓库透传 200 后 git 永不发送 token，白名单无法命中。修复：配置了 Token 白名单时，对 git 智能 HTTP 端点（`info/refs`、`info/lfs`、`git-upload-pack`、`git-receive-pack`）的无凭据请求返回 401 认证挑战，git 自动携带 URL 内嵌 Token 重试
  - 根因二：GitHub 的 git 端点只接受 Basic 认证，`Authorization: token` 头一律 401。修复：git 请求统一以 `Basic x-access-token:<PAT>` 格式转发，API 与文件下载仍使用 `token` 头
- 测试：新增 git 401 挑战链路、git 端点识别（含 Git LFS、下载文件名误伤排除）、上游认证格式区分用例

## v1.3.3

- 代码质量重构：清理死代码与重复实现（删除未使用的 `ProxyService` / `DownloadService` / `APIService` 实例层），下载与 API 两条路径统一使用公共上游请求构造
- 修复：单用户限速器首次写入不等待的时序偏差；IP 限流器禁用实例的 nil map 隐患
- 目录结构重构：Go 模块根移至仓库根（`cmd/github-proxy` + `internal/*` 标准布局），前端移至 `web/`
- 测试：新增 config、ratelimit、network、脚本处理、API 限流、IP 限流、静态服务单元测试，以及 mock GitHub 全链路集成测试与 git clone 白名单豁免验证（共 100+ 用例）
- CI：新增 `go vet` + `go test -race` 流水线
- 文档：新增代码分层约定、凭据刷新排查、文档站内容更新

## v1.3.2

- 前端隐藏草稿（draft）Release 的下载资源
- 修复短链接 git clone 的 `git-upload-pack` 路径匹配（`git-` 后不带斜杠）

## v1.3.1

- 修复短链接 git clone：`git-` 后不带斜杠的智能 HTTP 端点（`/git-upload-pack`、`/git-receive-pack`）不再被 403 拒绝

## v1.3.0

- 移除 fast mode 残留代码
- 修复限速器暂停/恢复时的突发（burst）
- Token 白名单（命中则豁免限速与 IP 限流）
- 双层限速：单流漏桶 + 全局共享令牌桶
- 水位线反压：真环形缓冲区
- 前端界面美化与移动端适配
- 构建脚本修复（锁定文件处理）

## v1.2.4

- 回滚通用代理，仅代理 GitHub 域名链接

## v1.2.3

- README 中外部域名图片（如 shields.io）经代理加载，解决 SSL 错误

## v1.2.2

- README 中相对路径图片经代理加载，修复 403 错误

## v1.2.1

- 前端移动端适配：项目列表按钮、排序栏、README 弹窗

## v1.2.0

- IP 三层限流体系
- 固定窗口计数器算法
- IPv6 /64 归一化
- 前端路径排除

## v1.1.0

- 双层限速：单流漏桶 + global 共享令牌桶
- 水位线反压：真环形缓冲区
- 断点续传：Range 预检 + 透传
- 脚本 URL 替换

## v1.0.0

- 初始发布
- GitHub 资源加速（Raw / Blob / Archive / Release / Gist）
- 仓库搜索
- Release 浏览
- README 预览
- Git 加速
- 私有仓库支持
- Token 三源提取
- Vue 3 前端界面