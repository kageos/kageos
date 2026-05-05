# SDK 文档索引

本目录是 Agent-App SDK 的权威参考文档。Skills 只负责按场景路由和执行清单，不复制完整 SDK 规则。

## 分工

- `agent-app-sdk-readme.md`：SDK 总说明，覆盖 Form/Table/Chart、widget、validate、search、文件、link、Chart、错误处理等完整规则。
- `common-runtime-capabilities.md`：所有场景共享的运行能力，覆盖消息、APICall、当前用户、权限/审批边界、定时任务、Table 回调、事务、日志、Python 和构建校验。
- `widget-system.md`：字段组件系统，覆盖 widget type、search、validate、display scenes、files、link、OnSelectFuzzy。
- `form-submit-basic.md`：单 Form 提交任务包，覆盖 Element 表单、FormTemplate、Request/Response、文件处理和验证。
- `table-crud-basic.md`：简单 Table CRUD 任务包，覆盖 Element 表格、TableTemplate、AutoCrudTable、搜索分页、新增编辑删除和验证。
- `combo-table-form.md`：Table + Form 组合任务包，覆盖长期记录管理加独立提交入口、link、事务和验证。
- `combo-table-form-chart.md`：复杂系统组合任务包，覆盖 Table 管数据、Form 做动作、Chart 看统计，以及投票/收银等组合范式。
- `form-table-chart-reference.md`：创建或修改 Form/Table/Chart 时优先读的短参考。
- `widget-reference.md`：字段建模、Go 类型与 widget 类型选择。
- `build-validation-reference.md`：`build_workspace`、`app.Run()`、`CompileAndValidate()`、widget validator 的启动期校验链路。
- `platform-api-reference.md`：`ctx.APICall` 与 `ctx.SendMessage` 的平台能力入口。
- `workbench-tools-sdk-relationship.md`：工作台、tools、docs、skills、SDK、runtime、app-server、web 的整体关系。

## 和 Skills 的关系

- Docs 是权威知识源，适合完整查规则。
- Skills 是任务入口，告诉模型该读哪些 docs、按什么 SOP 执行、怎么验收。
- 创建/修改代码时，优先读匹配的 `sdk.*` skill，再按 skill 的 `required_docs` 读取本目录文档。
