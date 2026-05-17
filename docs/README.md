# AI-Agent-OS 文档入口

> 状态：文档治理入口  
> 更新时间：2026-05-17

本目录只承载主项目级文档。服务内 README、SDK 参考、部署文档、Prompt 案例和生成工作区文档分别留在对应模块中，不再混到一个入口里。

## 必读入口

- [开源治理 TODO](governance/OPEN_SOURCE_TODO.md)：开源前治理事项、并行窗口分工、验收口径。
- [文档治理规则](governance/DOCUMENT_GOVERNANCE.md)：文档分层、历史文档处置、删除/归档标准。
- [部署分层模型](deployment-layers.md)：生产部署和排障的官方分层心智模型。
- [代码体检与重构记录](codebase-health-and-refactor-notes.md)：代码体量、重构方向和债务记录。
- [产品减法与主链路打深计划](产品减法与主链路打深计划.md)：当前阶段产品范围收敛口径。

## 文档分层

| 层级 | 位置 | 用途 |
| --- | --- | --- |
| 项目入口 | `README.md`、`docs/README.md` | 对外介绍、快速开始、文档导航 |
| 治理文档 | `docs/governance/` | 开源前 TODO、文档治理、发布检查清单 |
| 产品/架构决策 | `docs/` | 当前仍有效的跨模块产品、架构和重构文档 |
| 部署文档 | `deploy/` | 开发、生产、安全、镜像和配置说明 |
| 模块文档 | `core/**/README.md`、`pkg/**/README.md`、`sdk/**/README.md`、`web/**/README.md` | 模块内维护说明，不作为项目总入口 |
| Prompt/示例文档 | `core/agent-server/prompt/**`、`core/app-server/system-seed/**` | Agent 运行时素材或内置示例，按数据资产治理 |
| 第三方/嵌入项目 | `local/hermes-agent/**` | 独立边界，不纳入主项目文档治理，除非决定公开打包 |
| 生成工作区 | `namespace/**`、`tests/e2e/.mcp-output/**`、`temp/**` | 原则上不进入开源发布包 |

## 当前原则

1. 根 README 面向第一次打开仓库的人，不再承载所有历史设计细节。
2. `docs/` 只保留仍指导当前项目治理、架构和产品范围的文档。
3. 过期方案不直接删除，先进入处置清单，确认无引用、无决策价值后再删或归档。
4. 每个并行窗口新增文档时，必须先判断它属于“项目治理、模块说明、部署说明、SDK 参考、示例数据”哪一类。

