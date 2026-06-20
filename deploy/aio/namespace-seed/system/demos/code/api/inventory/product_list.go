package inventory

import (
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type Product struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	ProductCode   string         `json:"product_code" gorm:"column:product_code" widget:"name:商品编码;type:input" validate:"required"`
	ProductName   string         `json:"product_name" gorm:"column:product_name" widget:"name:商品名称;type:input" validate:"required"`
	Category      string         `json:"category" gorm:"column:category" widget:"name:商品分类;type:select;options:电子产品,办公用品,食品饮料,日用百货,其他;options_colors:409EFF,67C23A,E6A23C,909399,F56C6C" validate:"required,oneof=电子产品 办公用品 食品饮料 日用百货 其他"`
	Spec          string         `json:"spec" gorm:"column:spec" widget:"name:规格型号;type:input"`
	Unit          string         `json:"unit" gorm:"column:unit" widget:"name:单位;type:select;options:个,箱,件,瓶,包,千克" validate:"required,oneof=个 箱 件 瓶 包 千克"`
	PurchasePrice float64        `json:"purchase_price" gorm:"column:purchase_price" widget:"name:采购单价;type:float;precision:2;unit:元" validate:"required,min=0"`
	SalePrice     float64        `json:"sale_price" gorm:"column:sale_price" widget:"name:销售单价;type:float;precision:2;unit:元" validate:"required,min=0"`
	CurrentStock  int            `json:"current_stock" gorm:"column:current_stock" widget:"name:当前库存;type:integer;min:0" validate:"required,min=0"`
	SafetyStock   int            `json:"safety_stock" gorm:"column:safety_stock" widget:"name:安全库存;type:integer;min:0" validate:"required,min=0"`
	Status        string         `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:正常,停用;options_colors:67C23A,909399;render_default:正常" validate:"required,oneof=正常 停用"`
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index;column:deleted_at" widget:"-"`
}

type ProductListReq struct {
	ProductCode       string `json:"product_code" form:"product_code" widget:"name:商品编码;type:input"`
	ProductName       string `json:"product_name" form:"product_name" widget:"name:商品名称;type:input"`
	Category          string `json:"category" form:"category" widget:"name:商品分类;type:select;options:电子产品,办公用品,食品饮料,日用百货,其他"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,停用"`
	CreatedStart      string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd        string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

var ProductTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "商品管理",
		Request:      &ProductListReq{},
		CreateTables: []interface{}{&Product{}},
	},
	AutoCrudTable: &Product{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		var row Product
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		if err := ctx.GetGormDB().Create(&row).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		var updateFields Product
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		if err := ctx.GetGormDB().Model(&Product{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		if err := ctx.GetGormDB().Model(&Product{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func ProductList(ctx *app.Context, resp response.Response) error {
	var req ProductListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Product{})
	if req.ProductCode != "" {
		queryDB = queryDB.Where("product_code = ?", req.ProductCode)
	}
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name LIKE ?", "%"+req.ProductName+"%")
	}
	if req.Category != "" {
		queryDB = queryDB.Where("category = ?", req.Category)
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
	var lists []*Product
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
	packageContext.GET("product_list.table", ProductList, ProductTemplate)
}
