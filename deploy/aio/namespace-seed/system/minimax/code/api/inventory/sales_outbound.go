package inventory

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// 销售出库记录表 - 只读查看销售出库记录
type SalesOutbound struct {
	ID        int        `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	OutboundNo   string     `json:"outbound_no" gorm:"column:outbound_no" widget:"name:出库单号;type:input" hide:"create,update"`
	OutboundTime types.Time `json:"outbound_time" gorm:"column:outbound_time;type:datetime" widget:"name:出库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	CustomerName string     `json:"customer_name" gorm:"column:customer_name" widget:"name:客户;type:input" hide:"create,update"`
	ProductName  string     `json:"product_name" gorm:"column:product_name" widget:"name:商品;type:input" hide:"create,update"`
	Quantity     int        `json:"quantity" gorm:"column:quantity" widget:"name:销售数量;type:integer" hide:"create,update"`
	SalesPrice   float64    `json:"sales_price" gorm:"column:sales_price;type:decimal(10,2)" widget:"name:销售单价;type:float;precision:2;unit:元" hide:"create,update"`
	SalesAmount  float64    `json:"sales_amount" gorm:"column:sales_amount;type:decimal(10,2)" widget:"name:销售额;type:float;precision:2;unit:元" hide:"create,update"`
	Profit       float64    `json:"profit" gorm:"column:profit;type:decimal(10,2)" widget:"name:毛利;type:float;precision:2;unit:元" hide:"create,update"`
	Warehouse    string     `json:"warehouse" gorm:"column:warehouse" widget:"name:出库仓库;type:input" hide:"create,update"`
	CreatedBy    string     `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (SalesOutbound) TableName() string {
	return "sales_outbound"
}

var SalesOutboundTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "销售出库记录",
		Request:      &SalesOutboundListReq{},
		CreateTables: []interface{}{&SalesOutbound{}},
	},
	AutoCrudTable: &SalesOutbound{},
}

type SalesOutboundListReq struct {
	OutboundNo        string `json:"outbound_no" form:"outbound_no" widget:"name:出库单号;type:input"`
	CustomerName      string `json:"customer_name" form:"customer_name" widget:"name:客户;type:input"`
	ProductName       string `json:"product_name" form:"product_name" widget:"name:商品;type:input"`
	OutboundTimeStart string `json:"outbound_time_start" form:"outbound_time_start" widget:"name:出库开始时间;type:datetime"`
	OutboundTimeEnd   string `json:"outbound_time_end" form:"outbound_time_end" widget:"name:出库结束时间;type:datetime"`
	CreatedBy         string `json:"created_by" form:"created_by" gorm:"-" widget:"name:创建人;type:user" hide:"create,update"`
	CreatedAtStart    string `json:"created_at_start" form:"created_at_start" widget:"name:创建开始时间;type:datetime"`
	CreatedAtEnd      string `json:"created_at_end" form:"created_at_end" widget:"name:创建结束时间;type:datetime"`
	query.PageSortReq `widget:"-"`
}

func SalesOutboundList(ctx *app.Context, resp response.Response) error {
	var req SalesOutboundListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[系统错误]-[SalesOutboundList] 绑定参数失败, err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&SalesOutbound{})
	if req.OutboundNo != "" {
		queryDB = queryDB.Where("outbound_no LIKE ?", "%"+req.OutboundNo+"%")
	}
	if req.CustomerName != "" {
		queryDB = queryDB.Where("customer_name LIKE ?", "%"+req.CustomerName+"%")
	}
	if req.ProductName != "" {
		queryDB = queryDB.Where("product_name LIKE ?", "%"+req.ProductName+"%")
	}
	if req.OutboundTimeStart != "" {
		queryDB = queryDB.Where("outbound_time >= ?", req.OutboundTimeStart)
	}
	if req.OutboundTimeEnd != "" {
		queryDB = queryDB.Where("outbound_time <= ?", req.OutboundTimeEnd)
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
		logger.Errorf(ctx, "[系统错误]-[SalesOutboundList] 统计失败, err: %v", err)
		return err
	}
	var lists []*SalesOutbound
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[SalesOutboundList] 查询失败, err: %v", err)
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// GenerateOutboundNo 生成出库单号 XS + 日期 + 序号
func GenerateOutboundNo(db *gorm.DB) string {
	dateStr := fmt.Sprintf("%s", types.Time{}.Time().Format("20060102"))
	var count int64
	db.Model(&SalesOutbound{}).Where("outbound_no LIKE ?", "XS"+dateStr+"%").Count(&count)
	seq := count + 1
	return fmt.Sprintf("XS%s%03d", dateStr, seq)
}

func init() {
	packageContext.GET("sales_outbound_list.table", SalesOutboundList, SalesOutboundTemplate)
}
