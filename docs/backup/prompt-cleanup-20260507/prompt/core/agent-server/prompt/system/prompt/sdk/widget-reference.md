# SDK Widget 参考

本文档用于字段建模和组件选择。完整规则以 `/system/prompt/sdk/widget-system` 为准。

## 组件选择

| 需求 | Go 类型 | widget |
|---|---|---|
| 单行文本 | `string` | `input` |
| 多行文本 | `string` | `text_area` |
| 只读文本/结果 | `string` | `text` |
| 单选枚举 | `string` / `int` | `select` 或 `radio` |
| 多选候选项 | `string` / `[]string` / `[]int` | `multiselect` |
| 固定复选项 | `string` / `[]string` | `checkbox` |
| 自由输入数字数组 | `[]int` / `[]float64` | `list;item_type:number` |
| 自由输入文本数组 | `[]string` | `list;item_type:text` |
| 整数 | `int` / `int64` | `number` |
| 小数 | `float64` | `float` |
| 日期时间 | `types.Time` | `datetime` |
| 文件/图片/PDF/视频 | `string` | `files` |
| 用户 | `string` | `user` / `users` |
| 部门 | `string` | `department` / `departments` |
| 跳转链接 | `string` | `link` |
| 嵌套子表 | struct slice | `table` |
| 嵌套子表单 | struct | `form` |

## 数组字段

自由输入数组使用 `list`：

```go
Numbers []int    `json:"numbers" widget:"name:数字列表;type:list;item_type:number;placeholder:例如 1,2,3"`
Names   []string `json:"names" widget:"name:文本列表;type:list;item_type:text;placeholder:例如 张三,李四"`
```

候选项多选使用 `multiselect`：

```go
Tags string `json:"tags" widget:"name:标签;type:multiselect;options:紧急,重要;options_colors:F56C6C,E6A23C"`
```

## 枚举和校验

- 静态 `select` / `multiselect` 生成代码时带 `options_colors`，且数量必须和 `options` 一致。
- 动态 OnSelectFuzzy 下拉不写 `options`，也不要写 `options_colors`。
- `options_colors` 只允许 6 位十六进制 `RRGGBB`，不带 `#`，例如 `67C23A`；不要用语义色、`default`、`secondary`、`#67C23A` 或 `rgb(...)`。
- `validate:"oneof=..."`、`required_if` 等条件值必须和实际提交值一致。
- 条件规则引用 Go 字段名，不是 json 字段名。

## 搜索

- Table 搜索只使用 `eq`、`like`、`in`、`contains`、`gte,lte`。
- 不要生成 `gt`、`lt`、`not_eq`、`not_like`、`not_in`。
- Table Request 字段的 `json` 名不要和 AutoCrudTable / Response 表字段重名。

## 禁止生成

当前不要生成独立 `date`、`time`、`range`、`image`、`tag`、`tree`、`cascader`、`code` 等未支持 widget type。
