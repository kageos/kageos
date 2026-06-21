package inventory

import (
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// InventoryFlow 库存流水（只读）
type InventoryFlow struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	UpdatedBy string         `json:"updated_by" gorm:"column:updated_by" widget:"name:更新人;type:user" hide:"create,update"`

	Product     string     `json:"product" gorm:"column:product" widget:"name:商品;type:input" hide:"create,update"`
	ChangeType  string     `json:"change_type" gorm:"column:change_type" widget:"name:变动类型;type:select;options:采购入库,销售出库,手动调整;options_colors:4CAF50,F56C6C,909399" hide:"create,update"`
	ChangeQty   int        `json:"change_qty" gorm:"column:change_qty" widget:"name:变动数量;type:integer;step:1" hide:"create,update"`
	AfterStock  int        `json:"after_stock" gorm:"column:after_stock" widget:"name:变动后库存;type:integer;step:1" hide:"create,update"`
	RefOrderNo  string     `json:"ref_order_no" gorm:"column:ref_order_no" widget:"name:关联单号;type:input" hide:"create,update"`
	OperateTime types.Time `json:"operate_time" gorm:"column:operate_time;type:datetime" widget:"name:操作时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
}

// InventoryFlowListReq 库存流水列表查询请求
type InventoryFlowListReq struct {
	Product    string `json:"product" form:"product" widget:"name:商品;type:input"`
	ChangeType string `json:"change_type" form:"change_type" widget:"name:变动类型;type:select;options:采购入库,销售出库,手动调整;options_colors:4CAF50,F56C6C,909399"`
	CreatedBy  string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`

	OpStartTime string `json:"op_start_time" form:"op_start_time" widget:"name:操作开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	OpEndTime   string `json:"op_end_time" form:"op_end_time" widget:"name:操作结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

var InventoryFlowTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "库存流水",
		Request:      &InventoryFlowListReq{},
		CreateTables: []interface{}{&InventoryFlow{}},
	},
	AutoCrudTable: &InventoryFlow{},
}

// InventoryFlowList 库存流水列表（只读）
func InventoryFlowList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req InventoryFlowListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&InventoryFlow{})
	if req.Product != "" {
		queryDB = queryDB.Where("product LIKE ?", "%"+req.Product+"%")
	}
	if req.ChangeType != "" {
		queryDB = queryDB.Where("change_type = ?", req.ChangeType)
	}
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}
	if req.OpStartTime != "" {
		queryDB = queryDB.Where("operate_time >= ?", req.OpStartTime)
	}
	if req.OpEndTime != "" {
		queryDB = queryDB.Where("operate_time <= ?", req.OpEndTime)
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

	var lists []InventoryFlow
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
	packageContext.GET("inventory_flow_list.table", InventoryFlowList, InventoryFlowTemplate)
}
