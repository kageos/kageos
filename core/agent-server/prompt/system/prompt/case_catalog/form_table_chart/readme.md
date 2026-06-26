# Table + Form + Chart 案例

本目录收录同时包含列表、表单和图表的综合案例，适合多模块联动和统计分析场景。

优先参考案例内的 `prd.json` 组织结构化 PRD；`prd.md` 和 Go 文件只作为实现参考。

图表实现时使用 SDK 的 `chart.LineChart` / `chart.BarChart` 等结构。`YAxis` 是可选配置，普通数量图可以不加；需要覆盖默认数字展示时，按需使用 `YAxis: &chart.AxisConfig{ValueFormat: ...}`，不要让前端或模型从系列名里猜单位。常用值包括 `chart.ValueFormatCompact`、`chart.ValueFormatPlain`、`chart.ValueFormatDurationMS`、`chart.ValueFormatPercent`。耗时图的 `Series.Data` 保持毫秒原始值，并用 `chart.ValueFormatDurationMS` 控制展示。
