# SDK 文档索引

本目录是 Agent-App SDK 的权威参考文档。SOP/文档包只负责按场景提供流程和执行清单，不复制完整 SDK 规则。

## 分工

- `agent-app-sdk-readme.md`：SDK 总说明，覆盖 Form/Table/Chart、widget、validate、筛选字段、文件、link、Chart、错误处理等完整规则。
- `reference/`：按需专项参考。不要默认注入；由各身份 SOP 按场景指向具体 `read_doc` 路径。

## 使用方式

- Docs 是权威知识源，适合完整查规则。
- 当前身份 SOP 决定流程、确认点和验收方式。
- 创建/修改代码时，先按当前身份文档包执行；上下文不足时再按明确路径读取本目录文档。
- 原先的 Form/Table/Combo/widget 分散任务包已并入 SDK 主文档和案例链路，不再放在主 prompt 树。
