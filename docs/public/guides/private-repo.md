# 私有仓库

## 前端设置 Token

1. 访问代理前端页面
2. 点击右上角开关，切换到"私有仓库"模式
3. 输入 GitHub Personal Access Token（需要 `repo` 权限）
4. Token 存储在浏览器 localStorage 中

## 生成 Token

GitHub 提供两种 Personal Access Token，本项目都支持。

### Classic Token（推荐，最省事）

1. 访问 [GitHub Settings > Developer settings > Personal access tokens > Tokens (classic)](https://github.com/settings/tokens)
2. 点击 "Generate new token (classic)"
3. 勾选 `repo` 权限（完整仓库访问）
4. 生成并复制 Token

Classic Token 一个就够了，无需额外配置，直接作为认证凭据使用。

### Fine-grained Token

1. 访问 [GitHub Settings > Developer settings > Personal access tokens > Fine-grained tokens](https://github.com/settings/tokens?type=beta)
2. 点击 "Generate new token"
3. 选择资源所有者（个人账号或组织）
4. 选择 "Only select repositories" 并指定仓库
5. 在 Repository permissions 中设置 `Contents: Read and write`
6. 生成并复制 Token

Fine-grained Token 可以精确控制到具体仓库和权限，安全性更高，但配置步骤较多。适合对安全有严格要求的场景。

### 两种 Token 对比

| | Classic | Fine-grained |
|:---|:---|:---|
| 权限粒度 | 按域（repo/workflow/admin） | 按仓库 + 按操作 |
| 配置复杂度 | 低，勾选一个域即可 | 高，需逐一指定仓库和权限 |
| 有效期 | 可选永久 | 最长 1 年，必须设置过期 |
| 使用方式 | 直接作为认证凭据 | 直接作为认证凭据 |
| 推荐场景 | 个人开发、快速上手 | 组织管理、精确权限控制 |

## Token 使用方式

| 场景 | Token 传递方式 |
|:---|:---|
| 前端浏览 | 通过 `X-GitHub-Token` 请求头自动携带 |
| 新窗口下载 | 通过 `?token=` URL 参数 |
| Git 操作 | 在 remote URL 中嵌入 PAT |

## 注意事项

- Token 只存储在浏览器本地，不会上传到服务器
- 前端 Token 和 Git Token 互不干扰，需要分别设置
- 服务器 Token（`GITHUB_TOKEN`）仅用于 API 兜底，不对文件下载和 Git 操作生效
- Token 过期后，API 请求自动回退到服务器 Token，前端显示警告