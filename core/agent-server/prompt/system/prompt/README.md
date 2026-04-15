# 系统提示词

本目录是本地 prompt seed 根目录，对应树上的 `/system/prompt`。开发阶段改这里，启动时会把目录和文档 upsert 到服务树；运行时优先读取树上的内容，缺失时再回落到本地 seed。

- **doc/**：公用文档与配置，如环境模板、文档目录。
- **mode/**：工作台各模式的配置与提示词。
- **workspace/**：按任务类型拆分的工作台操作文档。
- **sdk/**：框架与平台使用手册。
- **case_catalog/**：案例目录，按类型归档真实示例。
- **_rebuild/**：提示词体系重构草案与迁移计划，仅供开发阶段整理结构使用。
- **platform-overview.md / platform-cross-cutting-capabilities.md**：平台总览与横切能力说明。
