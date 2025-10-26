在更新app的时候我们触发了update的回调，然后我们要做api的diff，diff的时候会解析新增的api，对新增api的字段要进行解析，例如先看一下我们的table的代码


```go

package crm

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/query"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"gorm.io/gorm"
)

type CrmTicket struct {
	// 框架标签：runner:"name:工单ID" - 设置字段在前端的显示名称
	// 框架标签：permission:"read" - 字段只读权限（不能编辑）
	// 注意：gorm:"column:id" 明确指定数据库列名，确保映射正确
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" permission:"read"` //在table 表格里只读，不能编辑

	// 框架标签：widget:"type:timestamp;format:YYYY-MM-DD HH:mm:ss" - 日期时间选择器组件
	// 注意：gorm:"autoCreateTime:milli" 自动填充创建时间（毫秒级时间戳，必须是毫秒级别）
	CreatedAt int64 `json:"created_at" gorm:"autoCreateTime:milli;column:created_at"  widget:"name:创建时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`

	// 框架标签：widget:"type:timestamp;format:YYYY-MM-DD HH:mm:ss" - 日期时间选择器组件，（毫秒级时间戳，必须是毫秒级别）
	UpdatedAt int64 `json:"updated_at" gorm:"autoUpdateTime:milli;column:updated_at" widget:"name:更新时间;type:timestamp;format:YYYY-MM-DD HH:mm:ss" permission:"read"`

	// 框架标签：widget:"-" - 隐藏字段（不在前端显示）
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"` //不做展示

	// 框架标签：widget:"type:input" - 文本输入框组件
	// 框架标签：search:"like" - 启用模糊搜索功能
	// 框架标签：validate:"required,min=2,max=200" - 必填字段，长度2-200字符
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" search:"like" validate:"required,min=2,max=200"` //该字段支持模糊搜索，同时新增时候前端会验证validate，后端sdk内部也会验证

	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	// 框架标签：validate:"required,min=10" - 必填字段，至少10字符
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`

	// 框架标签：widget:"type:select;options:低,中,高;default:中" - 下拉选择组件（选项：低/中/高）
	// 框架标签：validate:"required,oneof=低,中,高" - 必填字段，值必须是选项之一
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;default:中" validate:"required,oneof=低,中,高"`

	// 框架标签：widget:"type:select;options:待处理,处理中,已完成,已关闭;default:待处理" - 下拉选择组件
	// 框架标签：validate:"required,oneof=待处理,处理中,已完成,已关闭" - 值必须是有效状态
	Status string `json:"status" gorm:"column:status"  widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;default:待处理" validate:"required,oneof=待处理,处理中,已完成,已关闭"`

	// 框架标签：validate:"required,min=11,max=20" - 必填字段，长度11-20字符
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`

	// 框架标签：widget:"type:text_area" - 多行文本区域组件
	Remark string `json:"remark" gorm:"column:remark"  widget:"name:备注;type:text_area"`

	// 创建用户：用户组件
	CreateBy string `json:"create_by" gorm:"column:create_by" widget:"name:创建用户;type:user" permission:"read"` //read 表示只读，表示要后端赋值的，非read的字段前端界面会自动渲染成用户选择器进行选择
}

func (t *CrmTicket) TableName() string {
	return "crm_ticket"
}

var CrmTicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "工单管理",
		Tags:     []string{"工单管理系统"},
		Desc:     "一个简单的工单管理系统 ........",
		Request:  &CrmTicketSearchReq{},
		Response: []*CrmTicket{},
		CreateTables: []interface{}{
			&CrmTicket{},
		},
	},
	AutoCrudTable: &CrmTicket{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row CrmTicket
		if err := ctx.ShouldBindValidate(&row); err != nil { //这里内部会用validate的库验证validate的标签
			return nil, err
		}
		row.CreateBy = ctx.GetRequestUser() //获取请求用户
		err := db.Create(&row).Error
		if err != nil {
			return nil, err
		}
		//这里还没想好要返回什么有价值的信息，先留空吧
		return &callback.OnTableAddRowResp{}, nil
	},
	OnTableUpdateRows: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields CrmTicket
		if err := ctx.ShouldBind(&updateFields); err != nil { //这里不会验证validate，为啥？因为前端只传递了变更的字段，所以无需验证，所以updateFields只有 更新的字段才会有值，没更新的字段是零值
			return nil, err
		}
		//注意：updateFields主要是方便安全的操作变更的字段，如果更新数据我们还是配合用req.GetUpdates()来，这样例如某些字符串想更新成空，或者int想更新成0是可以实现的，
		//用updateFields 的话，gorm是无法更新零值的

		updates := req.GetUpdates()                                   //-> map[string]interface{} 这里值包含此次变更的字段，例如用户把status 变更成：“已完成”，那么这里的map就是只有一个 status:已完成
		err := db.Where("id = ?", req.GetId()).Updates(updates).Error //这是标准的更新方式
		if err != nil {
			return nil, err
		}
		//这里还没想好要返回什么有价值的信息，先留空吧
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Delete(&CrmTicket{}, "id in ?", req.GetIds()).Error
		if err != nil {
			return nil, err
		}

		//这里还没想好要返回什么有价值的信息，先留空吧
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type CrmTicketSearchReq struct {
	*query.SearchFilterPageReq //前端会传递符合框架规范的查询字符串，里面包含AutoCrudTable这里这张表的字段相关的 查询，排序，分页等等参数，后端无需关心这些
}

func CrmTicketSearch(ctx *app.Context, resp response.Response) error {
	var req CrmTicketSearchReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()

	var lists []*CrmTicket
	//直接把SearchFilterPageReq透传到框架里，框架可以直接处理内部的逻辑，最终返回的数据是lists，同时会包含分页的信息
	return resp.Table(lists).AutoSearchFilterPaged(db, &CrmTicket{}, req.SearchFilterPageReq).Build()
}

func init() {

	//这里系统在创建服务目录（package）的时候会自动创建WithCurrentRouterGroup函数，无需关心直接用即可
	app.GET(WithCurrentRouterGroup("crm_ticket"), CrmTicketSearch, CrmTicketTemplate)
}

```

我们其实request 要解析的是CrmTicketSearchReq和CrmTicket，CrmTicketSearchReq里
如果request里组合了*query.SearchFilterPageReq的话，直接跳过*query.SearchFilterPageReq的解析，我们例如
下面的这两个request的model，我们第一个应该解析为null，第二个只需要解析出self_only即可

```go

type CrmTicketSearchReq struct {
    *query.SearchFilterPageReq //前端会传递符合框架规范的查询字符串，里面包含AutoCrudTable这里这张表的字段相关的 查询，排序，分页等等参数，后端无需关心这些
}

type CrmTicketSearchReq struct {
	SelfOnly string `json:"self_only" widget:"name:只看我的;type:switch" data:"default_value:false"`

	*query.SearchFilterPageReq //前端会传递符合框架规范的查询字符串，里面包含AutoCrudTable这里这张表的字段相关的 查询，排序，分页等等参数，后端无需关心这些
}
```

然后response也就是DecodeTable的tableModel要解析的是AutoCrudTable的那个model，也就是CrmTicket

```go
func DecodeTable(request, tableModel interface{}) (requestFields []*Field, responseTableFields []*Field, err error) {
	
	//我们需要解析出每个字的的每个标签最终返回出[]*Field
	
	return nil, nil, nil
}
```