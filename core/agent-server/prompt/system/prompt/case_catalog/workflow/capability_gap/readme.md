# 能力缺口识别案例

用户目标：每天读取新线索，筛选高价值客户，写入跟进表，并通知销售。

MVP 判断：

- `workflow.start -> form.submit -> form.submit -> workflow.output` 可以承载“线索打分 Form -> 跟进话术生成 Form”这一段。
- `table.search`、`table.create`、消息发送和定时触发不是当前 `workflow.v1` MVP 的可运行节点。
- 正式 JSON 不能伪造未来 executor；应先输出可运行子流程，再把缺口写入 `missing_capabilities`。

推荐输出：

```json
{
  "missing_capabilities": [
    "table.search: 从线索表读取待处理记录",
    "table.create: 把高价值线索写入跟进表",
    "message.send: 通知销售",
    "timer trigger: 每天自动触发 workflow"
  ]
}
```

推荐交接：

- 缺少线索打分或话术生成 Form 时，交接给 `product_manager` 或 `app_developer`。
- 需要改造已有 Form 的入参出参时，交接给 `maintenance_engineer`。
- 子流程稳定后需要每天执行时，交接给 `scheduler_engineer`。
