// nps_questionnaire_list.go
// NPS问卷管理：数据模型、列表 Handler、Template

package nps

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// NpsQuestionnaire NPS问卷表
type NpsQuestionnaire struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:问卷ID;type:ID" hide:"create,update"`                                                      // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`                                                                                              // 软删除字段
	CreatedBy string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`                                                              // 前端仅在列表展示，不进入新增/编辑表单。

	Name        string     `json:"name" gorm:"column:name;comment:问卷名称" widget:"name:问卷名称;type:input" validate:"required"`
	Description string     `json:"description" gorm:"column:description;comment:问卷说明" widget:"name:问卷说明;type:text_area" validate:"required"`
	StartTime   types.Time `json:"start_time" gorm:"column:start_time;type:datetime;comment:开始时间" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	EndTime     types.Time `json:"end_time" gorm:"column:end_time;type:datetime;comment:截止时间" widget:"name:截止时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`

	// 状态是计算字段，按开始时间和截止时间实时计算
	Status string `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (NpsQuestionnaire) TableName() string {
	return "nps_questionnaire"
}

// ================ 辅助函数 ================

// getQuestionnaireStatus 获取问卷状态
func getQuestionnaireStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "未开始"
	} else if now.After(endTime.Time()) {
		return "已结束"
	}
	return "进行中"
}

// ================ 列表请求结构 ================

// NpsQuestionnaireListReq 问卷列表请求
type NpsQuestionnaireListReq struct {
	Name        string `json:"name" form:"name" widget:"name:问卷名称;type:input"`
	Status      string `json:"status" form:"status" gorm:"-" widget:"name:问卷状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`
	CreatedBy   string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	CreatedFrom string `json:"created_from" form:"created_from" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedTo   string `json:"created_to" form:"created_to" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// ================ 问卷列表 Handler ================

// NpsQuestionnaireList 问卷列表
func NpsQuestionnaireList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req NpsQuestionnaireListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&NpsQuestionnaire{})

	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
	}

	now := time.Now()
	if req.Status != "" {
		switch req.Status {
		case "未开始":
			queryDB = queryDB.Where("start_time > ?", now)
		case "进行中":
			queryDB = queryDB.Where("start_time <= ? AND end_time > ?", now, now)
		case "已结束":
			queryDB = queryDB.Where("end_time <= ?", now)
		}
	}

	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}

	if req.CreatedFrom != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedFrom)
	}
	if req.CreatedTo != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedTo)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var questionnaires []NpsQuestionnaire
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&questionnaires).Error; err != nil {
		return err
	}

	// 返回前计算状态
	for i := range questionnaires {
		questionnaires[i].Status = getQuestionnaireStatus(questionnaires[i].StartTime, questionnaires[i].EndTime)
	}

	return resp.Table(response.TableResult{
		Items:      questionnaires,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// ================ TableTemplate ================

var NpsQuestionnaireListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "NPS问卷管理",
		Desc:         "维护 NPS 调研问卷，包括标题、说明、收集时间范围和状态。",
		Tags:         []string{"NPS", "问卷管理"},
		Request:      &NpsQuestionnaireListReq{},
		CreateTables: []interface{}{&NpsQuestionnaire{}, &NpsScoreRecord{}},
	},
	AutoCrudTable: &NpsQuestionnaire{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		db := ctx.GetGormDB()
		var questionnaire NpsQuestionnaire
		if err := ctx.ShouldBindValidate(&questionnaire); err != nil {
			return nil, err
		}

		questionnaire.CreatedBy = ctx.GetRequestUser()

		err := db.Create(&questionnaire).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[OnTableAddRow] 创建问卷失败, req: %+v, err: %v", questionnaire, err)
			return nil, fmt.Errorf("[系统错误]-[OnTableAddRow]：创建问卷失败, err: %w", err)
		}

		return &callback.OnTableAddRowResp{Data: &questionnaire}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		db := ctx.GetGormDB()

		var updateFields NpsQuestionnaire
		if err := req.BindChangedFields(&updateFields); err != nil {
			return nil, fmt.Errorf("绑定更新字段失败: %w", err)
		}

		updates := req.ChangedFields()
		err := db.Model(&NpsQuestionnaire{}).Where("id = ?", req.GetId()).Updates(updates).Error
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[OnTableUpdateRow] 更新问卷失败, req.GetId(): %v, err: %v", req.GetId(), err)
			return nil, fmt.Errorf("[系统错误]-[OnTableUpdateRow]：更新问卷失败, err: %w", err)
		}

		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()

		err := db.Transaction(func(tx *gorm.DB) error {
			// 删除关联的评分记录
			if err := tx.Where("questionnaire_id IN ?", req.GetIds()).Delete(&NpsScoreRecord{}).Error; err != nil {
				return fmt.Errorf("删除关联评分记录失败: %w", err)
			}
			// 删除问卷
			if err := tx.Where("id IN ?", req.GetIds()).Delete(&NpsQuestionnaire{}).Error; err != nil {
				return fmt.Errorf("删除问卷失败: %w", err)
			}
			return nil
		})

		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[OnTableDeleteRows] 删除问卷失败, req.GetIds(): %v, err: %v", req.GetIds(), err)
			return nil, fmt.Errorf("[系统错误]-[OnTableDeleteRows]：删除问卷失败, err: %w", err)
		}

		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("nps_questionnaire_list.table", NpsQuestionnaireList, NpsQuestionnaireListTemplate)
}
