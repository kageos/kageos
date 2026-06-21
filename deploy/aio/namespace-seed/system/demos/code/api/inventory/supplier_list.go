package inventory

import (
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

type Supplier struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	SupplierCode  string         `json:"supplier_code" gorm:"column:supplier_code" widget:"name:供应商编码;type:input" validate:"required"`
	SupplierName  string         `json:"supplier_name" gorm:"column:supplier_name" widget:"name:供应商名称;type:input" validate:"required"`
	ContactPerson string         `json:"contact_person" gorm:"column:contact_person" widget:"name:联系人;type:input" validate:"required"`
	ContactPhone  string         `json:"contact_phone" gorm:"column:contact_phone" widget:"name:联系电话;type:input" validate:"required"`
	Address       string         `json:"address" gorm:"column:address" widget:"name:地址;type:input"`
	Status        string         `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:合作中,已停用;options_colors:67C23A,909399;render_default:合作中" validate:"required,oneof=合作中 已停用"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index;column:deleted_at" widget:"-"`
}

type SupplierListReq struct {
	SupplierName      string `json:"supplier_name" form:"supplier_name" widget:"name:供应商名称;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:合作中,已停用"`
	CreatedStart      string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd        string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

var SupplierTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "供应商管理",
		Request:      &SupplierListReq{},
		CreateTables: []interface{}{&Supplier{}},
	},
	AutoCrudTable: &Supplier{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		var row Supplier
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := ctx.GetGormDB().Create(&row).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		var updateFields Supplier
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		if err := ctx.GetGormDB().Model(&Supplier{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		if err := ctx.GetGormDB().Model(&Supplier{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func SupplierList(ctx *app.Context, resp response.Response) error {
	var req SupplierListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Supplier{})
	if req.SupplierName != "" {
		queryDB = queryDB.Where("supplier_name LIKE ?", "%"+req.SupplierName+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.CreatedStart != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedStart)
	}
	if req.CreatedEnd != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedEnd)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []*Supplier
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("supplier_list.table", SupplierList, SupplierTemplate)
}
