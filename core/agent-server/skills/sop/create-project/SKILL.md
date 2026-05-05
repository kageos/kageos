---
id: sop.create-project
name: create-project
description: 创建新应用、新模块、新目录、新 Form/Table/Chart，或用户要求“做一个系统/工具/管理后台”时使用。执行前必须先完成需求澄清和 PRD 确认。
triggers:
  - 创建系统
  - 新建应用
  - 新建模块
  - 生成管理后台
  - 做一个工具
  - 创建项目
modes:
  - execute
  - dev
  - agent
required_docs:
  - /system/prompt/platform-capability-boundaries
  - /system/prompt/platform-overview
  - /system/prompt/platform-function-architecture
  - /system/prompt/platform-cross-cutting-capabilities
  - /system/prompt/sdk/agent-app-sdk-readme
recommended_demos:
  - /system/prompt/case_catalog/table/ticket
  - /system/prompt/case_catalog/form/excelorcsv
  - /system/prompt/case_catalog/form_table_chart/cashier
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - write_doc
  - write_go_file
  - search_replace_file
  - delete_file
  - create_directory
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
  - search_tools
  - search_hub_directory
  - copy_directory
completion:
  - 已按需求选择 Form/Table/Chart，不混用函数类型
  - 创建类需求已先输出 PRD 并获得用户确认
  - 所有新增路由后缀与 Template 类型匹配
  - 所有新增 Go 文件名只使用普通 .go，不把 .table/.form/.chart 写进文件名
  - build_workspace 已通过
  - 至少验证一个核心函数
---

# 创建项目 SOP

## 使用条件

用户要求创建新应用、新模块、新目录、新函数，或表达“帮我做一个系统/管理后台/工具”时使用本 skill。

## 流程

1. 读取本 skill 后，工具会自动注入 `required_docs` 内容；旧 `/system/prompt/workspace/*` SOP 已下线，不再读取。
2. 先按 `/system/prompt/platform-function-architecture` 判断目录和 Form/Table/Chart 组合：管理一批记录用 Table，一次性动作用 Form，统计图表用 Chart，并在 PRD 里说明前端渲染形态。
3. 按 4 个粗场景选择任务包：单 Form 读 `sdk.form-submit-basic`，单 Table 读 `sdk.table-crud-basic`，Table + Form 读 `sdk.combo-table-form`，Table + Form + Chart 读 `sdk.combo-table-form-chart`。
4. 根据需求选择至少一个匹配案例并 `read_doc`，不要未读案例就出 PRD。
5. 创建类需求必须先输出 PRD，让用户确认后再写代码；PRD 必须包含示例数据和确认后将创建的目录/函数清单。
6. 写代码前先用 `read_dir` / `read_go_file` 看当前目录结构和已有文件。
7. 代码只写 SDK Go 应用，不生成独立 HTML/CSS/JS 页面；Go 文件名只用普通 `.go`，路由后缀只写在 `packageContext.GET/POST` 的路由字符串里。
8. 需要新增文件时用 `write_go_file`，需要局部修改时优先用 `search_replace_file`；任一写入/替换工具返回 error 时，本次未落盘，必须先修正该工具调用，不要继续声称文件已创建。
9. 复杂系统必须分阶段写入和构建：先主数据 Table，`build_workspace` 通过后再写 Form 动作，再 build，通过后再写 Chart 统计；不要一次性写完 4-5 个大文件后才首次编译。
10. build 失败时读取完整错误，按同类问题批量修复后再次 build。
11. build 成功后用对应 run 工具验证核心函数。

## 模糊系统需求处理

用户只说“帮我整个评价系统 / 工单系统 / 客户管理系统 / 管理后台”这类泛化需求时，不要抛大段选择题，也不要要求用户先填写完整功能清单。

- 先按平台后台的合理默认假设建模，并在 PRD 里明确“默认假设”。
- 只有某个问题会导致完全不同的数据模型或权限边界时，最多问 1 个阻塞问题；否则直接出 PRD 让用户确认。
- 默认优先做一个核心 `TableTemplate` 管理主记录；明确需要提交入口时再加 `FormTemplate`，明确需要统计时再加 `ChartTemplate`。
- 对“评价系统”默认按 Table + Form 处理：`evaluation_object_list.table` 管理评价对象，`evaluation_submit.form` 让用户提交一次评价，`evaluation_record_list.table` 查看和处理评价记录；评分趋势、标签占比等只有用户要求统计时才加 Chart。
- 平台已有权限、审批、评论、收藏、消息、操作日志等横切能力，不要在业务代码中重复造表或造流程；PRD 中只说明如何使用这些平台能力。
- PRD 不要默认承诺“仅创建人可编辑”“管理员可见”等行级权限规则；除非用户明确要求并准备在业务回调中实现，否则统一说明函数访问权限由平台按 `full_code_path` 管理。
- 评价状态可以作为“待审核/已发布/已隐藏”这类内容发布状态处理，但不要把它描述成平台通用审批流。
- 用户确认 PRD 后再写代码；未确认前不要创建文件。

## PRD 格式

PRD 必须用纯 Markdown，并包含以下内容：

1. 业务目标和范围：一句话说明系统解决什么问题，不擅自扩大范围。
2. 函数类型判断：明确需要 Table、Form、Chart 中哪些类型，前端分别如何展示，为什么这么拆。
3. 落地目录和函数清单：写清确认后会创建的应用目录，例如 `/用户/应用/evaluation`，并列出每个 `.table`、`.form`、`.chart` 路由、中文名称、前端形态和职责；不要只写抽象模块名。
4. 表单字段（新增/编辑）：使用五列表格「字段 | 类型 | 必填 | 默认值 | 说明」，只列用户需要填写的字段。
5. 列表模式：包含系统字段 `ID`、创建时间、更新时间、创建人，业务字段，以及仅列表展示的计算字段；如果写“操作/审核/隐藏/回复/发布/下架”等行操作，必须同时说明由哪个 Table 回调或 link 实现，否则不要在列表样例里承诺该操作。
6. 示例数据：至少给每个核心 Table/Form 一到两条贴近业务的样例行，帮助用户理解前端表格会展示什么、表单会填什么、提交后会生成什么记录。
7. 业务规则：写清状态流转、自动生成字段、跨表联动、默认排序/筛选和只读限制。
8. 创建确认：在确认语前写「确认后我将创建目录：xxx，并生成：a.table、b.form、c.table」。
9. 确认语：末尾必须写「请确认以上是否 OK，确认后我再生成代码。」

推荐结构：

```text
## 一、业务目标和范围
## 二、函数类型判断
## 三、落地目录和函数清单
## 四、字段设计
## 五、列表模式
## 六、示例数据
## 七、业务规则
## 八、确认后创建内容
请确认以上是否 OK，确认后我再生成代码。
```

「落地目录和函数清单」示例：

| 路由 | 类型 | 前端形态 | 职责 |
| --- | --- | --- | --- |
| `evaluation_object_list.table` | Table | Element 表格，支持搜索、分页、新增、编辑 | 管理评价对象 |
| `evaluation_submit.form` | Form | Element 表单，提交评分、评语和附件 | 用户提交一次评价 |
| `evaluation_record_list.table` | Table | Element 表格，默认只读展示评价记录 | 查看评价记录 |

「示例数据」示例：

| 表/表单 | 示例 |
| --- | --- |
| 评价对象表 | 对象名称：课程 A；分类：课程；负责人：张三；状态：开放；评价次数：12 |
| 提交评价表单 | 评价对象：课程 A；评分：5；评价内容：讲解清晰；附件：课堂截图 |
| 评价记录表 | 对象：课程 A；评分：5；提交人：李四；状态：待审核；提交时间：2026-05-04 10:00 |

表单字段类型使用用户能理解的话，如文本输入、多行文本、下拉选择、用户选择、时间选择、数字输入、滑块、多选下拉、文件上传。计算字段、后端自动生成字段不要放进表单字段表。静态 `select` / `multiselect` 要规划 `options_colors`，颜色只用不带 `#` 的 6 位十六进制 `RRGGBB`；动态 OnSelectFuzzy 下拉只规划选项来源。

## 案例选择

- 单表 CRUD：`/system/prompt/case_catalog/table/ticket`
- 文件上传或单 Form：`/system/prompt/case_catalog/form/excelorcsv`
- 主从表：`/system/prompt/case_catalog/tables/hr` 或 `/system/prompt/case_catalog/tables/meeting`
- Table + Form：`/system/prompt/case_catalog/formandtable/vote`
- Table + Form + Chart：`/system/prompt/case_catalog/form_table_chart/cashier`

## 任务包选择

- 单 Form/文件处理/转换生成/一次性提交：先 `read_skill("sdk.form-submit-basic")`。
- 简单后台/CRUD/台账：先 `read_skill("sdk.table-crud-basic")`。
- Table + Form，例如评价对象 + 提交评价、投票主题 + 投票提交、快速跟进、导入记录：先 `read_skill("sdk.combo-table-form")`。
- Table + Form + Chart，例如投票结果图表、收银统计、工单统计、客户看板：先 `read_skill("sdk.combo-table-form-chart")`。
- 字段组件拿不准：读 `sdk.widget-selection`。
- 本实验链路默认注入 `/system/prompt/sdk/agent-app-sdk-readme`，写代码时必须以其中真实 SDK API、widget 白名单和案例为准；不要按名称猜 API 或 widget。

## 关键约束

- `TableTemplate` 路由必须以 `.table` 结尾。
- `FormTemplate` 路由必须以 `.form` 结尾。
- `ChartTemplate` 路由必须以 `.chart` 结尾。
- 路由后缀不是文件名后缀；Go 文件名只用普通 `.go`。正确：`evaluation_object_list.go` 里注册 `packageContext.GET("evaluation_object_list.table", ...)`；错误做法是把 `.table`、`.form` 或 `.chart` 再拼到 `.go` 前面。
- 平台已有权限、审批、评论、收藏、消息、操作日志等横切能力，不要在业务代码中重复实现。
- struct tag 是 UI 协议，widget 类型必须来自 SDK 支持列表。
- 不要创建或修改 `init.go` / `init_.go`。
- 流水、日志、审计等记录类表默认只读，除非用户明确要求写能力。
- PRD 和代码必须一致：事实记录表若写“默认只读”，就不要承诺新增/编辑/删除/审核按钮；若确实需要审核、隐藏、回复等受控修改，PRD 中要明确配置 `OnTableUpdateRow`，实现时也必须写对应回调。
- 跨表联动涉及余额、库存、数量、状态或流水时必须事务化处理。
- 写完必须 `build_workspace`，不能只落盘不编译。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
