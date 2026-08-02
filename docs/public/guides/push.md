# 中转上传

通过 Github Proxy clone 的仓库，Push 操作会自动经由代理服务器中转上传到 GitHub，无需额外配置。

## 原理

Clone 时仓库的 remote URL 指向代理地址：

```bash
git clone https://hub.example.com/user/repo.git
```

此时 `git remote -v` 显示：

```
origin  https://hub.example.com/user/repo.git (fetch)
origin  https://hub.example.com/user/repo.git (push)
```

后续所有 `git push`、`git pull`、`git fetch` 操作都会经过代理服务器中转。代理将请求转发到 GitHub，GitHub 的响应再经由代理返回给客户端。上传和下载使用同一条代理链路。

## 使用方式

### 公开仓库

直接使用代理地址 clone 即可，后续 Push 自动走代理：

```bash
git clone https://hub.example.com/user/public-repo.git
cd public-repo
# 正常开发...
git add .
git commit -m "update"
git push origin main
```

### 私有仓库

Push 到私有仓库需要认证，有两种方式。

**方式一：Token 嵌入 URL（最省事）**

直接在 remote URL 中嵌入 PAT，一行命令搞定：

```bash
# 只嵌 Token（Git 自动将其作为用户名发送）
git clone https://ghp_xxx@hub.example.com/user/private-repo.git

# 或者带用户名（Git 发送 Basic Auth: user:ghp_xxx）
git clone https://git:ghp_xxx@hub.example.com/user/private-repo.git
```

两种写法都有效。代理从 Basic Auth 的密码字段提取 Token，为空时回退到用户名字段。适合个人开发环境。

如果已 clone 了公开仓库、后续需要 Push 到私有仓库，只需修改 remote URL：

```bash
git remote set-url origin https://ghp_xxx@hub.example.com/user/private-repo.git
git push origin main
```

**方式二：弹出认证框后输入**

不想把 Token 明文写在 URL 里，可以只填代理地址，让 Git 在 Push 时弹出认证窗口：

```bash
git remote set-url origin https://hub.example.com/user/private-repo.git
git push origin main
# → Git Credential Manager 弹出窗口
# → 输入 GitHub 用户名 + PAT
# → 首次认证后 GCM 自动缓存，后续不再弹窗
```

这种方式 Token 不暴露在命令行历史和 remote URL 中，安全性更高。macOS / Linux 上由系统钥匙串接管，流程类似。

## 全局配置 pushInsteadOf

如果希望所有 Git 操作都自动走代理（包括 clone、push），可以配置 Git 全局替换：

```bash
git config --global url."https://hub.example.com/".insteadOf "https://github.com/"
```

之后所有 `git clone https://github.com/...` 都会自动替换为代理地址，Push 同样受益。

## 上传体验

| 方式 | 下载 | 上传 |
|:---|:---|:---|
| 直连 GitHub | 可能较慢 | 可能较慢 |
| 通过代理 Clone | 加速 | 加速 |
| 只配 pullInsteadOf | 加速 | 直连 |

> 推荐使用 `insteadOf` 而非 `pushInsteadOf`，让上传和下载都走代理，获得完整的双向加速体验。

## 注意事项

- 代理服务器需要有足够的带宽支撑上传流量
- 大文件上传耗时较长，建议配合 `GLOBAL_RATE` 控制上传带宽、避免影响其他用户的下载体验
- 私有仓库 Push 必须在 remote URL 中嵌入 PAT，前端 localStorage 中的 Token 对 Git 操作无效