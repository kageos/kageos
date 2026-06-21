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

// Product 商品
type Product struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	UpdatedBy string         `json:"updated_by" gorm:"column:updated_by" widget:"name:更新人;type:user" hide:"create,update"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	Name      string  `json:"name" gorm:"column:name" widget:"name:商品名称;type:input;placeholder:请输入商品名称" validate:"required,min=1,max=200"`
	Category  string  `json:"category" gorm:"column:category" widget:"name:商品分类;type:select;options:食品,饮料,日用品,电子,其他;options_colors:FF9800,2196F3,4CAF50,9C27B0,607D8B;render_default:食品" validate:"required,oneof=食品 饮料 日用品 电子 其他"`
	Spec      string  `json:"spec" gorm:"column:spec" widget:"name:规格;type:input;placeholder:如 500ml、大号" validate:"required,min=1,max=200"`
	Unit      string  `json:"unit" gorm:"column:unit" widget:"name:单位;type:select;options:个,箱,瓶,袋,公斤,件;options_colors:607D8B,795548,2196F3,4CAF50,FF9800,9E9E9E;render_default:个" validate:"required,oneof=个 箱 瓶 袋 公斤 件"`
	CostPrice float64 `json:"cost_price" gorm:"column:cost_price" widget:"name:采购价;type:float;min:0;precision:2;step:0.01;unit:元;render_default:0" validate:"required,min=0"`
	SellPrice float64 `json:"sell_price" gorm:"column:sell_price" widget:"name:销售价;type:float;min:0;precision:2;step:0.01;unit:元;render_default:0" validate:"required,min=0"`
	Stock     int     `json:"stock" gorm:"column:stock" widget:"name:当前库存;type:integer;min:0;step:1;unit:件;render_default:0" validate:"required,min=0"`
	Status    string  `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:正常,停用;options_colors:4CAF50,F56C6C;render_default:正常" validate:"required,oneof=正常 停用"`
}

// ProductListReq 商品列表查询请求
type ProductListReq struct {
	Name      string `json:"name" form:"name" widget:"name:商品名称;type:input"`
	Category  string `json:"category" form:"category" widget:"name:商品分类;type:select;options:食品,饮料,日用品,电子,其他;options_colors:FF9800,2196F3,4CAF50,9C27B0,607D8B"`
	Status    string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,停用;options_colors:4CAF50,F56C6C"`
	CreatedBy string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`

	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

var ProductTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "商品管理",
		Request:      &ProductListReq{},
		CreateTables: []interface{}{&Product{}},
	},
	AutoCrudTable:     &Product{},
	OnTableAddRow:     onProductAddRow,
	OnTableUpdateRow:  onProductUpdateRow,
	OnTableDeleteRows: onProductDeleteRows,
}

func onProductAddRow(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
	db := ctx.GetGormDB()
	var row Product
	if err := ctx.ShouldBindValidate(&row); err != nil {
		return nil, err
	}
	row.CreatedBy = ctx.GetRequestUser()
	row.UpdatedBy = ctx.GetRequestUser()
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	return &callback.OnTableAddRowResp{Data: &row}, nil
}

func onProductUpdateRow(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
	db := ctx.GetGormDB()
	var updateFields Product
	if err := req.BindChangedFields(&updateFields); err != nil {
		return nil, err
	}
	updates := req.ChangedFields()
	updates["updated_by"] = ctx.GetRequestUser()
	if err := db.Model(&Product{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &callback.OnTableUpdateRowResp{}, nil
}

func onProductDeleteRows(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
	db := ctx.GetGormDB()
	err := db.Model(&Product{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
		"deleted_by": ctx.GetRequestUser(),
		"deleted_at": time.Now(),
	}).Error
	if err != nil {
		return nil, err
	}
	return &callback.OnTableDeleteRowsResp{}, nil
}

// ProductList 商品列表
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
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
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

func init() {
	packageContext.GET("product_list.table", ProductList, ProductTemplate)
}
