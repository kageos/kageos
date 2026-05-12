# Form 串联工作流案例

## 编排目标

用户上传一个 PDF 文件，工作流从 `workflow.start` 收集文件和摘要风格，先调用文本提取 Form，再把提取结果传给摘要 Form，最后由 `workflow.output` 输出摘要。

## 搜索确认

真实项目中必须先确认：

- 第一个 Form 的 `full_code_path`、请求字段 code 和响应字段 code。
- 第二个 Form 的 `full_code_path`、请求字段 code 和响应字段 code。
- 文件字段是否用 `files` 类型透传。
- 上一步输出字段是否能被 `$ref` 直接引用。

## 映射逻辑

- `workflow.start.schema.form.request` 声明 `source_file` 和 `summary_style`。
- `input.source_file` 映射到第一个节点的 `pdf_file`。
- `steps.extractText.output.text` 映射到第二个节点的 `source_text`。
- `input.summary_style` 映射到第二个节点的 `summary_style`。
- `workflow.output.input.summary` 映射 `steps.generateSummary.output.summary`。

## 自检清单

- `nodes` 有 `workflow.start`、两个 `form.submit`、`workflow.output` 四个节点。
- `start -> extractText -> generateSummary -> output` 是唯一链路。
- 每个 `form.submit` 节点都有真实 `.form` 路径。
- 每个必填字段都有 `$ref` 或 `$const` 来源。
- `workflow.output.schema.form.response` 只声明最终要给用户看的字段。

## 扩展边界

如果用户还要求把摘要写入表格、发送消息或每天自动执行，当前案例不能直接扩展为可运行 JSON。应把这些记录为缺口，等 `table.create`、消息节点或定时触发接入后再升级。
