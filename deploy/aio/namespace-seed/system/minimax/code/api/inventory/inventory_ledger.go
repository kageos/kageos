package inventory

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

// 库存台账表 - 查询各商品当前库存情况
type InventoryLedger struct {
	ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	ProductName string `json:"product_name" gorm:"column:product_name" widget:"name:商品名称;type:input" hide:"create,update"`
	Category    string `json:"category" gorm:"column:category" widget:"name:商品分类;type:input" hide:"create,update"`
	Unit        string `json:"unit" gorm:"column:unit" widget:"name:单位;type:input" hide:"create,update"`
	StockQty    int    `json:"stock_qty" gorm:"column:stock_qty" widget:"name:库存数量;type:integer" hide:"create,update"`
	SafeStock   int    `json:"safe_stock" gorm:"column:safe_stock" widget:"name:安全库存;type:integer" hide:"create,update"`
	StockStatus string `json:"stock_status" gorm:"column:stock_status" widget:"name:库存状态;type:select;options:正常,预警;options_colors:67C23A,E6A23C" hide:"create,update"`
	CreatedBy   string `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (InventoryLedger) TableName() string {
	return "inventory_ledger"
}

var InventoryLedgerTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "库存台账",
		Request:      &InventoryLedgerListReq{},
		CreateTables: []interface{}{&InventoryLedger{}},
	},
	AutoCrudTable: &InventoryLedger{},
}

type InventoryLedgerListReq struct {
	ProductName       string `json:"product_name" form:"product_name" widget:"name:商品名称;type:input"`
	Category          string `json:"category" form:"category" widget:"name:商品分类;type:select;options:饮料,零食,日用品,其他;options_colors:67C23A,FF9800,409EFF,9E9E9E"`
	StockStatus       string `json:"stock_status" form:"stock_status" widget:"name:库存状态;type:select;options:正常,预警;options_colors:67C23A,E6A23C"`
	CreatedBy         string `json:"created_by" form:"created_by" gorm:"-" widget:"name:创建人;type:user" hide:"create,update"`
	CreatedAtStart    string `json:"created_at_start" form:"created_at_start" widget:"name:创建开始时间;type:datetime"`
	CreatedAtEnd      string `json:"created_at_end" form:"created_at_end" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

func InventoryLedgerList(ctx *app.Context, resp response.Response) error {
	var req InventoryLedgerListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[系统错误]-[InventoryLedgerList] 绑定参数失败, err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&InventoryLedger{})
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name LIKE ?", "%"+req.ProductName+"%")
	}
	if req.Category != "" {
		queryDB = queryDB.Where("category = ?", req.Category)
	}
	if req.StockStatus != "" {
		queryDB = queryDB.Where("stock_status = ?", req.StockStatus)
	}
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}
	if req.CreatedAtStart != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedAtStart)
	}
	if req.CreatedAtEnd != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedAtEnd)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("product_name ASC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[InventoryLedgerList] 统计失败, err: %v", err)
		return err
	}
	var lists []*InventoryLedger
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[InventoryLedgerList] 查询失败, err: %v", err)
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("inventory_ledger_list.table", InventoryLedgerList, InventoryLedgerTemplate)
}
