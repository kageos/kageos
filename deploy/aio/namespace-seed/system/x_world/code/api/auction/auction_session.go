package auction

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

// ================ 拍卖场次管理 ================

// AuctionSession 拍卖场次表
type AuctionSession struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Name      string     `json:"name" gorm:"column:name;comment:场次名称" widget:"name:场次名称;type:input" validate:"required,min=2,max=100"`
	Rules     string     `json:"rules" gorm:"column:rules;type:text;comment:场次规则" widget:"name:场次规则;type:text_area"`
	StartTime types.Time `json:"start_time" gorm:"column:start_time;type:datetime;comment:开始时间" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	EndTime   types.Time `json:"end_time" gorm:"column:end_time;type:datetime;comment:结束时间" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	Status    string     `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"`
	ItemCount int        `json:"item_count" gorm:"-" widget:"name:拍品数量;type:integer" hide:"create,update"`
}

func (AuctionSession) TableName() string {
	return "auction_session"
}

// AuctionSessionListReq 拍卖场次列表请求
type AuctionSessionListReq struct {
	Name      string `json:"name" form:"name" widget:"name:场次名称;type:input"`
	Status    string `json:"status" form:"status" widget:"name:场次状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`
	CreatedBy string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	StartDate string `json:"start_date" form:"start_date" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// AuctionSessionList 拍卖场次管理
func AuctionSessionList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req AuctionSessionListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&AuctionSession{})

	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}
	if req.StartDate != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndDate)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("id DESC")
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var sessions []AuctionSession
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&sessions).Error; err != nil {
		return err
	}

	// 填充状态和拍品数量
	for i := range sessions {
		sessions[i].Status = calculateSessionStatus(sessions[i].StartTime, sessions[i].EndTime)
		// 统计拍品数量
		var count int64
		db.Model(&AuctionItem{}).Where("session_id = ?", sessions[i].ID).Count(&count)
		sessions[i].ItemCount = int(count)
	}

	return resp.Table(response.TableResult{
		Items:      sessions,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// AuctionSessionListTemplate 拍卖场次管理配置
var AuctionSessionListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "拍卖场次管理",
		Desc:         `维护拍卖活动的场次信息，包括名称、时间、规则和状态`,
		Tags:         []string{"拍卖会系统", "场次管理"},
		Request:      &AuctionSessionListReq{},
		CreateTables: []interface{}{&AuctionSession{}},
	},
	AutoCrudTable: &AuctionSession{},
	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var row AuctionSession
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		// 校验结束时间必须晚于开始时间
		if !row.EndTime.Time().After(row.StartTime.Time()) {
			return nil, fmt.Errorf("结束时间必须晚于开始时间")
		}
		err := db.Create(&row).Error
		if err != nil {
			logger.Errorf(ctx, "Create auction_session err: %v", err)
			return nil, err
		}
		return &callback.OnTableAddRowResp{Data: row}, nil
	},
	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields AuctionSession
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.ChangedFields()

		// 如果更新了时间字段，需要校验
		if req.IsFieldUpdated("start_time") || req.IsFieldUpdated("end_time") {
			var current AuctionSession
			if err := db.Where("id = ?", req.GetId()).First(&current).Error; err != nil {
				return nil, fmt.Errorf("场次记录不存在")
			}
			startTime := current.StartTime
			endTime := current.EndTime
			if req.IsFieldUpdated("start_time") {
				startTime = updateFields.StartTime
			}
			if req.IsFieldUpdated("end_time") {
				endTime = updateFields.EndTime
			}
			if !endTime.Time().After(startTime.Time()) {
				return nil, fmt.Errorf("结束时间必须晚于开始时间")
			}
		}

		err := db.Model(&AuctionSession{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "Update auction_session err: %v", err)
			return nil, err
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},
	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Model(&AuctionSession{}).Delete(&AuctionSession{}, "id in ?", req.GetIds()).Error
		if err != nil {
			logger.Errorf(ctx, "Delete auction_session err: %v", err)
			return nil, err
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// calculateSessionStatus 计算场次状态（实时计算，不存储）
func calculateSessionStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "未开始"
	} else if now.Before(endTime.Time()) {
		return "进行中"
	}
	return "已结束"
}

// ================ API 注册 ================

func init() {
	packageContext.GET("auction_session_list.table", AuctionSessionList, AuctionSessionListTemplate)
}
