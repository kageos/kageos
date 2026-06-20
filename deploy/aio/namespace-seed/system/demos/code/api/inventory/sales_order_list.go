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

type SalesOrder struct {
	ID           int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	OrderNo      string         `json:"order_no" gorm:"column:order_no" widget:"name:出库单号;type:input" validate:"required"`
	CustomerID   int            `json:"customer_id" gorm:"column:customer_id" widget:"name:客户;type:select" validate:"required" callback:"OnSelectFuzzy"`
	CustomerName string         `json:"customer_name" gorm:"-" widget:"name:客户名称;type:text" hide:"create,update"`
	SalesItems   string         `json:"sales_items" gorm:"type:text;column:sales_items" widget:"name:销售明细;type:text_area" validate:"required"`
	TotalAmount  float64        `json:"total_amount" gorm:"column:total_amount" widget:"name:销售总额;type:float;precision:2;unit:元" validate:"required,min=0"`
	SalesTime    types.Time     `json:"sales_time" gorm:"column:sales_time;type:datetime" widget:"name:出库时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	Remark       string         `json:"remark" gorm:"type:text;column:remark" widget:"name:备注;type:text_area"`
	Status       string         `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待出库,已出库,已取消;options_colors:E6A23C,67C23A,909399;render_default:待出库" validate:"required,oneof=待出库 已出库 已取消"`
	CreatedAt    types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt    types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index;column:deleted_at" widget:"-"`
}

type SalesOrderListReq struct {
	OrderNo           string `json:"order_no" form:"order_no" widget:"name:出库单号;type:input"`
	CustomerName      string `json:"customer_name" form:"customer_name" widget:"name:客户;type:input"`
	Status            string `json:"status" form:"status" widget:"name:状态;type:select;options:待出库,已出库,已取消"`
	CreatedStart      string `json:"created_start" form:"created_start" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEnd        string `json:"created_end" form:"created_end" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

var SalesOrderTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "销售出库单",
		Request:      &SalesOrderListReq{},
		CreateTables: []interface{}{&SalesOrder{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"customer_id": onSelectFuzzyCustomer,
		},
	},
	AutoCrudTable: &SalesOrder{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		var row SalesOrder
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		db := ctx.GetGormDB()
		var customer Customer
		if err := db.First(&customer, row.CustomerID).Error; err != nil {
			return nil, err
		}
		row.CustomerName = customer.CustomerName
		if err := db.Create(&row).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		var updateFields SalesOrder
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		if req.IsFieldUpdated("customer_id") {
			var customer Customer
			if err := ctx.GetGormDB().First(&customer, updateFields.CustomerID).Error; err == nil {
				updates["customer_name"] = customer.CustomerName
			}
		}
		if err := ctx.GetGormDB().Model(&SalesOrder{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		if err := ctx.GetGormDB().Model(&SalesOrder{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_at": time.Now(),
		}).Error; err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func SalesOrderList(ctx *app.Context, resp response.Response) error {
	var req SalesOrderListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&SalesOrder{})
	if req.OrderNo != "" {
		queryDB = queryDB.Where("order_no LIKE ?", "%"+req.OrderNo+"%")
	}
	if req.CustomerName != "" {
		queryDB = queryDB.Where("customer_name LIKE ?", "%"+req.CustomerName+"%")
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
	var lists []*SalesOrder
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func onSelectFuzzyCustomer(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var customers []Customer
	queryDB := db.Model(&Customer{}).Where("status = ?", "正常")
	if req.IsByValue() {
		queryDB = queryDB.Where("id = ?", req.GetValue())
	} else if req.IsByValues() {
		queryDB = queryDB.Where("id IN ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		if keyword != "" {
			queryDB = queryDB.Where("customer_name LIKE ?", "%"+keyword+"%")
		}
	}
	if err := queryDB.Limit(20).Find(&customers).Error; err != nil {
		return nil, err
	}
	items := make([]*callback.SelectFuzzyItem, 0, len(customers))
	for _, c := range customers {
		items = append(items, &callback.SelectFuzzyItem{
			Value: c.ID,
			Label: c.CustomerName,
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items:         items,
		MaxSelections: 1,
	}, nil
}

func init() {
	packageContext.GET("sales_order_list.table", SalesOrderList, SalesOrderTemplate)
}
