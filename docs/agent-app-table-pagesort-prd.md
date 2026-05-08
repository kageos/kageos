# Agent-App Table 分页查询链路 PRD：PageSortReq

## 1. 背景

Agent-App 的表格列表需要同时服务三类场景：

- 普通 CRUD 列表：分页、排序、按表单字段筛选。
- 关联表列表：先 Join 或 Preload，再分页返回。
- 计算字段列表：筛选条件不能直接依赖返回字段，需要在 Handler 中显式写 Where。

最新版链路只保留一个默认写法：Request 显式声明筛选字段，分页排序使用 `query.PageSortReq`，返回使用 `resp.Table(..., ...).Build()`。

Table 列表代码只保留一条正向路径：筛选字段在 Request 中显式声明，Handler 中显式转成查询条件，分页排序交给 `query.PageSortReq`。

## 2. 目标

- 生成代码默认使用 `query.PageSortReq`。
- 表格筛选字段只来自 Request 结构体，不从 Response Model 推导。
- Model 字段只描述落库、展示和编辑能力；Table Template 用 `AutoCrudTable` 声明列表模型。
- URL 查询参数直接使用 Request 字段名，例如 `status=处理中&title=合同&page=1&page_size=20&sorts=-created_at`；数组形态可用 `sort[]=-created_at&sort[]=score`。
- Handler 在 `Build()` 前完成所有筛选、Join、Preload、权限约束和默认条件。
- 文档、prompt、示例、前端运行时统一使用同一套表达。

## 3. 非目标

- 不再维护另一套自动推导协议。
- 不再在 Model 字段上声明表格筛选能力。
- 不再让前端发操作符分组参数。
- 不再要求 Request 字段与 Model 字段隔离；字段 code 可以一致，语义由 Handler 控制。

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

	var rows []Ticket
	if err := resp.Table(&rows, db, &Ticket{}, &req.PageSortReq).Build(); err != nil {
		return err
	}
	return nil
}
```

## 5. 生成规则

### 5.1 Request

- 每个可筛选项都写在 Request。
- Request 字段必须带 `json` 和 `form`。
- Request 字段用 `widget` 决定前端控件。
- 分页排序只嵌入 `query.PageSortReq`，并加 `widget:"-"`。

### 5.2 Handler

- `Table` 只负责分页、排序和执行查询。
- 所有业务筛选都写在 `Table` 之前。
- 关联表筛选用 Join 或先查 ID 再 `Where IN`。
- 计算字段筛选转成真实字段条件。
- 返回前需要补充展示字段时，在 `Build()` 之后遍历 rows。

### 5.3 Response Model

- Response Model 只描述返回列、展示组件、表单编辑能力和跳转链接。
- 不承载列表筛选协议。
- Table Template 通过 `AutoCrudTable` 指定 schema 来源。
- 审计字段仍按统一 widget/hide 规则展示或隐藏。

### 5.4 URL

前端 URL 与后端 Request 一一对应：

```text
?status=处理中&handler=alice&page=1&page_size=20&sorts=-created_at
```

数组排序参数同样会进入 `PageSortReq`：

```text
?status=处理中&page=1&page_size=20&sort[]=-created_at&sort[]=score
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
-> resp.Table(..., ...).Build()
```

示例、技能、工具说明和校验提示必须保持同一套表达。

## 8. 验收

- SDK 中 `Table` 不再暴露额外分页链式方法。
- 表格 schema 的筛选字段只来自 Request。
- 前端 table URL 只发 `field=value`。
- 生成提示不再出现另一套表格筛选协议。
- 示例、技能、工具说明和校验提示保持同一套表达。
