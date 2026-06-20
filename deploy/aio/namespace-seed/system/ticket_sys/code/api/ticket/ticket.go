package ticket

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

type Ticket struct {
	// ID：主键，仅列表展示
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`

	// CreatedAt：创建时间
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	// UpdatedAt：更新时间
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	// DeletedAt：软删除标记
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	// DeletedBy：删除操作人
	DeletedBy string `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	// 工单标题：必填，2-200字
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" validate:"required,min=2,max=200"`

	// 问题描述：必填，不少于10字
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required,min=10"`

	// 优先级：必填，低/中/高，默认中
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`

	// 工单状态：必填，待处理/处理中/已完成，默认待处理
	Status string `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成"`

	// 问题分类：必填，设备故障/软件问题/网络问题/其他
	Classify string `json:"classify" gorm:"column:classify" widget:"name:问题分类;type:select;options:设备故障,软件问题,网络问题,其他;options_colors:F56C6C,409EFF,FF9800,909399" validate:"required,oneof=设备故障 软件问题 网络问题 其他"`

	// 联系电话：必填
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required"`

	// 处理人：非必填
	Handler string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user"`

	// 截止时间：非必填
	Deadline types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	// 创建人：仅列表展示
	CreatedBy string `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (t *Ticket) TableName() string {
	return "ticket"
}

var TicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "工单列表",
		Tags:    []string{"工单系统"},
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
		// 新建工单默认状态为待处理
		if row.Status == "" {
			row.Status = "待处理"
		}
		// 优先级默认为中
		if row.Priority == "" {
			row.Priority = "中"
		}
		row.CreatedBy = ctx.GetRequestUser()
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[OnTableAddRow] 创建工单失败, req: %+v, err: %v", req, err)
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
			logger.Errorf(ctx, "[系统错误]-[OnTableUpdateRow] 更新工单失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&Ticket{}).Where("id in (?)", req.GetIds()).Updates(map[string]interface{}{
			"deleted_by": ctx.GetRequestUser(),
		}).Delete(&Ticket{}).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[OnTableDeleteRows] 删除工单失败, req: %+v, err: %v", req, err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// TicketListReq：工单列表查询请求
type TicketListReq struct {
	Title     string `json:"title" form:"title" widget:"name:工单标题;type:input"`
	Status    string `json:"status" form:"status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成;options_colors:909399,E6A23C,67C23A"`
	Priority  string `json:"priority" form:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`
	Classify  string `json:"classify" form:"classify" widget:"name:问题分类;type:select;options:设备故障,软件问题,网络问题,其他;options_colors:F56C6C,409EFF,FF9800,909399"`
	Handler   string `json:"handler" form:"handler" widget:"name:处理人;type:user"`
	CreatedBy string `json:"created_by" form:"created_by" widget:"name:创建人;type:user"`
	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

func TicketList(ctx *app.Context, resp response.Response) error {
	var req TicketListReq
	if err := ctx.ShouldBind(&req); err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketList] 绑定请求失败, req: %+v, err: %v", req, err)
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
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}
	// 按创建时间范围筛选
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
		logger.Errorf(ctx, "[系统错误]-[TicketList] 统计工单数量失败, req: %+v, err: %v", req, err)
		return err
	}

	var lists []*Ticket
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketList] 查询工单列表失败, req: %+v, err: %v", req, err)
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
