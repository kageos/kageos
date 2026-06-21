package ticket

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

// Ticket 工单表
type Ticket struct {
	// ID：主键，前端仅在列表展示，不进入新增/编辑表单
	ID int `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`

	// 创建时间：前端仅在列表展示
	CreatedAt types.Time `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	// 更新时间：前端仅在列表展示
	UpdatedAt types.Time `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`

	// 软删除标记
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	// 删除操作人
	DeletedBy string `json:"deleted_by" gorm:"column:deleted_by" widget:"-"`

	// 工单标题：必填，2-200字
	Title string `json:"title" gorm:"column:title" widget:"name:工单标题;type:input" validate:"required,min=2,max=200"`

	// 问题描述：必填，详细描述问题
	Description string `json:"description" gorm:"column:description" widget:"name:问题描述;type:text_area" validate:"required"`

	// 优先级：必填，低/中/高三个选项，默认中
	Priority string `json:"priority" gorm:"column:priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C;render_default:中" validate:"required,oneof=低 中 高"`

	// 工单状态：必填，待处理/处理中/已完成/已关闭，默认待处理
	Status string `json:"status" gorm:"column:status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C;render_default:待处理" validate:"required,oneof=待处理 处理中 已完成 已关闭"`

	// 问题分类：必填，民生/交通/医疗/就业/建议/其他
	Classify string `json:"classify" gorm:"column:classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0" validate:"required,oneof=民生 交通 医疗 就业 建议 其他"`

	// 联系电话：必填
	Phone string `json:"phone" gorm:"column:phone" widget:"name:联系电话;type:input" validate:"required"`

	// 处理人：负责处理该工单的人员
	Handler string `json:"handler" gorm:"column:handler" widget:"name:处理人;type:user"`

	// 截止时间：期望完成处理的截止时间
	Deadline types.Time `json:"deadline" gorm:"column:deadline;type:datetime" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	// 备注：处理过程中的补充说明
	Remark string `json:"remark" gorm:"column:remark" widget:"name:备注;type:text_area"`

	// 创建人：前端仅在列表展示，由后端通过 ctx.GetRequestUser() 赋值
	CreatedBy string `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
}

func (t *Ticket) TableName() string {
	return "ticket"
}

var TicketTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "工单列表",
		Tags:    []string{"工单管理"},
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
		// 新建工单默认状态为待处理（前端已设置 render_default，这里后端也确保）
		if row.Status == "" {
			row.Status = "待处理"
		}
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
	// 工单标题：按工单标题模糊搜索
	Title string `json:"title" form:"title" widget:"name:工单标题;type:input"`

	// 工单状态：按待处理、处理中、已完成、已关闭筛选
	Status string `json:"status" form:"status" widget:"name:工单状态;type:select;options:待处理,处理中,已完成,已关闭;options_colors:909399,E6A23C,67C23A,F56C6C"`

	// 优先级：按低、中、高筛选
	Priority string `json:"priority" form:"priority" widget:"name:优先级;type:select;options:低,中,高;options_colors:67C23A,E6A23C,F56C6C"`

	// 问题分类：按民生、交通、医疗、就业、建议、其他筛选
	Classify string `json:"classify" form:"classify" widget:"name:问题分类;type:select;options:民生,交通,医疗,就业,建议,其他;options_colors:909399,E6A23C,67C23A,F56C6C,FF9800,9C27B0"`

	// 处理人：按处理人筛选
	Handler string `json:"handler" form:"handler" widget:"name:处理人;type:user"`

	// 创建人：按创建人筛选
	Creator string `json:"creator" form:"creator" widget:"name:创建人;type:user"`

	// 创建开始时间：按记录创建时间范围查询的开始时间
	CreatedStartTime string `json:"created_start_time" form:"created_start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	// 创建结束时间：按记录创建时间范围查询的结束时间
	CreatedEndTime string `json:"created_end_time" form:"created_end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// TicketList 工单列表查询
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
	if req.CreatedStartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedStartTime)
	}
	if req.CreatedEndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedEndTime)
	}

	// 排序
	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}

	// 统计总数
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "TicketList Count err: %v", err)
		return err
	}

	// 分页查询
	var lists []*Ticket
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&lists).Error; err != nil {
		logger.Errorf(ctx, "TicketList Find err: %v", err)
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
