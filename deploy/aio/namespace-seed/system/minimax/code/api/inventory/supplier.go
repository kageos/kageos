package inventory

import (
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// 供应商表 - 维护供应商信息
type Supplier struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	SupplierName string `json:"supplier_name" gorm:"column:supplier_name" widget:"name:供应商名称;type:input" validate:"required"`
	Contact      string `json:"contact" gorm:"column:contact" widget:"name:联系人;type:input" validate:"required"`
	Phone        string `json:"phone" gorm:"column:phone" widget:"name:手机;type:input" validate:"required"`
	Address      string `json:"address" gorm:"column:address" widget:"name:地址;type:input"`
	Status       string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:正常,停用;options_colors:67C23A,F56C6C;render_default:正常" validate:"required,oneof=正常 停用"`
	CreatedBy    string `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (Supplier) TableName() string {
	return "supplier"
}

var SupplierTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "供应商管理",
		Request:      &SupplierListReq{},
		CreateTables: []interface{}{&Supplier{}},
	},
	AutoCrudTable: &Supplier{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row Supplier
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		row.CreatedBy = ctx.GetRequestUser()
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[SupplierAdd] 创建供应商失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields Supplier
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		err := db.Model(&Supplier{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[SupplierUpdate] 更新供应商失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&Supplier{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
			"deleted_at": time.Now(),
		}).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[SupplierDelete] 删除供应商失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type SupplierListReq struct {
	SupplierName      string `json:"supplier_name" form:"supplier_name" widget:"name:供应商名称;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,停用;options_colors:67C23A,F56C6C"`
	CreatorFilter     string `json:"creator_filter" form:"creator_filter" widget:"name:创建人;type:user"`
	CreatedAtStart    string `json:"created_at_start" form:"created_at_start" widget:"name:创建开始时间;type:datetime"`
	CreatedAtEnd      string `json:"created_at_end" form:"created_at_end" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

func SupplierList(ctx *app.Context, resp response.Response) error {
	var req SupplierListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[系统错误]-[SupplierList] 绑定参数失败, err: %v", err)
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
	if req.CreatorFilter != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatorFilter)
	}
	if req.CreatedAtStart != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedAtStart)
	}
	if req.CreatedAtEnd != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedAtEnd)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[SupplierList] 统计失败, err: %v", err)
		return err
	}
	var lists []*Supplier
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[SupplierList] 查询失败, err: %v", err)
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
