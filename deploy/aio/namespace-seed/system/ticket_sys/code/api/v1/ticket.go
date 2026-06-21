package v1

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// Ticket 工单
type Ticket struct {
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`

	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	DeletedBy string `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	// 工单标题
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" validate:"required,min=2,max=200"`

	// 问题描述
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`

	// 优先级：低、中、高，默认中
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`

	// 工单状态：待处理、处理中、已完成、已关闭，默认待处理
	Status string `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`

	// 问题分类：民生、交通、医疗、就业、建议、其他
	Classify string `json:"classify" gorm:"column:classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0" validate:"required,oneof=民生 交通 医疗 就业 建议 其他"`

	// 联系电话
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required,min=11,max=20"`

	// 工单来源：电话、邮件、在线、现场、其他，默认在线
	Source string `json:"source" gorm:"column:source" widget:"name:工单来源;type:radio;options:电话,邮件,在线,现场,其他;render_default:在线" validate:"required,oneof=电话 邮件 在线 现场 其他"`

	// 处理人
	Handler string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user"`

	// 截止时间
	Deadline types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	// 备注
	Remark string `json:"remark" gorm:"column:remark" widget:"name:备注;type:text_area"`

	// 创建人
	CreatedBy string `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`

	// 剩余时间（仅展示）
	RemainingTime string `json:"remaining_time" gorm:"-" widget:"name:剩余时间;type:text" hide:"create,update"`
}

func (t *Ticket) TableName() string {
	return "ticket"
}

var TicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "工单列表",
		Desc:    "管理工单记录，支持新增、编辑、删除和按条件筛选。",
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
		// 新建工单默认状态为待处理（widget 的 render_default 已在前端处理）
		row.CreatedBy = ctx.GetRequestUser()
		err := db.Create(&row).Error
		if err != nil {
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
		// 工单状态流转：待处理 → 处理中 → 已完成 → 已关闭
		// 此处仅校验，不限制流转方向（业务可根据实际情况调整）
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

// TicketListReq 工单列表查询请求
type TicketListReq struct {
	Title    string `json:"title" form:"title" widget:"name:工单标题;type:input"`
	Status   string `json:"status" form:"status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C"`
	Priority string `json:"priority" form:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`
	Classify string `json:"classify" form:"classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0"`
	Handler  string `json:"handler" form:"handler" widget:"name:处理人;type:user"`
	Creator  string `json:"creator" form:"creator" widget:"name:创建人;type:user"`
	// 创建开始时间
	CreateStartTime string `json:"create_start_time" form:"create_start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	// 创建结束时间
	CreateEndTime string `json:"create_end_time" form:"create_end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

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

	// 按工单标题模糊搜索
	if req.Title != "" {
		queryDB = queryDB.Where("title LIKE ?", "%"+req.Title+"%")
	}
	// 按工单状态筛选
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	// 按优先级筛选
	if req.Priority != "" {
		queryDB = queryDB.Where("priority = ?", req.Priority)
	}
	// 按问题分类筛选
	if req.Classify != "" {
		queryDB = queryDB.Where("classify = ?", req.Classify)
	}
	// 按处理人筛选
	if req.Handler != "" {
		queryDB = queryDB.Where("handler = ?", req.Handler)
	}
	// 按创建人筛选
	if req.Creator != "" {
		queryDB = queryDB.Where("created_by = ?", req.Creator)
	}
	// 按创建时间范围筛选
	if req.CreateStartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreateStartTime)
	}
	if req.CreateEndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreateEndTime)
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
		logger.Errorf(ctx, "TicketList err: %v", err)
		return err
	}

	// 填充剩余时间（仅展示）
	now := time.Now()
	for _, item := range lists {
		if item.Deadline.IsZero() {
			continue
		}
		deadline := item.Deadline.Time()
		if !now.Before(deadline) {
			item.RemainingTime = "已过期"
			continue
		}
		diff := deadline.Sub(now)
		days := int(diff.Hours()) / 24
		hours := int(diff.Hours()) % 24
		if days > 0 {
			item.RemainingTime = fmt.Sprintf("%d天%d小时", days, hours)
		} else {
			item.RemainingTime = fmt.Sprintf("%d小时", hours)
		}
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
