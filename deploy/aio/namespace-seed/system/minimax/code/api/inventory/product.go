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

// 商品表 - 维护商品资料和基本信息
type Product struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	ProductName   string  `json:"product_name" gorm:"column:product_name" widget:"name:商品名称;type:input" validate:"required"`
	Category      string  `json:"category" gorm:"column:category" widget:"name:商品分类;type:select;options:饮料,零食,日用品,其他;options_colors:67C23A,FF9800,409EFF,9E9E9E" validate:"required"`
	Unit          string  `json:"unit" gorm:"column:unit" widget:"name:单位;type:select;options:瓶,箱,个,件;options_colors:67C23A,FF9800,409EFF,9E9E9E" validate:"required"`
	PurchasePrice float64 `json:"purchase_price" gorm:"column:purchase_price;type:decimal(10,2)" widget:"name:采购单价;type:float;min:0;precision:2;step:0.01;unit:元" validate:"required,gte=0"`
	RetailPrice   float64 `json:"retail_price" gorm:"column:retail_price;type:decimal(10,2)" widget:"name:零售单价;type:float;min:0;precision:2;step:0.01;unit:元" validate:"required,gte=0"`
	SafeStock     int     `json:"safe_stock" gorm:"column:safe_stock" widget:"name:安全库存;type:integer;min:0;step:1" validate:"required,gte=0"`
	ListingStatus string  `json:"listing_status" gorm:"column:listing_status" widget:"name:上架状态;type:select;options:上架,下架;options_colors:67C23A,F56C6C;render_default:上架" validate:"required,oneof=上架 下架"`
	CreatedBy     string  `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (Product) TableName() string {
	return "product"
}

var ProductTemplate = &app.TableTemplate{
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
			logger.Errorf(ctx, "[系统错误]-[ProductAdd] 创建商品失败, req: %+v, err: %v", req, err)
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
		err := db.Model(&Product{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[ProductUpdate] 更新商品失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&Product{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
			"deleted_at": time.Now(),
		}).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[ProductDelete] 删除商品失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type ProductListReq struct {
	ProductName       string `json:"product_name" form:"product_name" widget:"name:商品名称;type:input"`
	Category          string `json:"category" form:"category" widget:"name:商品分类;type:select;options:饮料,零食,日用品,其他;options_colors:67C23A,FF9800,409EFF,9E9E9E"`
	ListingStatus     string `json:"listing_status" form:"listing_status" widget:"name:上架状态;type:select;options:上架,下架;options_colors:67C23A,F56C6C"`
	CreatorFilter     string `json:"creator_filter" form:"creator_filter" widget:"name:创建人;type:user"`
	CreatedAtStart    string `json:"created_at_start" form:"created_at_start" widget:"name:创建开始时间;type:datetime"`
	CreatedAtEnd      string `json:"created_at_end" form:"created_at_end" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

func ProductList(ctx *app.Context, resp response.Response) error {
	var req ProductListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[系统错误]-[ProductList] 绑定参数失败, err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Product{})
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name LIKE ?", "%"+req.ProductName+"%")
	}
	if req.Category != "" {
		queryDB = queryDB.Where("category = ?", req.Category)
	}
	if req.ListingStatus != "" {
		queryDB = queryDB.Where("listing_status = ?", req.ListingStatus)
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
		logger.Errorf(ctx, "[系统错误]-[ProductList] 统计失败, err: %v", err)
		return err
	}
	var lists []*Product
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[ProductList] 查询失败, err: %v", err)
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("product_list.table", ProductList, ProductTemplate)
}
