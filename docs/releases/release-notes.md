# Release Notes

完整的用户可见版本更新说明与发布历史。

## v1.3.7

- 修复：归档下载（master zip / archive / zipball）上游断连后下载失败
  - 根因：归档最终由 codeload.github.com 服务，而 codeload 忽略 Range 请求头（一律返回 200 全量）；上游断连时自动重连要求 206 才能按偏移续传，codeload 给不了，只能放弃 → 下载失败
  - 修复：归档请求启用 skip-resume——断连后整包重拉、跳过已发字节，输出字节流保持连续，下载不中断（归档按 commit ref 不可变，跳过安全）
  - 附带：归档请求跳过无意义的 Range 预检（codeload 忽略 Range，预检拿不到 Content-Range）
- 说明：归档/ git 下载的慢速（上游 codeload / github.com 拥塞）属网络路径问题，代理无法加速；本版本保证归档下载断流后自动续传直至完成

## v1.3.6

- 修复：git clone / push 断流（`fetch-pack: unexpected disconnect` / `early EOF`）
  - 根因：git 智能 HTTP 流（`info/refs`、`git-upload-pack` 等）此前走水位线缓冲，消费端（git 客户端 / 限速）慢时上游 GitHub 连接长时间空闲，被空闲超时掐断；`git-upload-pack` 为 POST 流式响应、不支持 Range 重连，断连即失败
  - 修复：git 流改为直连 pipe 透传，上游流量与客户端消费同步，连接始终有数据流动；git 请求同时跳过 Range 预检与重连包装（无意义且浪费上游请求），限速保持生效
- 修复：不限速（`downloadBytesPerSec=0`）仍卡在约 300KB/s
  - 根因：服务端对 accept 的客户端连接不做套接字调优，Windows 默认 `SO_SNDBUF` 仅 64KB；高延迟链路（跨境 RTT 200ms+）下吞吐被发送窗口卡死：64KB/200ms ≈ 300KB/s
  - 修复：连接建立时按平台调优——Windows 放大下游发送缓冲区至 4MB 并禁用 Nagle（吞吐上限提升至 4MB/RTT）；Linux 依赖内核 `tcp_wmem` 发送缓冲自动调优（手动设置会被 `net.core.wmem_max` 默认 208KB 钳制并关闭自动调优，反而劣化吞吐），仅禁用 Nagle
- 修复：全局限速器单例把首次调用时的速率永久固化，配置变化后仍按旧速率限速（测试/热加载场景下未认证下载被悄悄卡速）
- 测试：新增 socket 调优、全局限速器配置联动、git 直连透传（无预检 / chunked 完整 / 限速保留）用例
- 文档：新增故障模式「高延迟窗口瓶颈」「限速器单例污染」

## v1.3.5

- 修复：Release 列表页显示错误的版本总数与页数（如实际 91~99 个版本却显示"共 100 个版本，10 页"）
  - 根因一：GitHub 分页 `Link` 头解析正则会把 `per_page=10` 的值误当成 `page` 参数，导致页数恒显示为 10、总数恒显示为 100（版本数 > 10 即触发）。修复：提取 `rel="last"` 的完整 URL 后按 `page` 查询参数精确解析，与参数顺序无关
  - 根因二：非最后一页时版本总数按「最后一页满页」估算，最后一页不满时总数偏大。修复：首次加载时以 `per_page=1` 探测精确总数（此时 `rel="last"` 页码即版本总数），翻到最后一页时精确计算并缓存复用
- 修复：切换仓库时进行中的总数探测可能把旧仓库的结果写入新仓库视图

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