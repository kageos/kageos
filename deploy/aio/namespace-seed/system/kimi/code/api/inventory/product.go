package inventory

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 商品管理 ================

type Product struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Name      string  `json:"name" gorm:"column:name" widget:"name:商品名称;type:input" validate:"required"`
	Category  string  `json:"category" gorm:"column:category" widget:"name:分类;type:select;options:饮料,零食,日用品,其他;options_colors:409EFF,67C23A,E6A23C,909399" validate:"required"`
	Price     float64 `json:"price" gorm:"column:price" widget:"name:单价;type:float;precision:2;step:0.01;unit:元" validate:"required"`
	Unit      string  `json:"unit" gorm:"column:unit" widget:"name:单位;type:select;options:瓶,包,个,箱;options_colors:409EFF,67C23A,E6A23C,909399" validate:"required"`
	StockQty  int     `json:"stock_qty" gorm:"column:stock_qty;default:0" widget:"name:库存数量;type:integer"`
	CreatedBy string  `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (Product) TableName() string {
	return "inventory_product"
}

type ProductListReq struct {
	Name      string `json:"name" form:"name" widget:"name:商品名称;type:input"`
	Category  string `json:"category" form:"category" widget:"name:分类;type:select;options:饮料,零食,日用品,其他;options_colors:409EFF,67C23A,E6A23C,909399"`
	Creator   string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

func ProductList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req ProductListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&Product{})
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Category != "" {
		queryDB = queryDB.Where("category = ?", req.Category)
	}
	if req.Creator != "" {
		queryDB = queryDB.Where("created_by = ?", req.Creator)
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []Product
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var ProductListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "商品管理",
		Request:      &ProductListReq{},
		CreateTables: []interface{}{&Product{}},
	},
	AutoCrudTable: &Product{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row Product
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		row.CreatedBy = ctx.GetRequestUser()
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "Create product err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields Product
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		if err := db.Model(&Product{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "Update product err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		if err := db.Model(&Product{}).Delete(&Product{}, "id in ?", req.GetIds()).Error; err != nil {
			logger.Errorf(ctx, "Delete product err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("product_list.table", ProductList, ProductListTemplate)
}
