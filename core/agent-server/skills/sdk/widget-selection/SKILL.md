---
id: sdk.widget-selection
name: sdk-widget-selection
description: 选择 Agent-App SDK widget 组件、Go 类型、search/validate 标签和字段展示方式时使用。适用于数组、枚举、时间、文件、用户、部门、link、嵌套 form/table 等字段建模问题。
triggers:
  - widget
  - 组件选择
  - 字段类型
  - Go 类型
  - 数组
  - list
  - item_type
  - select
  - multiselect
  - datetime
modes:
  - qa
  - dev
  - modify
  - agent
required_docs:
  - /system/prompt/sdk/widget-system
  - /system/prompt/sdk/build-validation-reference
allowed_tools:
  - read_doc
  - read_dir
  - read_go_file
  - read_go_file_lines
  - write_go_file
  - search_replace_file
  - build_workspace
completion:
  - 已确认字段是自由输入、候选选择、只读展示还是关联跳转
  - 已确认 Go 类型与 widget 类型匹配
  - 已确认数组字段使用 list 或 multiselect 的边界
  - 已确认 search/validate/display 标签与实际提交值一致
  - 修改代码时已通过 build_workspace 或说明未执行原因
---

# SDK Widget 选择

## 使用条件

当用户问“某个 Go 类型该用什么组件”、或要生成/修改带字段的 Form/Table/Chart 时使用本 skill。

## 决策规则

1. 自由输入多个值用 `type:list`。
   - `[]int`、`[]float64` 等数字数组：`type:list;item_type:number`
   - `[]string` 文本数组：`type:list;item_type:text`
2. 从候选项里选择多个值用 `multiselect`，不是 `list`。
3. 少量固定枚举平铺勾选用 `checkbox`。
4. 单选枚举用 `select` 或 `radio`；静态 `select/multiselect` 同时写 `options_colors`，颜色只用不带 `#` 的 6 位十六进制 `RRGGBB`；动态 OnSelectFuzzy 下拉不写 `options_colors`。
5. 日期时间用 `types.Time` + `type:datetime`，不要生成未支持的 `date`、`time`、`range`。
6. 文件、图片、视频、PDF 上传都先用 `type:files`，字段类型为 `string`。
7. 用户/部门选择用 `user/users/department/departments`，存储值直接传给平台接口或消息接口。
8. 只读跳转用 `link`，跳 Table 用目标表 Model，跳 Form 用目标 Form Request，跳 Chart 用目标 Chart Request。

## 必读文档

先读 `/system/prompt/sdk/widget-system`。如果是构建或启动校验失败，再读 `/system/prompt/sdk/build-validation-reference`。
