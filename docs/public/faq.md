# 常见问题

## Token 相关

### 应该用哪种 GitHub Token？

GitHub 提供两种 Token，本项目都支持：

- **Classic Token**（推荐）：一个 Token 搞定所有，勾选 `repo` 域即可。配置简单，适合个人开发。
- **Fine-grained Token**：可精确控制到具体仓库和操作权限，安全性更高，但配置步骤较多。

详见 [私有仓库指南](#/docs/private-repo)。

### 前端设了 token，git push 还需要再设吗？

需要。前端的 token 存在浏览器的 localStorage 中，只有浏览器发出的请求才会携带。git 命令从终端发起，完全读不到浏览器的数据，必须在 remote URL 里单独嵌入 token。

### 如果 token 已过期或无效，会发生什么？

代理检测到 GitHub 返回 401/403 后，自动用服务器 token 重试 API 请求。同时响应头标记 `X-Token-Status: invalid`，前端在搜索框下方显示警告。

### Windows 上 git push 时弹出了凭据输入窗口，兼容吗？

完全兼容。Git Credential Manager 弹出的窗口中输入 GitHub 用户名和 PAT，git 以 `Authorization: Basic` 发给代理。首次认证后 GCM 缓存凭据。

## 下载相关

### 下载没有进度条？

部分资源类型（如 Archive）的 GitHub 服务器不提供 Content-Length，无法显示进度条。数据仍然完整可靠。

### 下载速度慢？

检查是否配置了 `DOWNLOAD_RATE` 限速。设为 `0` 取消限速。如果使用 Nginx 反向代理，确保 `gzip off` 和 `proxy_buffering off`。

### 断点续传失败？

确认客户端发送了正确的 `Range` 头。代理透传 Range 到 GitHub，不修改 Content-Length。

## 配置相关

### 如何查看当前配置？

所有配置项以环境变量为准。查看 `src/config.toml` 了解完整配置项和默认值。

### 如何限制某个用户的下载速度？

设置 `DOWNLOAD_RATE` 限制单用户速度（B/s），设置 `GLOBAL_RATE` 限制全局总带宽。

## Git 操作相关

### Push 私有仓库怎么认证？

两种方式任选：

**Token 嵌入 URL**（最省事）：`git remote set-url origin https://ghp_xxx@hub.example.com/user/repo.git`

**弹出认证框**（更安全）：只填代理地址不嵌 Token，Push 时 Git Credential Manager 弹出窗口，输入用户名和 PAT。首次认证后自动缓存，后续不再弹窗。

### 如何不让 Push 走代理中转上传？

如果通过代理 clone 了仓库，但希望 Push 直连 GitHub 而不走代理，有两种方式：

**方式一：设置 pushInsteadOf 例外**

配置 `pushInsteadOf` 让 Push 走直连：

```bash
git config --global url."https://github.com/".pushInsteadOf "https://hub.example.com/"
```

这样 Pull/Fetch 走代理加速下载，Push 直连 GitHub。

**方式二：单独修改 Push URL**

只修改当前仓库的 Push 地址，不影响 Fetch：

```bash
git remote set-url --push origin https://github.com/user/repo.git
```

验证：

```bash
git remote -v
# origin  https://hub.example.com/user/repo.git (fetch)
# origin  https://github.com/user/repo.git (push)
```

> 注意：如果只配置了 `insteadOf`（全局替换），需要先移除再按上述方式设置。`insteadOf` 会同时替换 fetch 和 push 地址。