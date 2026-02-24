# prompt 嵌入内容根目录（content）

本目录由 **`//go:embed content`** 整目录嵌入，配置、公用提示词、各模式个性化等**全部放在这一层子目录**，代码通过 `promptFS` 可读取其下所有文件。

- **doc/**：公用提示词与文档（工作台操作提示词、环境模板、文档目录、SDK 文档等）
- **mode/**：各模式个性化（dev/modify/execute 的 config.json + system_prompt.md + first_assistant.md）
- **提示词现状分析.md**、**excel转换管理系统插件提示词.md** 等：其他提示词/分析文档，需要时从 `promptFS` 按路径读取

Go 的 `//go:embed` 不允许用 `.` 嵌入当前目录，因此再包一层子目录 `content/`，只嵌入该子目录即可；配置啥的都放这里。
