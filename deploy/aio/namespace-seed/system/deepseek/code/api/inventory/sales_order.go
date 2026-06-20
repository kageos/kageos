package inventory

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// SalesOrder 销售单
type SalesOrder struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	CreatedBy string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	UpdatedBy string         `json:"updated_by" gorm:"column:updated_by" widget:"name:更新人;type:user" hide:"create,update"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	OrderNo     string     `json:"order_no" gorm:"column:order_no" widget:"name:销售单号;type:input" validate:"required" hide:"create"`
	Customer    string     `json:"customer" gorm:"column:customer" widget:"name:客户;type:select;placeholder:请选择客户" validate:"required" callback:"OnSelectFuzzy"`
	OrderDate   types.Time `json:"order_date" gorm:"column:order_date;type:datetime" widget:"name:销售日期;type:datetime;format:YYYY-MM-DD HH:mm:ss;render_default:CURRENT_TIMESTAMP" validate:"required"`
	ItemsDetail string     `json:"items_detail" gorm:"column:items_detail;type:text" widget:"name:商品明细;type:text_area;placeholder:商品名称×数量@单价，每行一条" validate:"required"`
	TotalAmount float64    `json:"total_amount" gorm:"column:total_amount" widget:"name:销售总金额;type:float;min:0;precision:2;step:0.01;unit:元;render_default:0" validate:"required,min=0"`
	Status      string     `json:"status" gorm:"column:status" widget:"name:状态;type:select;options:待出库,已出库,已取消;options_colors:E6A23C,4CAF50,F56C6C" hide:"create,update"`
}

// SalesOrderListReq 销售单列表查询请求
type SalesOrderListReq struct {
	OrderNo   string `json:"order_no" form:"order_no" widget:"name:销售单号;type:input"`
	Customer  string `json:"customer" form:"customer" widget:"name:客户;type:input"`
	Status    string `json:"status" form:"status" widget:"name:状态;type:select;options:待出库,已出库,已取消;options_colors:E6A23C,4CAF50,F56C6C"`
	CreatedBy string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`

	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

var SalesOrderTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "销售订单",
		Request:      &SalesOrderListReq{},
		CreateTables: []interface{}{&SalesOrder{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"customer": onSelectFuzzyCustomer,
		},
	},
	AutoCrudTable:     &SalesOrder{},
	OnTableAddRow:     onSalesOrderAddRow,
	OnTableUpdateRow:  onSalesOrderUpdateRow,
	OnTableDeleteRows: onSalesOrderDeleteRows,
}

func generateSalesOrderNo(db *gorm.DB) (string, error) {
	prefix := "SO" + time.Now().Format("20060102")
	var maxNo string
	err := db.Model(&SalesOrder{}).Where("order_no LIKE ?", prefix+"%").
		Order("order_no DESC").Limit(1).Pluck("order_no", &maxNo).Error
	if err != nil {
		return "", err
	}
	seq := 1
	if maxNo != "" {
		fmt.Sscanf(maxNo[len(prefix):], "%d", &seq)
		seq++
	}
	return fmt.Sprintf("%s%04d", prefix, seq), nil
}

func onSalesOrderAddRow(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
	db := ctx.GetGormDB()
	var row SalesOrder
	if err := ctx.ShouldBindValidate(&row); err != nil {
		return nil, err
	}
	orderNo, err := generateSalesOrderNo(db)
	if err != nil {
		return nil, fmt.Errorf("[系统错误]-[onSalesOrderAddRow] 生成销售单号失败, err: %w", err)
	}
	row.OrderNo = orderNo
	row.Status = "待出库"
	row.CreatedBy = ctx.GetRequestUser()
	row.UpdatedBy = ctx.GetRequestUser()
	if err := db.Create(&row).Error; err != nil {
		return nil, fmt.Errorf("[系统错误]-[onSalesOrderAddRow] 创建销售单失败, err: %w", err)
	}
	return &callback.OnTableAddRowResp{Data: &row}, nil
}

func onSalesOrderUpdateRow(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
	db := ctx.GetGormDB()

	var current SalesOrder
	if err := db.First(&current, req.GetId()).Error; err != nil {
		return nil, fmt.Errorf("[系统错误]-[onSalesOrderUpdateRow] 查询销售单失败, id: %d, err: %w", req.GetId(), err)
	}
	if current.Status == "已出库" || current.Status == "已取消" {
		return nil, fmt.Errorf("销售单状态为%s，不可修改", current.Status)
	}

	var updateFields SalesOrder
	if err := req.BindChangedFields(&updateFields); err != nil {
		return nil, err
	}
	updates := req.ChangedFields()
	delete(updates, "status")
	delete(updates, "order_no")
	updates["updated_by"] = ctx.GetRequestUser()
	if err := db.Model(&SalesOrder{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("[系统错误]-[onSalesOrderUpdateRow] 更新销售单失败, id: %d, err: %w", req.GetId(), err)
	}
	return &callback.OnTableUpdateRowResp{}, nil
}

func onSalesOrderDeleteRows(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
	db := ctx.GetGormDB()
	err := db.Model(&SalesOrder{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
		"deleted_by": ctx.GetRequestUser(),
		"deleted_at": time.Now(),
	}).Error
	if err != nil {
		return nil, fmt.Errorf("[系统错误]-[onSalesOrderDeleteRows] 删除销售单失败, ids: %v, err: %w", req.GetIds(), err)
	}
	return &callback.OnTableDeleteRowsResp{}, nil
}

func onSelectFuzzyCustomer(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var customers []Customer
	queryDB := db.Model(&Customer{}).Where("status = ?", "合作中")
	if req.IsByValue() {
		queryDB = queryDB.Where("name = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		queryDB = queryDB.Where("name in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		if keyword != "" {
			queryDB = queryDB.Where("name LIKE ?", "%"+keyword+"%")
		}
		queryDB = queryDB.Limit(20)
	}
	queryDB.Find(&customers)

	items := make([]*callback.SelectFuzzyItem, 0, len(customers))
	for _, c := range customers {
		items = append(items, &callback.SelectFuzzyItem{
			Value: c.Name,
			Label: c.Name,
			DisplayInfo: map[string]interface{}{
				"联系人": c.Contact,
				"电话":  c.Phone,
			},
		})
	}
	return &callback.OnSelectFuzzyResp{
		Items:         items,
		MaxSelections: 1,
	}, nil
}

// SalesOrderList 销售单列表
func SalesOrderList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req SalesOrderListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&SalesOrder{})
	if req.OrderNo != "" {
		queryDB = queryDB.Where("order_no LIKE ?", "%"+req.OrderNo+"%")
	}
	if req.Customer != "" {
		queryDB = queryDB.Where("customer LIKE ?", "%"+req.Customer+"%")
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

	var lists []SalesOrder
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
	packageContext.GET("sales_order_list.table", SalesOrderList, SalesOrderTemplate)
}
