# 应用修改二级分类索引

`app.modify` 进入后，先判断修改类型，再读取对应专项文档。

| 修改类型 | 文档 |
| --- | --- |
| 字段改名 | `/system/prompt/intents/modify/field-rename` |
| 新增字段 | `/system/prompt/intents/modify/add-field` |
| 删除字段 | `/system/prompt/intents/modify/remove-field` |
| 修改 widget | `/system/prompt/intents/modify/widget-change` |
| 新增 select 选项 | `/system/prompt/intents/modify/select-options` |
| 修改搜索条件 | `/system/prompt/intents/modify/search-filter` |
| 新增 OnSelectFuzzy | `/system/prompt/intents/modify/onselect-fuzzy` |
| 新增 Table 回调逻辑 | `/system/prompt/intents/modify/table-callback` |
| 新增消息通知 | `/system/prompt/intents/modify/send-message` |
| 新增 link 跳转 | `/system/prompt/intents/modify/function-link` |
| 新增 Form/Table/Chart | `/system/prompt/intents/modify/add-function` |
| 修改 Chart 指标 | `/system/prompt/intents/modify/chart-metric` |
| 修业务 bug | `/system/prompt/intents/modify/bugfix` |

通用要求：读取相关文件，小改用 `search_replace_file`，完整落盘后统一 build，build 成功后进入 `app.operate_test` 验证。
