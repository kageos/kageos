# prompt 嵌入内容根目录（content）

本目录由 **`//go:embed content`** 整目录嵌入，配置、公用提示词、各模式个性化等**全部放在这一层子目录**，代码通过 `promptFS` 可读取其下所有文件。

- **doc/**：公用提示词与文档（环境模板、文档目录等）
- **mode/**：各模式个性化（dev/modify/execute/agent 的 config.json + system_prompt.md + first_assistant.md）
- **builtin/**：内置文档（SDK、案例、workspace 子文档），由 read_doc 按需拉取

Go 的 `//go:embed` 不允许用 `.` 嵌入当前目录，因此再包一层子目录 `content/`，只嵌入该子目录即可。
