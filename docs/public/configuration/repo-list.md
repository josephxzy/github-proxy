# 仓库黑白名单

通过 `REPO_WHITELIST` 和 `REPO_BLACKLIST` 控制可访问的仓库范围。

## 匹配规则

| 格式 | 匹配范围 | 示例 |
|:---|:---|:---|
| `user/repo` | 精确匹配指定仓库 | `josephxzy/github-proxy` |
| `user` | 匹配该用户的所有仓库 | `josephxzy` |
| `user/*` | 匹配该用户的所有仓库（同 `user`） | `josephxzy/*` |
| `prefix*` | 前缀匹配 | `joseph*` |

## 配置方式

环境变量，逗号分隔：

```bash
# 白名单：只允许访问指定仓库
REPO_WHITELIST=josephxzy/github-proxy,torvalds/linux

# 黑名单：禁止访问指定仓库
REPO_BLACKLIST=blocked-user,spam-org/*
```

## 优先级

- 白名单和黑名单同时配置时，白名单优先：配置白名单后黑名单不再生效，白名单即为唯一允许集（命中放行，未命中拒绝）
- 只配置白名单：仅允许白名单中的仓库
- 只配置黑名单：禁止黑名单中的仓库，其余允许
- 都不配置：所有仓库允许

## 注意

- 黑白名单匹配的是代理路径中的仓库标识
- 通配符 `*` 只能用于前缀匹配
- 大小写敏感