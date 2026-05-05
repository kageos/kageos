---
id: sdk.combo-table-form-chart
name: sdk-combo-table-form-chart
description: 设计或实现复杂业务系统时使用，尤其是 Table 管长期数据、Form 做一次性动作、Chart 做统计分析的组合场景。适合投票、收银、工单、客户管理、库存等多函数系统。
triggers:
  - Table Form Chart
  - table form chart
  - 组合系统
  - 复杂系统
  - 多函数
  - 投票系统
  - 收银系统
  - 工单系统
  - 客户管理系统
  - 库存系统
  - 统计看板
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/platform-function-architecture
  - /system/prompt/sdk/combo-table-form-chart
  - /system/prompt/sdk/table-crud-basic
  - /system/prompt/sdk/form-submit-basic
  - /system/prompt/sdk/common-runtime-capabilities
  - /system/prompt/sdk/form-table-chart-reference
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
recommended_demos:
  - /system/prompt/case_catalog/formandtable/vote
  - /system/prompt/case_catalog/form_table_chart/cashier
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
  - run_table_search
  - run_table_create
  - run_table_update
  - run_table_delete
  - run_form_submit
  - run_chart_query
  - run_on_select_fuzzy
completion:
  - 已拆出长期数据 Table、一次性动作 Form、统计分析 Chart 的职责边界
  - PRD 已说明每类函数的前端渲染形态和目录/路由
  - 已读取至少一个组合案例
  - 已分阶段 build_workspace 并验证核心 Table/Form/Chart 路径
  - 已确认收银记录、支付流水等事实记录表默认只读且不配置新增/编辑/删除回调
  - 已说明 link、事务、只读流水或平台横切能力取舍
---

# SDK Table/Form/Chart 组合

## 使用条件

用户要做一个完整业务系统，且需求不止单表 CRUD 时，使用本 skill。典型场景：投票系统、收银系统、工单系统、客户管理系统、库存系统、带统计看板的管理后台。

## 流程

1. 读取本 skill 后，`required_docs` 会自动注入组合任务包和基础文档。
2. 先拆职责：哪些是长期数据 Table，哪些是一次动作 Form，哪些是统计 Chart。
3. 选择组合案例：投票读 `/system/prompt/case_catalog/formandtable/vote`，收银读 `/system/prompt/case_catalog/form_table_chart/cashier`。
4. 创建类需求先输出 PRD，让用户确认后再写代码。
5. 实现时分阶段推进：先 Table，再 Form，再 Chart，再 link 和联动。
6. 每阶段都 `build_workspace`，并用对应 `run_*` 验证。

## PRD 要求

PRD 开头必须写“函数类型判断”，说明：

- 哪些对象用 Table，前端以 Element 表格展示什么列和搜索。
- 哪些动作用 Form，前端以表单提交什么输入，后端执行什么事务或副作用。
- 哪些指标用 Chart，前端展示什么图表。
- Table/Form/Chart 如何通过 link 或参数连接。

## 关键约束

- 不要把复杂系统写成一个大 Form。
- 不要把收银结算、提交评价、投票提交这类独立动作塞进 Table 新增。
- 不要把 Chart 结构放进 Form Response。
- 不要手写独立前端页面。
- 平台横切能力不要重复实现。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
