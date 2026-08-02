# Git 加速

## Clone

将远程 URL 中的 `github.com` 替换为代理地址：

```bash
# 公开仓库
git clone https://hub.example.com/user/repo.git

# 私有仓库（嵌入 PAT）
git clone https://ghp_xxx@hub.example.com/user/private-repo.git
```

## Fetch / Pull

修改已有仓库的 remote URL：

```bash
git remote set-url origin https://hub.example.com/user/repo.git
git fetch origin
```

## Push

Push 到私有仓库需要认证，有两种方式可选。

### 方式一：Token 嵌入 URL（最省事）

直接在 remote URL 中嵌入 PAT，一行命令搞定：

```bash
# 只嵌 Token（Git 自动将其作为用户名发送）
git remote set-url origin https://ghp_xxx@hub.example.com/user/repo.git

# 或者带用户名（Git 发送 Basic Auth: user:ghp_xxx）
git remote set-url origin https://git:ghp_xxx@hub.example.com/user/repo.git
```

两种写法都有效。代理从 Basic Auth 的密码字段提取 Token，为空时回退到用户名字段。用户名可以是任意值（甚至省略），GitHub 认证只校验 Token。

> 这和 GitHub 原生支持的 `https://user:token@github.com/repo.git` 格式一致——标准 HTTP Basic Auth，代理只是把 `github.com` 换成了代理地址。

### 方式二：弹出认证框后输入（Windows）

如果不想把 Token 明文写在 URL 里，可以只填代理地址，让 Git 在 Push 时弹出认证窗口：

```bash
git remote set-url origin https://hub.example.com/user/repo.git
git push origin main
# → Git Credential Manager 弹出窗口
# → 输入 GitHub 用户名 + PAT
# → 首次认证后 GCM 自动缓存，后续不再弹窗
```

这种方式 Token 不暴露在命令行历史和 remote URL 中，安全性更高。macOS / Linux 上由系统钥匙串接管，流程类似。

## pushInsteadOf 配置

配置 Git 自动将 GitHub URL 替换为代理地址，无需手动修改每个仓库的 remote：

```bash
git config --global url."https://hub.example.com/".insteadOf "https://github.com/"
```

之后所有 `git clone https://github.com/...` 都会自动走代理。

## Windows 凭据管理

Windows 上首次 git push 时，Git Credential Manager 会弹出凭据输入窗口。输入 GitHub 用户名和 PAT 即可。首次认证后 GCM 缓存凭据，不再弹窗。

## 注意事项

- 前端的 Token（localStorage）对 Git 操作无效，终端无法读取浏览器数据
- Git 操作必须在 remote URL 中单独嵌入 Token
- 代理会自动从 `Authorization: Basic` 中提取 PAT 并转换为 `token` 格式