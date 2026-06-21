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

type Customer struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CustomerCode  string         `json:"customer_code" gorm:"column:customer_code" widget:"name:客户编码;type:input" validate:"required"`
	CustomerName  string         `json:"customer_name" gorm:"column:customer_name" widget:"name:客户名称;type:input" validate:"required"`
	ContactPerson string         `json:"contact_person" gorm:"column:contact_person" widget:"name:联系人;type:input"`
	ContactPhone  string         `json:"contact_phone" gorm:"column:contact_phone" widget:"name:联系电话;type:input"`
	CustomerType  string         `json:"customer_type" gorm:"column:customer_type" widget:"name:客户类型;type:select;options:企业客户,个人客户,散客;options_colors:409EFF,67C23A,909399;render_default:散客" validate:"required,oneof=企业客户 个人客户 散客"`
	Status        string         `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:正常,已停用;options_colors:67C23A,909399;render_default:正常" validate:"required,oneof=正常 已停用"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index;column:deleted_at" widget:"-"`
}

type CustomerListReq struct {
	CustomerName      string `json:"customer_name" form:"customer_name" widget:"name:客户名称;type:input"`
	CustomerType      string `json:"customer_type" form:"customer_type" widget:"name:客户类型;type:select;options:企业客户,个人客户,散客"`
	CreatedStart      string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd        string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

var CustomerTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "客户管理",
		Request:      &CustomerListReq{},
		CreateTables: []interface{}{&Customer{}},
	},
	AutoCrudTable: &Customer{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		var row Customer
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := ctx.GetGormDB().Create(&row).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		var updateFields Customer
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		if err := ctx.GetGormDB().Model(&Customer{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		if err := ctx.GetGormDB().Model(&Customer{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func CustomerList(ctx *app.Context, resp response.Response) error {
	var req CustomerListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Customer{})
	if req.CustomerName != "" {
		queryDB = queryDB.Where("customer_name LIKE ?", "%"+req.CustomerName+"%")
	}
	if req.CustomerType != "" {
		queryDB = queryDB.Where("customer_type = ?", req.CustomerType)
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
	var lists []*Customer
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
	packageContext.GET("customer_list.table", CustomerList, CustomerTemplate)
}
