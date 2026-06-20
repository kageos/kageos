package inventory

import (
	"fmt"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// 采购入库记录表 - 只读查看采购入库记录
type PurchaseInbound struct {
	ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	InboundNo     string     `json:"inbound_no" gorm:"column:inbound_no" widget:"name:入库单号;type:input" hide:"create,update"`
	InboundTime   types.Time `json:"inbound_time" gorm:"column:inbound_time;type:datetime" widget:"name:入库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	SupplierName  string     `json:"supplier_name" gorm:"column:supplier_name" widget:"name:供应商;type:input" hide:"create,update"`
	ProductName   string     `json:"product_name" gorm:"column:product_name" widget:"name:商品;type:input" hide:"create,update"`
	Quantity      int        `json:"quantity" gorm:"column:quantity" widget:"name:采购数量;type:integer" hide:"create,update"`
	PurchasePrice float64    `json:"purchase_price" gorm:"column:purchase_price;type:decimal(10,2)" widget:"name:采购单价;type:float;precision:2;unit:元" hide:"create,update"`
	Amount        float64    `json:"amount" gorm:"column:amount;type:decimal(10,2)" widget:"name:采购金额;type:float;precision:2;unit:元" hide:"create,update"`
	Warehouse     string     `json:"warehouse" gorm:"column:warehouse" widget:"name:入库仓库;type:input" hide:"create,update"`
	CreatedBy     string     `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (PurchaseInbound) TableName() string {
	return "purchase_inbound"
}

var PurchaseInboundTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "采购入库记录",
		Request:      &PurchaseInboundListReq{},
		CreateTables: []interface{}{&PurchaseInbound{}},
	},
	AutoCrudTable: &PurchaseInbound{},
}

type PurchaseInboundListReq struct {
	InboundNo         string `json:"inbound_no" form:"inbound_no" widget:"name:入库单号;type:input"`
	SupplierName      string `json:"supplier_name" form:"supplier_name" widget:"name:供应商;type:input"`
	ProductName       string `json:"product_name" form:"product_name" widget:"name:商品;type:input"`
	InboundTimeStart  string `json:"inbound_time_start" form:"inbound_time_start" widget:"name:入库开始时间;type:datetime"`
	InboundTimeEnd    string `json:"inbound_time_end" form:"inbound_time_end" widget:"name:入库结束时间;type:datetime"`
	CreatedBy         string `json:"created_by" form:"created_by" gorm:"-" widget:"name:创建人;type:user" hide:"create,update"`
	CreatedAtStart    string `json:"created_at_start" form:"created_at_start" widget:"name:创建开始时间;type:datetime"`
	CreatedAtEnd      string `json:"created_at_end" form:"created_at_end" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

func PurchaseInboundList(ctx *app.Context, resp response.Response) error {
	var req PurchaseInboundListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[系统错误]-[PurchaseInboundList] 绑定参数失败, err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&PurchaseInbound{})
	if req.InboundNo != "" {
		queryDB = queryDB.Where("inbound_no LIKE ?", "%"+req.InboundNo+"%")
	}
	if req.SupplierName != "" {
		queryDB = queryDB.Where("supplier_name LIKE ?", "%"+req.SupplierName+"%")
	}
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name LIKE ?", "%"+req.ProductName+"%")
	}
	if req.InboundTimeStart != "" {
		queryDB = queryDB.Where("inbound_time >= ?", req.InboundTimeStart)
	}
	if req.InboundTimeEnd != "" {
		queryDB = queryDB.Where("inbound_time <= ?", req.InboundTimeEnd)
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
		queryDB = queryDB.Order("created_at DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[PurchaseInboundList] 统计失败, err: %v", err)
		return err
	}
	var lists []*PurchaseInbound
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[PurchaseInboundList] 查询失败, err: %v", err)
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// GenerateInboundNo 生成入库单号 CG + 日期 + 序号
func GenerateInboundNo(db *gorm.DB) string {
	dateStr := fmt.Sprintf("%s", types.Time{}.Time().Format("20060102"))
	var count int64
	db.Model(&PurchaseInbound{}).Where("inbound_no LIKE ?", "CG"+dateStr+"%").Count(&count)
	seq := count + 1
	return fmt.Sprintf("CG%s%03d", dateStr, seq)
}

func init() {
	packageContext.GET("purchase_inbound_list.table", PurchaseInboundList, PurchaseInboundTemplate)
}
