# 系统提示词

本目录是本地 prompt seed 根目录，对应树上的 `/system/prompt`。开发阶段改这里，启动时会把目录和文档 upsert 到服务树；运行时优先读取树上的内容，缺失时再回落到本地 seed。

- **doc/**：公用文档与配置，如环境模板、文档目录。
- **mode/**：工作台各模式的配置与提示词。
- **sdk/**：框架与平台使用手册。
- **case_catalog/**：案例目录，按类型归档真实示例。
- **platform-capability-boundaries.md**：平台能力边界说明。
