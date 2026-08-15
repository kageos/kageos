# Table CRUD 最佳实践案例

适用：用户要求创建业务台账、记录列表、资产清单、预约记录、配置列表等可增删改查能力。

## 推荐文件布局

```text
code/api/<package>/
├── init_.go
└── item_list.go
code/cmd/app/main.go
```

落地时把 `sample`、`Item`、`sample_item` 和路由替换成真实业务名称。

## 核心模式

```go
package sample

import (
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"gorm.io/gorm"
)

type Item struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`
	Name      string         `json:"name" gorm:"column:name" widget:"name:名称;type:input" validate:"required"`
	Status    string         `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待处理,进行中,完成;render_default:待处理"`
	Owner     string         `json:"owner" gorm:"column:owner" widget:"name:负责人;type:user"`
}

func (Item) TableName() string { return "sample_item" }

type ItemListReq struct {
	Name   string `json:"name" form:"name" widget:"name:名称;type:input"`
	Status string `json:"status" form:"status" widget:"name:状态;type:select;options:待处理,进行中,完成"`
	query.PageSortReq `widget:"-"`
}

func ItemList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req ItemListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	q := db.Model(&Item{})
	if req.Name != "" {
		q = q.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		q = q.Order(order)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return err
	}
	var rows []Item
	if err := q.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var ItemListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "示例台账",
		Request:      &ItemListReq{},
		CreateTables: []interface{}{&Item{}},
	},
	AutoCrudTable: &Item{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row Item
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := db.Create(&row).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		if err := db.Model(&Item{}).Where("id = ?", req.GetId()).Updates(req.ChangedFields()).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		if err := db.Model(&Item{}).Where("id in ?", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"deleted_by": ctx.GetRequestUser(),
		}).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("item_list.table", ItemList, ItemListTemplate)
}
```

## 质量清单

- Request 显式包含筛选字段和 `query.PageSortReq`。
- Handler 先 `Count` 再 `Offset/Limit/Find`，返回 `response.TableResult`。
- 可编辑表必须设置 `CreateTables` 和 `AutoCrudTable`，不要依赖 SDK 用 `CreateTables` 第一个表推断。
- `CreateTables` 只列当前函数真实读写或迁移需要的表；多表业务直接写最小列表，不要为了省代码封装成“全包所有表”。
- 用户、多人、时间、选择器字段优先用 widget tag 表达，不要用裸字符串。
- 新 package 必须在 `code/cmd/app/main.go` blank import。
