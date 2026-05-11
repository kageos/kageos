# Agent-App Table 分页查询链路 PRD：PageSortReq

## 1. 背景

Agent-App 的表格列表需要同时服务三类场景：

- 普通 CRUD 列表：分页、排序、按表单字段筛选。
- 关联表列表：先 Join 或 Preload，再分页返回。
- 计算字段列表：筛选条件不能直接依赖返回字段，需要在 Handler 中显式写 Where。

最新版链路只保留一个默认写法：Request 显式声明筛选字段，分页排序使用 `query.PageSortReq`，Handler 显式拿到当前页数据和总数，返回使用 `resp.Table(response.TableResult{...}).Build()`。

Table 列表代码只保留一条正向路径：筛选字段在 Request 中显式声明，Handler 中显式转成查询条件，分页排序交给 `query.PageSortReq`。

## 2. 目标

- 生成代码默认使用 `query.PageSortReq`。
- 表格筛选字段只来自 Request 结构体，不从 Response Model 推导。
- Model 字段只描述落库、展示和编辑能力；Table Template 用 `AutoCrudTable` 声明列表模型。
- URL 查询参数直接使用 Request 字段名，例如 `status=处理中&title=合同&page=1&page_size=20&sorts=[{"field":"created_at","order":"desc"}]`；排序参数统一使用结构化 JSON。
- Handler 在 `Build()` 前完成所有筛选、Join、Preload、权限约束和默认条件。
- 文档、prompt、示例、前端运行时统一使用同一套表达。

## 3. 设计边界

- 表格筛选能力只由 Request 字段表达。
- Model 字段只描述数据结构、展示组件、编辑能力和跳转链接。
- 前端分页排序参数固定为 `page`、`page_size`、`sorts`，业务筛选参数直接使用 Request 字段名。
- Request 字段与 Model 字段可以同名，语义由 Handler 控制。

## 4. 推荐代码形态

```go
type TicketListReq struct {
	query.PageSortReq `widget:"-"`
	Title         string `json:"title" form:"title" widget:"name:标题;type:input"`
	Status        string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,处理中,已完成"`
    Handler       string `json:"handler" form:"handler" widget:"name:处理人;type:user"`
}

func TicketList(ctx *app.Context, resp response.Response) error {
	var req TicketListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB().Model(&Ticket{})

	if req.Title != "" {
		db = db.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.Status != "" {
		db = db.Where("status = ?", req.Status)
	}
	if req.Handler != "" {
		db = db.Where("handler = ?", req.Handler)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		db = db.Order(order)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return err
	}

	var rows []Ticket
	if err := db.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      rows,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}
```

## 5. 生成规则

### 5.1 Request

- 每个可筛选项都写在 Request。
- Request 字段必须带 `json` 和 `form`。
- Request 字段用 `widget` 决定前端控件。
- 分页排序只嵌入 `query.PageSortReq`，并加 `widget:"-"`。

### 5.2 Handler

- `PageSortReq` 只负责分页、排序参数和 `GetOrder/GetOffset/GetLimit` 辅助方法。
- Handler 负责执行查询并拿到 `rows` 和 `total`。
- `resp.Table(response.TableResult{...})` 只负责渲染响应，不查询数据库。
- 关联表筛选用 Join 或先查 ID 再 `Where IN`。
- 计算字段筛选转成真实字段条件。
- 返回前需要补充展示字段时，在 `Build()` 之前遍历 rows。

### 5.3 Response Model

- Response Model 只描述返回列、展示组件、表单编辑能力和跳转链接。
- 不承载列表筛选协议。
- Table Template 通过 `AutoCrudTable` 指定 schema 来源。
- 审计字段仍按统一 widget/hide 规则展示或隐藏。

### 5.4 URL

前端 URL 与后端 Request 一一对应：

```text
?status=处理中&handler=alice&page=1&page_size=20&sorts=[{"field":"created_at","order":"desc"}]
```

- `page`、`page_size`、`sorts` 是表格控制参数。
- 其他业务参数必须来自 Request 字段。
- 多值控件用逗号分隔。
- 平台状态使用 `_` 前缀，不传入用户函数。

## 6. 前端运行时

- 搜索区只读取 Table Request fields。
- URL 恢复只恢复 Request 字段。
- 写 URL 时不生成字段命名空间、不生成显示值伴随参数。
- URL 只维护当前 Request 字段和表格控制参数。

## 7. 文档与 Prompt

所有面向代码生成的材料只描述以下路径：

```text
Request 显式筛选字段
-> Handler 写 Where / Join / Preload
-> query.PageSortReq
-> Handler 显式 Count/Find 或调用第三方 API
-> resp.Table(response.TableResult{...}).Build()
```

示例、技能、工具说明和校验提示必须保持同一套表达。

## 8. 验收

- SDK 中 `Table` 不再暴露额外分页链式方法。
- 表格 schema 的筛选字段只来自 Request。
- 前端 table URL 只发 `field=value`。
- 生成提示只描述本链路中的 Request、Handler、PageSortReq、TableResult 写法。
- 示例、技能、工具说明和校验提示保持同一套表达。
