---
id: sop.modify-project
name: modify-project
description: 修改已有应用、已有目录、已有函数或已有代码时使用。适合新增字段、调整组件、修复 bug、优化已有业务逻辑。
triggers:
  - 修改
  - 调整
  - 修复
  - 优化
  - 新增字段
  - 改已有功能
modes:
  - execute
  - modify
  - dev
  - agent
required_docs:
  - /system/prompt/platform-capability-boundaries
  - /system/prompt/platform-function-architecture
  - /system/prompt/sdk/form-table-chart-reference
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
  - /system/prompt/sdk/agent-app-sdk-readme
recommended_demos:
  - /system/prompt/case_catalog/table/ticket
  - /system/prompt/case_catalog/tables/meeting
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - write_go_file
  - search_replace_file
  - delete_file
  - build_workspace
  - read_app_log
  - run_form_submit
  - run_table_search
  - run_table_create
  - run_table_batch_create
  - run_table_update
  - run_table_delete
  - run_chart_query
  - run_on_select_fuzzy
completion:
  - 已读取目标文件和相关上下文
  - 改动范围只覆盖用户要求和必要依赖
  - build_workspace 已通过
  - 已验证受影响函数或说明无法验证的原因
---

# 修改项目 SOP

## 使用条件

用户要求修改、修复、调整、优化已有功能时使用本 skill。

## 流程

1. 先确认目标目录或函数，必要时用 `read_dir` 看结构。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 读取相关 Go 文件，不要凭函数名猜实现。
3. 小改优先用 `search_replace_file`，大改或新增文件再用 `write_go_file`。
4. 不要修改 `init_.go`，该文件由脚手架管理。
5. 保留用户已有改动，不做无关重构。
6. 修改 widget、schema、Template、路由时必须检查对应任务包：单 Form 读 `sdk.form-submit-basic`，单 Table 读 `sdk.table-crud-basic`，Table + Form 读 `sdk.combo-table-form`，Table + Form + Chart 读 `sdk.combo-table-form-chart`。
7. 修改完成后调用 `build_workspace`。
8. build 失败时根据错误修复；成功后用 run 工具验证受影响函数。

## 关键约束

- 不要把 Form/Table/Chart 的结构互相混用。
- 不要引入 SDK 不支持的 widget type。
- 修改 Table 写能力时要确认对应回调是否存在。
- 问题若来自平台横切能力，不要在业务代码里临时重造一套。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
