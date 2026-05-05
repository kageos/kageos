---
id: sdk.combo-table-form
name: sdk-combo-table-form
description: 设计或实现 Table + Form 组合场景时使用，适合长期对象管理加用户侧一次性提交动作，但暂时不需要 Chart。覆盖 Element 表格、Element 表单、link、事务、只读记录和验证闭环。
triggers:
  - Table Form
  - table form
  - table+form
  - 表格表单
  - 提交评价
  - 评价系统
  - 评价对象
  - 用户评价
  - 提交入口
  - 快速跟进
  - 导入记录
  - 记录+动作
  - 投票提交
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/platform-function-architecture
  - /system/prompt/sdk/combo-table-form
  - /system/prompt/sdk/table-crud-basic
  - /system/prompt/sdk/form-submit-basic
  - /system/prompt/sdk/common-runtime-capabilities
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
recommended_demos:
  - /system/prompt/case_catalog/formandtable/vote
  - /system/prompt/case_catalog/tables/hr
  - /system/prompt/case_catalog/form/excelorcsv
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
  - run_on_select_fuzzy
completion:
  - 已拆出长期数据 Table 和一次性动作 Form 的职责边界
  - PRD 已说明 Table/Form 的前端渲染形态和目录/路由
  - 已读取至少一个组合案例
  - 已验证核心 Table 查询和 Form 提交
  - 已说明 link、事务、只读记录或平台横切能力取舍
---

# SDK Table/Form 组合

## 使用条件

用户要做“长期记录管理 + 用户侧一次性提交动作”时使用本 skill。典型场景：评价对象 + 提交评价、投票主题 + 投票提交、客户档案 + 快速跟进、Excel 导入 + 导入历史。

## 流程

1. 读取本 skill 后，`required_docs` 会自动注入 Table + Form 组合任务包和基础文档。
2. 先拆职责：哪些是长期数据 Table，哪些是一次动作 Form。
3. 选择组合案例：投票读 `/system/prompt/case_catalog/formandtable/vote`，导入处理读 `/system/prompt/case_catalog/form/excelorcsv`。
4. 创建类需求先输出 PRD，让用户确认后再写代码。
5. 实现时分阶段推进：先 Table，再 Form，再 link 和联动。
6. 每阶段都 `build_workspace`，并用对应 `run_*` 验证。

## PRD 要求

PRD 开头必须写“函数类型判断”，说明：

- 哪些对象用 Table，前端以 Element 表格展示什么列和搜索。
- 哪些动作用 Form，前端以 Element 表单提交什么输入，后端执行什么校验、事务或副作用。
- Table 和 Form 如何通过 link 或参数连接。
- 当前是否不需要 Chart；如果需要统计看板，升级到 `sdk.combo-table-form-chart`。

## 关键约束

- 只是简单新增/编辑记录时，用 Table 回调，不要额外建 Form。
- 有用户侧提交、导入、快速跟进这类独立动作时，优先拆 Form。
- 简单新增、编辑、审核、隐藏、回复、状态更新优先走 Table 回调，不要额外拆 Form。
- Form 写多张表或关键状态时要事务化。
- 不要手写独立前端页面。
- 平台横切能力不要重复实现。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
