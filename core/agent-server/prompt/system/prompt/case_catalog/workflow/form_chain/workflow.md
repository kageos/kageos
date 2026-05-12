# Form 串联工作流案例

## 编排目标

用户上传一个 PDF 文件，工作流先调用文本提取 Form，再把提取结果传给摘要 Form，最终只输出摘要。

## 搜索确认

真实项目中必须先确认：

- 第一个 Form 的 `full_code_path`、请求字段和响应字段。
- 第二个 Form 的 `full_code_path`、请求字段和响应字段。
- 文件字段是否用 `files` 类型透传。
- 上一步输出字段是否能被 `$ref` 直接引用。

## 映射逻辑

- `input.source_file` 映射到第一个节点的 `上传PDF文件`。
- `steps.extractText.output.提取的文本` 映射到第二个节点的 `待总结文本`。
- `input.summary_style` 映射到第二个节点的 `摘要风格`。
- 最终输出只暴露 `steps.generateSummary.output.摘要`。

## 自检清单

- `nodes` 有两个节点，`edges` 有一条线。
- `extractText -> generateSummary` 是唯一链路。
- 每个 `form.submit` 节点都有真实 `.form` 路径。
- 每个必填字段都有 `$ref` 或 `$const` 来源。
- `outputs` 不暴露中间大段文本，避免最终结果噪声过大。

## 扩展边界

如果用户还要求把摘要写入表格、发送消息或每天自动执行，当前案例不能直接扩展为可运行 JSON。应把这些记录为缺口，等 `table.create`、消息节点或定时触发接入后再升级。
