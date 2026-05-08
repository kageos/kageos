# SDK Form/Table/Chart 参考

本文档用于创建或修改 SDK 函数。Form/Table/Chart 的组合建模先读 `/system/prompt/platform-function-architecture`，完整 SDK 规则以 `/system/prompt/sdk/agent-app-sdk-readme` 为准。

## 选择

- Table：管理长期记录，前端渲染为 Element Plus 表格，支持列表、搜索、分页、新增、编辑、删除、批量导入。
- Form：一次性提交参数并返回结果，前端渲染为 Element Plus 表单，适合文件处理、业务动作、快速登记、工具函数。
- Chart：按条件查询并返回一张图表，前端渲染为筛选条件 + ECharts 图表。

## 路由和 Template

| 类型 | 路由后缀 | Template | 响应 |
|---|---|---|---|
| Table | `.table` | `app.TableTemplate` | `resp.Table(...).Build()` |
| Form | `.form` | `app.FormTemplate` | `resp.Form(...).Build()` |
| Chart | `.chart` | `app.ChartTemplate` | `resp.Chart(...).Build()` |

启动期 `CompileAndValidate()` 会检查路由后缀和 Template 类型是否匹配。

路由后缀不是 Go 文件名后缀。业务文件只用普通 `.go` 命名，例如 `order_list.go`、`order_submit.go`、`sales_trend.go`；路由类型只写在注册字符串里，例如 `packageContext.GET("order_list.table", ...)`、`packageContext.POST("order_submit.form", ...)`。不要把 `.table`、`.form` 或 `.chart` 再拼到 `.go` 前面。

## Table 要点

1. Model 字段写 `gorm`、`json`、`widget`、`validate`；Table 筛选字段写在 Request 中。
2. `AutoCrudTable` 指向列表 Model。
3. Request 字段的 `json` 名不要和 AutoCrudTable / Response 表字段重名。
4. Table 筛选字段写在 Request 中，Handler Build 前手写查询条件。
5. 不需要某类操作就不要配置对应回调；PRD 或示例列表里承诺的“审核/隐藏/回复/发布/下架”等行操作，必须和 `OnTableUpdateRow` 或 link 字段对应。
6. 支付记录、操作日志、流水类表默认只读。
7. List 函数 Build 前做 `Where/Preload`，Build 后填 link、计算字段、展示字段。

## Form 要点

1. Request/Response 是普通表单或结果结构体。
2. 使用 `ctx.ShouldBindValidate(&req)`。
3. 返回 `resp.Form(&out).Build()`。
4. 不要在 Form Request/Response 中使用 Chart 结构体。

## Chart 要点

1. 一个 Chart 路由返回一张图。
2. 使用 `sdk/agent-app/chart` 包。
3. `ChartTemplate.Response` 填具体图表类型，如 `&chart.LineChart{}`。
4. 多系列趋势或对比直接用 `Series` 数组。
5. Handler 传给 `resp.Chart(...)` 的必须是 SDK chart 包里的具体图表对象；不要传自定义业务响应结构体。
6. 图表附加指标放在 `Metadata`；需要多张图就拆多个 `.chart` 路由。

## 注册要点

业务目录的 `init_.go` 会提供 `packageContext`。业务文件只用 `packageContext.GET(...)` / `packageContext.POST(...)` 注册路由；不要编造全局 `app.GET` / `app.POST`。

## 验证

写完代码后统一 `build_workspace`，再用 `run_table_search`、`run_form_submit` 或 `run_chart_query` 验证核心路径。
