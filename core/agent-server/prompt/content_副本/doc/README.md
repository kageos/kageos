# 公用提示词与文档（doc）

本目录存放**所有模式共用**的提示词与配置，由代码通过 `//go:embed content` 嵌入（content 含 doc 与 mode），doc 为公用提示词与文档。

## 约定

- **工作台操作提示词.md**：工作台通用操作规则、PRD 先行、SOP、工具说明等，拼入 system。
- **工作台环境模板.md**：工作台环境信息模板，占位符由代码填充（用户、时间、当前目录、可读文档列表等）。
- **文档目录.json**：可读文档目录（full_code_path、name、when_to_use），用于系统消息「可读的目录」块与 read_doc 拉取。
- SDK 使用手册在 **content/builtin/doc/sdk/agent-app-sdk-readme.md**，按 full_code_path `/builtin/doc/sdk/agent-app-sdk-readme` 暴露，供模型 read_doc 按需拉取。

## 与 mode 的关系

- **doc/**：公用内容，所有工作台模式共享。
- **mode/<code>/**：各模式个性化（system_prompt、first_assistant、config.json 等）。

代码在 init 时从嵌入的 doc 加载上述文件，无需在业务里硬编码路径。
