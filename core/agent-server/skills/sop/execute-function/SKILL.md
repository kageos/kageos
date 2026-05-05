---
id: sop.execute-function
name: execute-function
description: 用户要操作已有系统、查表、提交表单、查图表、调用已有工具或处理文件时使用。优先复用当前目录函数、system 工具函数或 Hub 能力。
triggers:
  - 查一下
  - 执行
  - 调用工具
  - 提交表单
  - 查询表格
  - 处理文件
  - 生成图表
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-overview
recommended_demos:
  - /system/prompt/case_catalog/form/excelorcsv
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - search_tools
  - run_form_submit
  - run_table_search
  - run_table_create
  - run_table_batch_create
  - run_table_update
  - run_table_delete
  - run_chart_query
  - run_on_select_fuzzy
  - search_hub_directory
  - copy_directory
  - run_official_python
  - create_scheduled_task
  - list_scheduled_tasks
  - cancel_scheduled_task
  - list_scheduled_task_executions
  - create_scheduled_agent_task
  - list_scheduled_agent_tasks
  - list_scheduled_agent_task_executions
  - run_scheduled_agent_task_now
completion:
  - 已确认目标函数 full_code_path 和 schema 字段
  - 未根据函数名猜 body
  - 已调用对应 run 工具或说明无法调用原因
  - 输出文件或关键结果已明确返回给用户
---

# 执行已有函数 SOP

## 使用条件

用户要求查数据、提交表单、查询图表、执行已有工具、处理文件，且不需要新增或修改代码时使用本 skill。

## 流程

1. 先看当前环境中的可执行函数，确认是否已有合适能力。旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 当前目录没有合适能力时，用 `search_tools` 搜 system 已注册函数和内置工具。
3. 仍没有时，再考虑 `search_hub_directory`。
4. 调用前必须确认目标函数的 schema 字段、必填项、枚举值和文件字段。
5. Form 用 `run_form_submit`。
6. Table 默认先用 `run_table_search`，只有 schema/能力摘要明确支持写入时才用 create/update/delete。
7. Chart 用 `run_chart_query`。
8. 有 OnSelectFuzzy 时可用 `run_on_select_fuzzy` 调试选项。

## 关键约束

- 不要根据函数名、路由名或经验猜请求 body。
- 用户在表格/项目操作上下文里说“搜索一下”“试试搜索”“搜索效果”时，默认指 Table Search，先用 `run_table_search`；只有用户明确说“下拉搜索”“联想搜索”“选择框搜索”“OnSelectFuzzy”时，才用 `run_on_select_fuzzy`。
- `run_table_search.url_query` 必须使用 `操作符=字段:值` 格式，不要写成 `字段=值`、`eq_id=3`、`id=3` 或 `eq_id=3,4`。
- 按 ID 查询：`eq=id:3`；多 ID 查询：`in=id:3,4`；模糊查询：`like=name:奶茶`；组合条件用 `&`：`eq=id:3&page=1&page_size=20`。
- 如果查询结果明显没有被过滤，例如查 ID 却返回全表，先检查 `url_query` 格式，不要把全量返回当作筛选成功。
- `render_default` 是前端默认值，不会自动进入 body；需要使用时必须显式传入。
- 文件字段传 `bucket/object_key` refs 字符串，多文件用英文逗号分隔。
- 函数返回里如有输出文件，工作台会展示，不要编造 URL。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
