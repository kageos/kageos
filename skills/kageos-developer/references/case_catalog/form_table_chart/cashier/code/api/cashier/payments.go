package cashier

import (
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/pkg/gormx/query"
)

type PaymentListReq struct {
	OrderNo           string `json:"order_no" form:"order_no" widget:"name:订单号;type:input"`
	Keyword           string `json:"keyword" form:"keyword" widget:"name:消费明细;type:input;placeholder:商品名称"`
	PaymentMethod     string `json:"payment_method" form:"payment_method" widget:"name:支付方式;type:select;options:全部,现金,微信,支付宝,银行卡,其他;options_colors:909399,67C23A,409EFF,00B8A9,E6A23C,909399;render_default:全部"`
	PaymentStatus     string `json:"payment_status" form:"payment_status" widget:"name:支付状态;type:select;options:全部,支付成功;options_colors:909399,67C23A;render_default:全部"`
	Cashier           string `json:"cashier" form:"cashier" widget:"name:收银员;type:user"`
	StartTime         string `json:"start_time" form:"start_time" widget:"name:支付开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime           string `json:"end_time" form:"end_time" widget:"name:支付结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

func PaymentList(ctx *app.Context, resp response.Response) error {
	db, err := cashierDB(ctx)
	if err != nil {
		return err
	}
	var req PaymentListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&Payment{})
	if orderNo := strings.TrimSpace(req.OrderNo); orderNo != "" {
		queryDB = queryDB.Where("order_no LIKE ?", "%"+orderNo+"%")
	}
	if keyword := strings.TrimSpace(req.Keyword); keyword != "" {
		queryDB = queryDB.Where("items_desc LIKE ?", "%"+keyword+"%")
	}
	if method := strings.TrimSpace(req.PaymentMethod); method != "" && method != "全部" {
		queryDB = queryDB.Where("payment_method = ?", method)
	}
	if status := strings.TrimSpace(req.PaymentStatus); status != "" && status != "全部" {
		queryDB = queryDB.Where("payment_status = ?", status)
	}
	if cashier := strings.TrimSpace(req.Cashier); cashier != "" {
		queryDB = queryDB.Where("cashier = ?", cashier)
	}
	if strings.TrimSpace(req.StartTime) != "" {
		queryDB = queryDB.Where("payment_time >= ?", req.StartTime)
	}
	if strings.TrimSpace(req.EndTime) != "" {
		queryDB = queryDB.Where("payment_time <= ?", req.EndTime)
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("payment_time DESC, id DESC")
	}
	var rows []Payment
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rows).Error; err != nil {
		return err
	}
	return resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build()
}

var PaymentListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "支付记录",
		Desc:         "只读查看收银台生成的支付流水。支付记录不支持手工新增、编辑或删除。",
		Tags:         []string{"收银", "支付流水"},
		Request:      &PaymentListReq{},
		CreateTables: []interface{}{&Payment{}},
	},
	AutoCrudTable: &Payment{},
}

func init() {
	packageContext.GET("payments.table", PaymentList, PaymentListTemplate)
}
