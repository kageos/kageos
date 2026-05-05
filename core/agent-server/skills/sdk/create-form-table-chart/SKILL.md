---
id: sdk.create-form-table-chart
name: sdk-create-form-table-chart
description: 使用 Agent-App SDK 创建或修改 Form、Table、Chart 函数时使用。覆盖路由后缀、Template、handler、响应构建、案例选择和验证闭环。
triggers:
  - Form
  - Table
  - Chart
  - 表单
  - 表格
  - 图表
  - 创建函数
  - 修改函数
  - Template
  - 路由后缀
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/platform-function-architecture
  - /system/prompt/sdk/form-table-chart-reference
  - /system/prompt/sdk/common-runtime-capabilities
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
recommended_demos:
  - /system/prompt/case_catalog/table/ticket
  - /system/prompt/case_catalog/form/excelorcsv
  - /system/prompt/case_catalog/form_table_chart/cashier
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - write_go_file
  - search_replace_file
  - delete_file
  - build_workspace
  - run_form_submit
  - run_table_search
  - run_table_create
  - run_table_update
  - run_table_delete
  - run_chart_query
  - run_on_select_fuzzy
completion:
  - 已确认需求应使用 Form、Table 还是 Chart
  - 已确认路由后缀与 Template 类型匹配
  - 已确认 Go 文件名只使用普通 .go，未把路由后缀写进文件名
  - 已读取至少一个匹配案例或说明无需案例
  - 已完成 build_workspace
  - 已用 run_* 验证核心函数或说明无法验证原因
---

# SDK Form/Table/Chart 写法

## 使用条件

创建或修改 SDK 函数时使用本 skill。它只负责 SDK 写法；业务 SOP 仍由 `sop.create-project` 或 `sop.modify-project` 承接。涉及函数类型选择时，先按 `/system/prompt/platform-function-architecture` 判断前端渲染形态和组合职责。

## 基本判断

- 管理一批长期记录：Table，前端以 Element 表格展示列表、搜索、分页和操作入口。
- 一次性提交并返回结果：Form，前端以 Element 表单收集输入并展示一次处理结果。
- 统计查询并渲染图表：Chart，前端以筛选条件 + ECharts 图表展示统计结果。
- 创建新功能时优先按四个粗场景读任务包：`sdk.form-submit-basic`、`sdk.table-crud-basic`、`sdk.combo-table-form`、`sdk.combo-table-form-chart`。

## 硬规则

1. `TableTemplate` 路由必须以 `.table` 结尾。
2. `FormTemplate` 路由必须以 `.form` 结尾。
3. `ChartTemplate` 路由必须以 `.chart` 结尾。
4. 路由后缀只属于注册路由字符串，不属于 Go 文件名。正确：`order_list.go` + `packageContext.GET("order_list.table", ...)`；不要把 `.table`、`.form` 或 `.chart` 再拼到 `.go` 前面。
5. Form 的 Request/Response 不要混入 `chart.LineChart` 等图表结构；图表单独用 Chart 路由。
6. Table 的列表查询使用 `resp.Table(...).AutoSearchFilterPaged(...).Build()`。
7. Form 使用 `ctx.ShouldBindValidate(&req)` 后 `resp.Form(&out).Build()`。
8. Chart 使用 `ctx.ShouldBind(&req)` 后 `resp.Chart(chart).Build()`。

## 流程

1. 读 `/system/prompt/platform-function-architecture`、`/system/prompt/sdk/form-table-chart-reference` 和 `/system/prompt/sdk/widget-system`。
2. 按需求选择匹配案例文档。
3. 读现有目录和相关 Go 文件。
4. 写代码或局部替换。
5. 统一 `build_workspace`。
6. 用对应 `run_*` 工具验证。

只有任务包、短参考和案例仍无法覆盖具体 SDK 细节时，才按需读取 `/system/prompt/sdk/agent-app-sdk-readme`，不要默认全文加载。
