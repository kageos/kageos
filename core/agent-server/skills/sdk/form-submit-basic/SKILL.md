---
id: sdk.form-submit-basic
name: sdk-form-submit-basic
description: 创建或修改单 Form、一次性提交、文件处理、转换生成、导入、发送或工具函数时使用。覆盖 Element 表单前端形态、FormTemplate、Request/Response、文件处理、link 和验证闭环。
triggers:
  - Form
  - 表单
  - 单表单
  - 文件处理
  - 上传
  - 转换
  - 生成
  - 提交
  - 工具函数
  - 一次性动作
modes:
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/platform-function-architecture
  - /system/prompt/sdk/form-submit-basic
  - /system/prompt/sdk/common-runtime-capabilities
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
recommended_demos:
  - /system/prompt/case_catalog/form/excelorcsv
  - /system/prompt/case_catalog/form/images
  - /system/prompt/case_catalog/form/pdf
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
  - run_on_select_fuzzy
  - run_official_python
completion:
  - 已确认该需求是一次性动作或文件处理，不是长期记录管理
  - 已说明 Form 前端为 Element 表单，提交后展示 Response、文件或链接
  - 已规划 Request、Response 和必要文件字段
  - 已读取匹配案例或说明无需案例
  - 已 build_workspace 并用 run_form_submit 验证核心路径
---

# SDK Form 提交基础

## 使用条件

当用户要做单表单、文件处理、转换生成、上传解析、一次性提交、发送通知或临时工具函数时，使用本 skill。

## 流程

1. 读取本 skill 后，`required_docs` 会自动注入闭环任务包。
2. 在 PRD 中说明：Form 前端会渲染为 Element 表单，用户填写 Request 字段并提交，后端返回 Response 结果。
3. 选择匹配案例，文件处理优先读 `/system/prompt/case_catalog/form/excelorcsv`、`/system/prompt/case_catalog/form/pdf` 或 `/system/prompt/case_catalog/form/images`。
4. 写代码前先读当前目录结构和相关 Go 文件。
5. 生成或修改 Request、Response、FormTemplate、Handler 和必要业务函数。
6. `build_workspace`。
7. 用 `run_form_submit` 验证提交链路；有 OnSelectFuzzy 时验证搜索和回显。

## 关键判断

- 一次性提交、文件处理、转换生成：Form。
- 管理一批长期记录：不要用 Form，读 `sdk.table-crud-basic`。
- 长期记录 + 独立动作：读 `sdk.combo-table-form`。
- 统计图表：不要塞进 Form Response，读组合文档。

## 完成标准

满足 frontmatter `completion` 中所有项目后，才认为任务完成。
