package inventory

import (
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 销售记录（只读） ================

type SalesRecord struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	OrderNo      string     `json:"order_no" gorm:"column:order_no" widget:"name:销售单号;type:input"`
	ProductName  string     `json:"product_name" gorm:"column:product_name" widget:"name:商品名称;type:input"`
	Qty          int        `json:"qty" gorm:"column:qty" widget:"name:数量;type:integer"`
	Price        float64    `json:"price" gorm:"column:price" widget:"name:单价;type:float;precision:2;unit:元"`
	TotalAmount  float64    `json:"total_amount" gorm:"column:total_amount" widget:"name:实付金额;type:float;precision:2;unit:元"`
	OutboundTime types.Time `json:"outbound_time" gorm:"column:outbound_time;type:datetime" widget:"name:出库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedBy    string     `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (SalesRecord) TableName() string {
	return "inventory_sales_record"
}

type SalesRecordListReq struct {
	ProductName   string `json:"product_name" form:"product_name" widget:"name:商品名称;type:select" callback:"OnSelectFuzzy"`
	OrderNo       string `json:"order_no" form:"order_no" widget:"name:销售单号;type:input"`
	Creator       string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	OutboundStart string `json:"outbound_start" form:"outbound_start" widget:"name:出库开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	OutboundEnd   string `json:"outbound_end" form:"outbound_end" widget:"name:出库结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

func SalesRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req SalesRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&SalesRecord{})
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name = ?", req.ProductName)
	}
	if req.OrderNo != "" {
		queryDB = queryDB.Where("order_no LIKE ?", "%"+req.OrderNo+"%")
	}
	if req.Creator != "" {
		queryDB = queryDB.Where("created_by = ?", req.Creator)
	}
	if req.OutboundStart != "" {
		queryDB = queryDB.Where("outbound_time >= ?", req.OutboundStart)
	}
	if req.OutboundEnd != "" {
		queryDB = queryDB.Where("outbound_time <= ?", req.OutboundEnd)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	var lists []SalesRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

var SalesRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "销售记录",
		Request:      &SalesRecordListReq{},
		CreateTables: []interface{}{&SalesRecord{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"product_name": onSelectFuzzyProductNameForSaleRecord,
		},
	},
	AutoCrudTable: &SalesRecord{},
}

func onSelectFuzzyProductNameForSaleRecord(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var products []Product
	if req.IsByValue() {
		db.Where("name = ?", req.GetValue()).Limit(1).Find(&products)
	} else if req.IsByValues() {
		db.Where("name in ?", req.GetValues()).Find(&products)
	} else {
		db.Where("name LIKE ?", "%"+req.Keyword()+"%").Limit(20).Find(&products)
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(products))
	for _, p := range products {
		items = append(items, &callback.SelectFuzzyItem{Value: p.Name, Label: p.Name})
	}
	return &callback.OnSelectFuzzyResp{Items: items}, nil
}

func init() {
	packageContext.GET("sales_record_list.table", SalesRecordList, SalesRecordListTemplate)
}
