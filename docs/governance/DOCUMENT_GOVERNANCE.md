# 文档治理规则与历史文档处置清单

> 状态：执行口径  
> 更新时间：2026-05-17  
> 目标：把文档从“历史堆积”治理成“入口清晰、职责明确、可开源维护”的体系。

## 1. 现状判断

当前仓库文档有四类混在一起：

1. **主项目文档**：根 `README.md`、`docs/`、`deploy/`、`web/README.md`、`sdk/agent-app/**`。
2. **模块内部文档**：`core/**/README.md`、`pkg/**/README.md`、`web/src/architecture/**/README.md`。
3. **运行时素材/内置示例**：`core/agent-server/prompt/**`、`core/app-server/system-seed/**`。
4. **非主项目或生成数据**：`local/hermes-agent/**`、`namespace/**`、`tests/e2e/.mcp-output/**`、`temp/**`。

最大问题不是文档少，而是：

- 入口不唯一，外部读者不知道先看哪个。
- 历史方案和当前事实混在一起。
- 产品愿景、架构设计、开发说明、Prompt 素材、生成工作区没有边界。
- 一些文档适合内部决策，不适合开源首页暴露。

## 2. 文档分层规则

### 2.1 根目录

根目录只放开源项目门面文件：

- `README.md`
- `LICENSE`
- `THIRD_PARTY_LICENSES.md`
- `CONTRIBUTING.md`
- `SECURITY.md`
- `CODE_OF_CONDUCT.md`

根目录历史设计文档原则上迁入 `docs/archive/` 或删除。

### 2.2 docs

`docs/` 放跨模块、当前有效的主项目文档：

- 产品范围和路线
- 架构决策
- 代码健康和重构
- 开源治理
- 部署模型的总说明

不放模块私有细节，不放一次性会议纪要，不放生成工作区输出。

### 2.3 deploy

`deploy/` 只放部署、配置、安全和运维文档。  
如果内容是产品说明或代码架构，迁回 `docs/`。

### 2.4 core/pkg/sdk/web

模块内文档只解释本模块：

- 如何启动
- 如何测试
- 关键接口
- 模块内部架构
- 与外部模块的契约

模块文档不能充当项目总入口。

### 2.5 local、namespace、temp、tests output

这些目录默认不是主项目公开文档来源：

- `local/hermes-agent/**`：疑似嵌入/外部项目，必须先确认授权和发布边界。
- `namespace/**`：用户应用/系统种子/工作区数据，开源前必须单独审查。
- `tests/e2e/.mcp-output/**`：测试输出，不应进入发布文档。
- `temp/**`：临时材料，不应进入发布文档。

## 3. 历史文档处置规则

每份历史文档只允许进入四种状态：

| 状态 | 含义 | 处理 |
| --- | --- | --- |
| Keep | 当前仍指导项目 | 保留原位或纳入入口 |
| Move | 内容有价值但位置不对 | 迁到正确目录 |
| Archive | 有历史决策价值但不再指导当前实现 | 移入 `docs/archive/`，顶部标注“历史归档” |
| Delete | 重复、过期、无引用、无决策价值 | 删除 |

删除前检查：

1. `rg "文件名或标题"` 确认没有有效引用。
2. 判断是否包含尚未迁移的决策。
3. 如果是 SDK/Prompt/部署相关文档，确认对应模块负责人窗口已经接手。

## 4. 初始处置清单

### 4.1 建议保留并纳入入口

| 文件 | 状态 | 理由 |
| --- | --- | --- |
| `docs/deployment-layers.md` | Keep | 部署和排障心智模型，当前有效 |
| `docs/codebase-health-and-refactor-notes.md` | Keep | 代码治理和债务记录，当前有效 |
| `docs/产品减法与主链路打深计划.md` | Keep | 当前 MVP 范围口径，开源前关键 |
| `web/src/architecture/README.md` | Keep | 前端架构层说明 |
| `sdk/agent-app/runtime/python/README.md` | Keep | SDK 运行时说明 |

### 4.2 建议合并或归档

| 文件 | 建议 | 理由 |
| --- | --- | --- |
| `Agent工作流流程图.md` | 已归档 | 移至 `docs/archive/Agent工作流流程图.md`，当前入口以 `docs/工作台最新设计.md` 为准 |
| `项目介绍.md` | 已归档 | 移至 `docs/archive/项目介绍.md`，作为 README 重写素材 |
| `项目架构.md` | 已归档 | 移至 `docs/archive/项目架构.md`，作为架构入口重写素材 |
| `docs/工作台角色状态机重构方案.md` | Keep/Review | 已恢复原位，避免当前工作台设计入口断链；后续可单独判断是否并入 `docs/工作台最新设计.md` |
| `docs/工作台单Dev模式与角色切换架构设计.md` | Keep/Review | 已恢复原位，避免当前工作台设计入口断链；后续可单独判断是否并入 `docs/工作台最新设计.md` |
| `docs/工作台最新设计.md` | 已保留 | 作为唯一工作台设计入口，并已纳入 `docs/README.md` |
| `web/ARCHITECTURE_ANALYSIS.md` | 已归档 | 移至 `docs/archive/web-ARCHITECTURE_ANALYSIS.md`，当前入口以 `web/src/architecture/README.md` 为准 |
| `web/THEME_COMPARISON_REPORT.md`、`web/THEME_FIX_SUMMARY.md`、`web/COLOR_ANALYSIS.md` | 已归档 | 移至 `docs/archive/`，不再作为开源入口 |
| `web/FORM_TABLE_IMPLEMENTATION.md`、`web/UPLOAD_PROGRESS.md` | Keep/Review | 已恢复原位；后续如仍有效再合并到模块文档 |
| `web/todos.md` | 已归档 | 移至 `docs/archive/web-todos.md`，后续治理统一以 `docs/governance/OPEN_SOURCE_TODO.md` 为准 |

### 4.3 暂不处理，先确认边界

| 路径 | 原因 |
| --- | --- |
| `local/hermes-agent/**` | 文档量巨大且像独立项目，需确认是否随主项目开源 |
| `namespace/**` | 用户/系统工作区数据，需敏感信息和发布边界审查 |
| `core/agent-server/prompt/system/prompt/case_catalog/**` | Agent 示例资产，不能按普通文档删除 |
| `core/app-server/system-seed/**` | 系统种子数据，需产品和安全窗口确认 |

## 5. 新增文档要求

每个新文档顶部必须包含：

```md
> 状态：草案/执行口径/历史归档
> 更新时间：YYYY-MM-DD
> 负责人窗口：事项 N / 分支名
```

每个文档必须回答：

- 面向谁
- 解决什么问题
- 是否当前有效
- 被哪个入口引用

## 6. 并行窗口协作规则

1. 每个窗口只处理一个治理事项。
2. 每个窗口优先在 `docs/governance/OPEN_SOURCE_TODO.md` 更新状态。
3. 删除历史文档必须在 PR/提交说明中列出理由。
4. 如果两个窗口都要动根 `README.md`，先由文档治理窗口合并，避免互相覆盖。
5. 模块文档由模块窗口改，治理窗口只改入口和规则。
