# Token 白名单

将 Token 加入白名单后，使用该 Token 的请求将跳过下载限速、全局带宽限速、API 限速和 IP 限流。

## 配置方式

```bash
TOKEN_WHITELIST=ghp_xxx,ghp_yyy
```

或通过 `config.toml`：

```toml
[tokenWhiteList]
tokens = ["ghp_xxx", "ghp_yyy"]
```

## 白名单效果

| Token 状态 | 下载限速 | 全局带宽限速 | API 限速 | IP 限流 |
|:---|:---|:---|:---|:---|
| 未传入 | 限速 | 限速 | 限速 | 限流 |
| 传入，不在白名单 | 限速 | 限速 | 限速 | 限流 |
| 传入，在白名单 | 不限速 | 不限速 | 不限速 | 不限流 |

## 典型场景

- 给可信用户/团队成员分配白名单 Token，免限速下载
- 给 CI/CD 使用的 Token 加入白名单，避免被限速影响构建
- 给自己使用的 Token 加入白名单，获得最佳体验，API 请求也不受限

## 共享配额计费规则

代理的共享 API 池只统计**使用站内 Token 的请求**：

- 用户自带有效 Token：请求走用户自己的 GitHub 配额，**不占用**共享池
- 用户 Token 失效：代理自动改用站内 Token 重试，**该次重试占用一次**共享配额
- 未携带 Token：请求使用站内 Token（或匿名），**占用**共享配额
- 白名单 Token：全程豁免，不占用共享池

## 注意

- Token 必须是完整的 GitHub Personal Access Token
- Token 白名单只能匹配通过 `X-GitHub-Token` 头、`?token=` 参数或 `Authorization: Basic` 提取的 Token
- 白名单 Token 不限制使用次数，请妥善保管