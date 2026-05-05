---
id: system.openapi.audit
name: audit-openapi
description: 通过 /system/openapi/operate_log 和 /system/openapi/directory_history 查询操作日志、表单/表格记录、应用版本和目录变更历史。
triggers:
  - 操作日志
  - 审计
  - 变更记录
  - 资源变更日志
  - 目录历史
  - 应用版本
  - operate log
  - directory history
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-overview
  - /system/prompt/sdk/platform-api-reference
capabilities:
  - /system/openapi/operate_log/table.form
  - /system/openapi/operate_log/form.form
  - /system/openapi/directory_history/app_version.form
  - /system/openapi/directory_history/directory.form
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - run_form_submit
  - run_table_search
  - record_workspace_event
completion:
  - 已确认要查的是操作日志、审计记录、应用版本变更或目录变更
  - 已确认 app_id、full_code_path、row_id、分页等查询条件
  - 已返回可追踪的日志或变更记录
---

# Audit OpenAPI SOP

## 使用条件

用户要查询操作日志、表单/表格记录变更、应用版本历史、目录变更历史、审计记录时，使用本 skill。此类任务一般只读，不要为了查日志去修改业务代码。

## 标准流程

1. 读取 `required_docs`。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 用 `search_tools` 搜索 `/system/openapi/operate_log` 或 `/system/openapi/directory_history`。
3. 明确查询目标：
   - 表格/表单操作日志：确认 `full_code_path`、`row_id`、操作类型、分页条件。
   - 应用版本变更：确认应用 ID 和版本范围。
   - 目录变更：确认应用 ID 和目录完整路径。
4. 查询结果需要保留原始 JSON 或关键记录，方便用户继续审计。

## 当前函数

- `/system/openapi/operate_log/table.form`：查询表格操作日志。
- `/system/openapi/operate_log/form.form`：查询表单操作日志。
- `/system/openapi/directory_history/app_version.form`：查询应用版本变更记录。
- `/system/openapi/directory_history/directory.form`：查询目录变更记录。
