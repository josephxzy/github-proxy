# 仓库搜索

## 搜索入口

访问代理前端页面，在搜索框中输入关键词，按回车搜索。

## 搜索语法

| 限定符 | 说明 | 示例 |
|:---|:---|:---|
| `user:` | 按用户名搜索 | `user:josephxzy` |
| `repo:` | 按仓库名搜索 | `repo:github-proxy` |
| `desc:` | 搜索仓库描述 | `desc:proxy` |
| `readme:` | 搜索 README 内容 | `readme:install` |

限定符可以组合使用：

```
user:josephxzy desc:proxy
```

## 结果排序

搜索结果支持按以下方式排序：

- **Stars** — 按 Star 数量降序
- **Forks** — 按 Fork 数量降序
- **Updated** — 按最近更新时间降序

## 结果操作

每个搜索结果卡片提供以下操作：

- **ZIP 下载** — 下载仓库最新源码压缩包
- **Releases** — 查看仓库的 Release 列表
- **ReadMe** — 查看仓库的 README 文件

## 搜索限制

- 搜索结果来自 GitHub API，受 API 限流影响
- 设置 `GITHUB_TOKEN` 可提升 GitHub API 限额（普通 REST API 从 60 次/小时提升到 5000 次/小时，搜索 API 从 10 次/分钟提升到 30 次/分钟）
- 每个搜索请求消耗 1 次 API 限额
- 可通过 `API_SEARCH_HOURLY` 配置搜索 API 每小时限额