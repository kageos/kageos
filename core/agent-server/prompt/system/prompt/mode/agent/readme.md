# Agent 模式

Agent 模式是工作台的综合模式：同一会话内既能创建/修改项目，也能操作已有表单、表格、图表和定时任务。

## 文件与用途

| 文件 | 用途 | 何时、怎么用 |
|------|------|--------------|
| **config.json** | 模式配置：name、description、tool_names、引用的 md 文件名 | 代码 init 时 `loadModeProvider("agent")` 读取；`ToolNames()` 决定本模式可用工具。 |
| **system_prompt.md** | Agent 模式的系统级合同：skill 路由、写入/执行边界 | 拼入发给模型的 system；详细 SOP 通过 `read_skill` 读取，长文档按 skill 的 `required_docs` 读取。 |
| **first_assistant.md** | 会话开始时的首条 assistant 消息 | 目前留空，不注入开幕词。 |

## 维护原则

- 创建或修改项目的规则以 `sop.create-project` / `sop.modify-project` 为准。
- 操作已有项目的规则以 `sop.execute-function` 为准。
- 平台接口优先读取具体 `system.openapi.*` skill，例如 `system.openapi.hub`、`system.openapi.message`、`system.openapi.scheduled-task`、`system.openapi.permission`、`system.openapi.audit`；无法归类时再用 `system.openapi` 兜底。
- 图片、视频、Excel、临时转换等一次性任务以具体 `system.tools.*` 为准，不确定时使用 `system.tools`。
- `system_prompt.md` 只保留 skill 路由和总约束，避免把 SOP 细节复制到模式提示词里。
