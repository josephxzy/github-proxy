# Docs 管理约定

`docs/` 用来承接根目录之外的设计文档、阶段检查点、模块计划和历史归档，避免方案文档继续散落在仓库根目录。

## 根目录保留规则

根目录是 Go 模块根（`go.mod`），源码按标准布局组织，只保留下面几类：

- 项目入口与对外说明：`README.md`
- Go 模块与构建工具链：`go.mod`、`go.sum`、`build.ps1`、`build.sh`
- Go 源码（标准布局）：`cmd/github-proxy/`（main 包）、`internal/`（全部私有包）
- 前端项目：`web/`（Vue 3 源码，产物输出到 `cmd/github-proxy/public/` 供 embed 打包）
- 文档站：`site/`（React + Vite，内容源在 `docs/public/`）
- 运行时配置：`config.toml`、`.gitignore`

其余设计稿、阶段总结、模块计划、历史规格，统一进入 `docs/` 对应子目录。

## 目录划分

### `docs/checkpoints`

用于记录阶段性检查点、架构迁移里程碑、进度审计和对照说明。

- [代码质量重构检查点](./checkpoints/code-quality-refactor.md)：死代码清理、config/handlers 重构与测试补充

### `docs/plans`

用于放仍有执行价值的模块计划、工作拆解和产品推进方案。

### `docs/design`

用于放系统设计、模块接口、产品机制和领域建模说明。

- [Token 透传与认证](./design/token.md)
- [下载与断点续传](./design/download.md)
- [限速与稳定性设计](./design/rate-limit.md)
- [IP 限流设计](./design/ip-limit.md)

### `docs/architecture`

承接横切架构说明与工程约定（不改变根目录对外入口）。

- [构建与测试](./architecture/testing.md)：Go 项目的构建与测试运行方式。

### `docs/wiki`

用于沉淀长期项目知识，帮助未来开发者和 AI Agent 理解关键架构决策、工作流边界、运行协议、调试经验和产品设计依据。

Wiki 不替代计划、检查点或发布说明：

- `docs/wiki` 记录稳定规则和原因。
- `docs/plans` 记录仍有执行价值的方案和工作拆解。
- `docs/checkpoints` 记录阶段性状态、迁移里程碑和审计对照。
- `docs/design` 记录模块设计、领域建模和产品机制。
- `docs/releases` 记录用户可见变化。

- [Wiki Index](./wiki/README.md)
- [Wiki Entry Template](./wiki/entry-template.md)
- [Token 认证体系](./design/token.md)
- [下载管道与水位线反压](./design/download.md)
- [双层限速设计](./design/rate-limit.md)
- [IP 三层限流体系](./design/ip-limit.md)

### `docs/releases`

用于放完整的用户可见版本更新说明与发布历史；根 `README.md` 只保留最新一次更新，本目录负责承接完整历史。

### `docs/archive`

用于放历史初始化方案、已不再作为主执行依据但仍需要保留的资料。

## 新文档命名规则

- 统一使用小写英文文件名，单词之间用 `-` 连接。
- 计划类文档优先放到 `docs/plans/`。
- 架构调整、进度校验、迁移检查点优先放到 `docs/checkpoints/`。
- 模块设计、数据模型、交互机制优先放到 `docs/design/`。
- 长期架构规则、工作流边界、调试经验和产品设计依据优先放到 `docs/wiki/`。
- 用户可见版本更新历史优先放到 `docs/releases/`。
- 已废弃、乱码、明显被当前发布事实取代但需要留档的方案放到 `docs/archive/outdated/`。

## 维护约束

- 新增文档时，先判断是否真的需要留在根目录；默认答案应当是"不需要"。
- 新增或修改核心工作流、限速、Token、下载管道或重要调试结论时，先判断是否产生稳定 Wiki 价值。
- Wiki 页面应解释长期规则和原因，不写成文件修改列表、临时 TODO 或 release notes 复制品。
- 文档迁移后，如根 `README.md` 或其他入口文档里有引用，应同步更新路径。
- 根 `README.md` 的更新说明只保留最新一次；完整历史统一维护在 `docs/releases/release-notes.md`。