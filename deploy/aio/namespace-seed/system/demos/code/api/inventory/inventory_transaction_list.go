package inventory

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

type InventoryTransaction struct {
	ID              int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	TransactionNo   string     `json:"transaction_no" gorm:"column:transaction_no" widget:"name:流水号;type:input" hide:"create,update"`
	ProductID       int        `json:"product_id" gorm:"column:product_id" widget:"name:商品ID;type:integer" hide:"create,update"`
	ProductName     string     `json:"product_name" gorm:"column:product_name" widget:"name:商品名称;type:input" hide:"create,update"`
	ProductCode     string     `json:"product_code" gorm:"column:product_code" widget:"name:商品编码;type:input" hide:"create,update"`
	TransactionType string     `json:"transaction_type" gorm:"column:transaction_type" widget:"name:变动类型;type:select;options:采购入库,销售出库;options_colors:67C23A,F56C6C" hide:"create,update"`
	Quantity        int        `json:"quantity" gorm:"column:quantity" widget:"name:变动数量;type:integer" hide:"create,update"`
	BeforeStock     int        `json:"before_stock" gorm:"column:before_stock" widget:"name:变动前库存;type:integer" hide:"create,update"`
	AfterStock      int        `json:"after_stock" gorm:"column:after_stock" widget:"name:变动后库存;type:integer" hide:"create,update"`
	RelatedOrderNo  string     `json:"related_order_no" gorm:"column:related_order_no" widget:"name:关联单号;type:input" hide:"create,update"`
	TransactionTime types.Time `json:"transaction_time" gorm:"column:transaction_time;type:datetime" widget:"name:变动时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CreatedAt       types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
}

type InventoryTransactionListReq struct {
	ProductName       string `json:"product_name" form:"product_name" widget:"name:商品名称;type:input"`
	ProductCode       string `json:"product_code" form:"product_code" widget:"name:商品编码;type:input"`
	TransactionType   string `json:"transaction_type" form:"transaction_type" widget:"name:变动类型;type:select;options:采购入库,销售出库"`
	StartTime         string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime           string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

var InventoryTransactionTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "库存流水",
		Request:      &InventoryTransactionListReq{},
		CreateTables: []interface{}{&InventoryTransaction{}},
	},
	AutoCrudTable: &InventoryTransaction{},
}

func InventoryTransactionList(ctx *app.Context, resp response.Response) error {
	var req InventoryTransactionListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&InventoryTransaction{})
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name LIKE ?", "%"+req.ProductName+"%")
	}
	if req.ProductCode != "" {
		queryDB = queryDB.Where("product_code LIKE ?", "%"+req.ProductCode+"%")
	}
	if req.TransactionType != "" {
		queryDB = queryDB.Where("transaction_type = ?", req.TransactionType)
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("transaction_time >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("transaction_time <= ?", req.EndTime)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []*InventoryTransaction
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
	packageContext.GET("inventory_transaction_list.table", InventoryTransactionList, InventoryTransactionTemplate)
}
