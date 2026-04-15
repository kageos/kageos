# 工作台模式

本目录按模式组织各模式自己的配置与提示词，每个子目录对应一种工作台模式（如 `dev`、`modify`、`execute`）。公用文档在上一级 `doc/` 与 `workspace/` 中统一维护。

## 目录约定

- `mode/<code>/`：如 `dev`、`modify`、`execute`
- 每个模式目录内建议：
  - **config.json**：该模式的元信息与行为配置（name、description、tool_names、引用的 md 文件名等）
  - **system_prompt.md**：该模式拼进 system 的片段（身份/规则/工具说明等）
  - **first_assistant.md**（可选）：该模式首条 assistant 内容

## 配置文件说明（config.json）

- `name`：模式展示名
- `description`：模式简短说明
- `tool_names`：该模式启用的工具名列表（与 DB workspace_mode.tool_names 可对齐或由配置优先）
- `system_prompt_file`：本目录下 system 片段文件名，如 `system_prompt.md`
- `first_assistant_file`（可选）：首条 assistant 文件名，如 `first_assistant.md`

实现层：各模式对应的 `WorkspaceModePromptProvider` 从本地 seed 或树上的 `/system/prompt/mode/<code>` 读取配置和文档。
