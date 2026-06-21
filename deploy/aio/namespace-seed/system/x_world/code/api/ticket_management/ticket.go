package ticket_management

import (
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

type Ticket struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DeletedBy string         `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	Title       string     `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" validate:"required,min=2,max=200"`
	Description string     `json:"description" gorm:"column:description" widget:"name:问题描述;type:richtext;height:360" validate:"required,min=10"`
	Priority    string     `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`
	Status      string     `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`
	Classify    string     `json:"classify" gorm:"column:classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0" validate:"required,oneof=民生 交通 医疗 就业 建议 其他"`
	Phone       string     `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`
	Source      string     `json:"source" gorm:"column:source" widget:"name:工单来源;type:radio;options:电话,邮件,在线,现场,其他;render_default:在线" validate:"required,oneof=电话 邮件 在线 现场 其他"`
	Handler     string     `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user"`
	Deadline    types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	Remark      string     `json:"remark" gorm:"column:remark" widget:"name:备注;type:text_area"`
	CreatedBy   string     `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (Ticket) TableName() string {
	return "ticket"
}

var TicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "工单列表",
		Request: &TicketListReq{},
		CreateTables: []interface{}{
			&Ticket{},
		},
	},
	AutoCrudTable: &Ticket{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row Ticket
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		row.CreatedBy = ctx.GetRequestUser()
		if err := db.Create(&row).Error; err != nil {
			logger.Errorf(ctx, "Create ticket err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()
		var updateFields Ticket
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, err
		}
		updates := req.ChangedFields()
		err := db.Model(&Ticket{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&Ticket{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
			"deleted_at": time.Now(),
		}).Error
		if err != nil {
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

type TicketListReq struct {
	Title             string `json:"title" form:"title" widget:"name:工单标题;type:input"`
	Status            string `json:"status" form:"status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C"`
	Priority          string `json:"priority" form:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`
	Classify          string `json:"classify" form:"classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0"`
	Handler           string `json:"handler" form:"handler" widget:"name:处理人;type:user"`
	Creator           string `json:"creator" form:"creator" widget:"name:创建人;type:user"`
	CreatedStartAt    string `json:"created_start_at" form:"created_start_at" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedEndAt      string `json:"created_end_at" form:"created_end_at" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	query.PageSortReq `widget:"-"`
}

func TicketList(ctx *app.Context, resp response.Response) error {
	var req TicketListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "TicketList ShouldBind err: %v", err)
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Ticket{})
	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.Priority != "" {
		queryDB = queryDB.Where("priority = ?", req.Priority)
	}
	if req.Classify != "" {
		queryDB = queryDB.Where("classify = ?", req.Classify)
	}
	if req.Handler != "" {
		queryDB = queryDB.Where("handler = ?", req.Handler)
	}
	if req.Creator != "" {
		queryDB = queryDB.Where("created_by = ?", req.Creator)
	}
	if req.CreatedStartAt != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedStartAt)
	}
	if req.CreatedEndAt != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedEndAt)
	}
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "TicketCount err: %v", err)
		return err
	}
	var lists []*Ticket
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "TicketSearch err: %v", err)
		return err
	}
	return resp.Table(response.TableResult{
		Items:      lists,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

func init() {
	packageContext.GET("ticket_list.table", TicketList, TicketTemplate)
}
