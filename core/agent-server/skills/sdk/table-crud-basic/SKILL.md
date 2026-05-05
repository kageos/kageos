---
id: sdk.table-crud-basic
name: sdk-table-crud-basic
description: 创建或修改简单 Table CRUD、后台管理列表、台账、记录库时使用。覆盖 Element 表格前端形态、TableTemplate、AutoCrudTable、搜索分页、新增编辑删除、只读表和验证闭环。
triggers:
  - CRUD
  - 增删改查
  - Table CRUD
  - 表格管理
  - 列表管理
  - 后台管理
  - 台账
  - 记录管理
  - 客户管理
  - 工单管理
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/platform-function-architecture
  - /system/prompt/sdk/table-crud-basic
  - /system/prompt/sdk/common-runtime-capabilities
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
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
  - run_table_search
  - run_table_create
  - run_table_update
  - run_table_delete
  - run_on_select_fuzzy
completion:
  - 已确认该需求是长期记录管理，不是一次性 Form 或纯 Chart
  - 已说明 Table 前端为 Element 表格列表，包含搜索、分页、列展示和操作入口
  - 已确认是否开放新增、编辑、删除、批量导入
  - 若是收银记录、支付流水、导入历史、审计记录等事实记录表，已默认不配置写操作回调
  - 已读取匹配案例或说明无需案例
  - 已 build_workspace 并用 run_table_* 验证核心路径
---

# SDK Table CRUD 基础

## 使用条件

当用户要做后台管理、列表、台账、CRUD、客户/工单/商品/订单/库存等长期记录管理时，使用本 skill。

## 流程

1. 读取本 skill 后，`required_docs` 会自动注入闭环任务包。
2. 在 PRD 中说明：Table 前端会渲染为 Element 表格，支持搜索、分页、列展示，写操作入口由回调决定。
3. 选择匹配案例，单表优先 `/system/prompt/case_catalog/table/ticket`。
4. 写代码前先读当前目录结构和相关 Go 文件。
5. 生成或修改 TableTemplate、Model、Request、List 函数和必要回调。
6. `build_workspace`。
7. 用 `run_table_search` 验证列表；有写能力时继续验证 create/update/delete。

## 关键判断

- 只是管理一批记录：Table。
- 一次性处理文件、转换、发送：不要强塞 Table 新增，读 `sdk.form-submit-basic`。
- 提交评价、导入记录、快速跟进这类长期记录 + 独立动作：读 `sdk.combo-table-form`。
- 带统计看板的复杂系统：读 `sdk.combo-table-form-chart`。
- 统计趋势、占比、看板：需要 Chart，不要写进 Table。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
