# Table + Form + Chart 案例

本目录收录同时包含列表、表单和图表的综合案例，适合多模块联动和统计分析场景。

优先参考案例内的 `prd.json` 组织结构化 PRD；`prd.md` 和 Go 文件只作为实现参考。

图表实现时使用 SDK 的 `chart.LineChart` / `chart.BarChart` 等结构。普通数量图保持基础字段即可；需要补充单位、口径或展示提示时，优先放到 `Metadata` 或标题中，不要让前端或模型从系列名里猜单位。耗时图的 `Series.Data` 建议保持毫秒原始值，并在图表说明中标清单位。
