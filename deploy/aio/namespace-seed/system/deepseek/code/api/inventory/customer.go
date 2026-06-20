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

// Customer 客户
type Customer struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	UpdatedBy string         `json:"updated_by" gorm:"column:updated_by" widget:"name:更新人;type:user" hide:"create,update"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	Name    string `json:"name" gorm:"column:name" widget:"name:客户名称;type:input;placeholder:请输入客户名称" validate:"required,min=1,max=200"`
	Contact string `json:"contact" gorm:"column:contact" widget:"name:联系人;type:input;placeholder:请输入联系人姓名" validate:"required,min=1,max=100"`
	Phone   string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input;placeholder:请输入联系电话" validate:"required,min=1,max=50"`
	Address string `json:"address" gorm:"column:address" widget:"name:地址;type:text_area;placeholder:请输入地址" validate:"max=500"`
	Status  string `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:合作中,已停用;options_colors:4CAF50,F56C6C;render_default:合作中" validate:"required,oneof=合作中 已停用"`
}

// CustomerListReq 客户列表查询请求
type CustomerListReq struct {
	Name      string `json:"name" form:"name" widget:"name:客户名称;type:input"`
	Status    string `json:"status" form:"status" widget:"name:状态;type:select;options:合作中,已停用;options_colors:4CAF50,F56C6C"`
	CreatedBy string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`

	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

var CustomerTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "客户管理",
		Request:      &CustomerListReq{},
		CreateTables: []interface{}{&Customer{}},
	},
	AutoCrudTable:     &Customer{},
	OnTableAddRow:     onCustomerAddRow,
	OnTableUpdateRow:  onCustomerUpdateRow,
	OnTableDeleteRows: onCustomerDeleteRows,
}

func onCustomerAddRow(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
	db := ctx.GetGormDB()
	var row Customer
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

func onCustomerUpdateRow(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
	db := ctx.GetGormDB()
	var updateFields Customer
	if err := req.BindChangedFields(&updateFields); err != nil {
		return nil, err
	}
	updates := req.ChangedFields()
	updates["updated_by"] = ctx.GetRequestUser()
	if err := db.Model(&Customer{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
		return nil, err
	}
	return &callback.OnTableUpdateRowResp{}, nil
}

func onCustomerDeleteRows(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
	db := ctx.GetGormDB()
	err := db.Model(&Customer{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
		"deleted_by": ctx.GetRequestUser(),
		"deleted_at": time.Now(),
	}).Error
	if err != nil {
		return nil, err
	}
	return &callback.OnTableDeleteRowsResp{}, nil
}

// CustomerList 客户列表
func CustomerList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req CustomerListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&Customer{})
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
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

	var lists []Customer
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
	packageContext.GET("customer_list.table", CustomerList, CustomerTemplate)
}
