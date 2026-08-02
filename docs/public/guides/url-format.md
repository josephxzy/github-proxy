# URL 格式

Github Proxy 支持两种 URL 格式，效果完全等价。

## 完整 URL 格式

将完整的 GitHub URL 附加在代理地址之后：

```
https://hub.example.com/https://github.com/user/repo/releases/download/v1.0/file.zip
```

## 短路径格式

使用 `user/repo` 路径格式：

```
https://hub.example.com/user/repo/releases/download/v1.0/file.zip
```

## 支持的资源类型

| 类型 | 完整 URL 示例 | 短路径 |
|:---|:---|:---|
| Release | `/https://github.com/user/repo/releases/download/v1.0/file.zip` | `/user/repo/releases/download/v1.0/file.zip` |
| Archive | `/https://github.com/user/repo/archive/refs/heads/main.zip` | `/user/repo/archive/refs/heads/main.zip` |
| Blob | `/https://github.com/user/repo/blob/main/file.go` | `/user/repo/blob/main/file.go` |
| Raw | `/https://raw.githubusercontent.com/user/repo/main/file.go` | `/user/repo/raw/main/file.go` |
| Git | — | `/user/repo.git` |

## 前端 URL

访问代理根路径即可使用 Web 前端界面：

```
https://hub.example.com/
```

前端支持仓库搜索、Release 浏览、README 预览、Token 设置等功能。

## Token 传递

在 URL 中附加 `?token=` 参数传递 Token（新窗口下载、分享链接场景）：

```
https://hub.example.com/user/repo/releases/download/v1.0/file.zip?token=ghp_xxx
```

Token 参数会被代理提取后从 URL 中删除，不会发送给 GitHub。

## Git 操作中的认证

Git 操作使用 HTTP Basic Auth 传递凭据，不支持在完整 URL 路径中嵌入 `user:token`。

**正确写法**（凭据在代理主机之前）：

```bash
# 短路径格式
git clone https://ghp_xxx@hub.example.com/user/repo.git

# 带用户名的完整凭据
git clone https://git:ghp_xxx@hub.example.com/user/repo.git
```

**错误写法**（凭据在路径中，Git 不会解析）：

```bash
# 这种写法 Token 不会被提取，认证失败
git clone https://hub.example.com/https://ghp_xxx@github.com/user/repo.git
```

> 原因：Git 只解析紧跟在 `https://` 之后、`@` 之前的 `user:password` 部分。完整 URL 格式中 `user:token@github.com` 在路径段里，Git 不会将其作为凭据发送。