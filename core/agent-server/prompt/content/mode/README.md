# 工作台模式提示词模块

本目录按**模式**组织**各模式个性化**提示词与配置，每个子目录对应一种工作台模式（如 dev / modify / execute）。  
公用提示词（工作台操作提示词、环境模板、文档目录、SDK 文档等）在上一级 **doc/** 中，由代码统一嵌入；本目录仅放各模式自己的 system_prompt、first_assistant、config。  
代码通过 **interface + 多态** 按模式加载本目录下的 config + md，**尽量不在代码里硬编码** 文案与工具列表。

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

实现层：各模式对应的 `WorkspaceModePromptProvider` 实现从本目录读取 config + 上述 md，封装在内部，对外只暴露接口方法。
