---
id: system.openapi.scheduled-task
name: scheduled-task-openapi
description: 通过 /system/openapi/scheduled_task 创建、查询、取消平台定时任务，并查看定时任务执行记录。
triggers:
  - 定时任务
  - 创建定时任务
  - 查询定时任务
  - 取消定时任务
  - 调度任务
  - cron
  - schedule
  - scheduled task
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-cross-cutting-capabilities
  - /system/prompt/sdk/agent-app-sdk-readme
  - /system/prompt/sdk/platform-api-reference
capabilities:
  - /system/openapi/scheduled_task/create.form
  - /system/openapi/scheduled_task/list.form
  - /system/openapi/scheduled_task/cancel.form
  - /system/openapi/scheduled_task/executions.form
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - search_tools
  - run_form_submit
  - create_scheduled_task
  - list_scheduled_tasks
  - cancel_scheduled_task
  - list_scheduled_task_executions
  - record_workspace_event
completion:
  - 已确认目标函数 full_code_path 和调用 payload
  - 已确认调度类型 atime、cron 或 every 及对应时间参数
  - 创建或取消任务前已获得用户明确授权
  - 已返回任务 ID、状态、下次执行时间或执行记录
---

# Scheduled Task OpenAPI SOP

## 使用条件

用户要创建定时任务、查询定时任务、取消定时任务或查看执行记录时，使用本 skill。业务应用不要自己实现 cron 表、任务队列或轮询逻辑。

## 标准流程

1. 读取 `required_docs`。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 用 `search_tools` 搜索 `/system/openapi/scheduled_task`。
3. 创建任务前必须确认：
   - `full_code_path`：要执行的目标函数完整路径。
   - `action`：默认 `execute`，表格增删改按 schema 选择对应 action。
   - `payload`：必须是合法 JSON 对象字符串。
   - `schedule_type`：`atime`、`cron` 或 `every`。
   - 时间参数：`run_at`、`cron_expr` 或 `interval_seconds`。
   - 通知对象和通知条件。
4. 查询任务和执行记录属于只读；创建和取消属于副作用，需要用户授权。
5. 不要根据自然语言猜 cron 表达式；不确定时先向用户确认具体时间和时区。

## 当前函数

- `/system/openapi/scheduled_task/create.form`：创建定时任务。
- `/system/openapi/scheduled_task/list.form`：查询定时任务。
- `/system/openapi/scheduled_task/cancel.form`：取消定时任务。
- `/system/openapi/scheduled_task/executions.form`：查询定时任务执行记录。
