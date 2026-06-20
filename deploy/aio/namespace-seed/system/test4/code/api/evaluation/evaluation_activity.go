package evaluation

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/statistics"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// EvaluationActivity 评价活动表
type EvaluationActivity struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	UpdatedAt types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	Name        string     `json:"name" gorm:"column:name" widget:"name:评价活动名称;type:input;placeholder:请输入评价活动名称" validate:"required,min=2,max=100"`
	Description string     `json:"description" gorm:"column:description;type:text" widget:"name:评价描述;type:text_area;placeholder:请输入评价活动的背景说明"`
	StartTime   types.Time `json:"start_time" gorm:"column:start_time;type:datetime" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required"`
	EndTime     types.Time `json:"end_time" gorm:"column:end_time;type:datetime" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" validate:"required,gtfield=StartTime"`

	Status        string  `json:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A" hide:"create,update"`
	EvalCount     int     `json:"eval_count" gorm:"column:eval_count;default:0" widget:"name:评价人数;type:integer;unit:人" hide:"create,update"`
	AverageScore  float64 `json:"average_score" gorm:"column:average_score;default:0;type:decimal(4,2)" widget:"name:平均得分;type:float;precision:1;min:0;max:5" hide:"create,update"`
	CreatedBy     string  `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	DimensionLink string  `json:"dimension_link" gorm:"-" widget:"name:评价维度;type:link;target:_blank" hide:"create,update"`
	EvalLink      string  `json:"eval_link" gorm:"-" widget:"name:评价操作;type:link;target:_blank" hide:"create,update"`
}

func (EvaluationActivity) TableName() string {
	return "evaluation_activity"
}

// ================ 辅助函数 ================

func getActivityStatus(startTime, endTime types.Time) string {
	now := time.Now()
	if now.Before(startTime.Time()) {
		return "未开始"
	}
	if now.After(endTime.Time()) {
		return "已结束"
	}
	return "进行中"
}

// ================ 模糊搜索回调 ================

// evaluationOnSelectFuzzyActivity 评价活动模糊搜索回调
func evaluationOnSelectFuzzyActivity(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var activities []EvaluationActivity

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id IN ?", req.GetValues())
	} else {
		db = db.Where("(name LIKE ? OR description LIKE ?)",
			"%"+req.Keyword()+"%", "%"+req.Keyword()+"%").
			Limit(20)
	}
	db.Find(&activities)

	items := make([]*callback.SelectFuzzyItem, 0, len(activities))
	for _, a := range activities {
		status := getActivityStatus(a.StartTime, a.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value: a.ID,
			Label: fmt.Sprintf("%s - %s", a.Name, status),
			DisplayInfo: map[string]interface{}{
				"评价活动": a.Name,
				"评价描述": a.Description,
				"活动状态": status,
				"时间范围": fmt.Sprintf("%s - %s",
					a.StartTime.Time().Format("2006-01-02 15:04"),
					a.EndTime.Time().Format("2006-01-02 15:04")),
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"评价活动": statistics.Value("评价活动"),
			"活动状态": statistics.Value("活动状态"),
			"时间范围": statistics.Value("时间范围"),
			"评价描述": statistics.Value("评价描述"),
		},
	}, nil
}

// evaluationOnSelectFuzzyActivityForSubmit 仅返回进行中的评价活动
func evaluationOnSelectFuzzyActivityForSubmit(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	var activities []EvaluationActivity
	now := time.Now()

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id IN ?", req.GetValues())
	} else {
		db = db.Where("(name LIKE ? OR description LIKE ?) AND start_time <= ? AND end_time > ?",
			"%"+req.Keyword()+"%", "%"+req.Keyword()+"%", now, now).
			Limit(20)
	}
	db.Find(&activities)

	items := make([]*callback.SelectFuzzyItem, 0, len(activities))
	for _, a := range activities {
		status := getActivityStatus(a.StartTime, a.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value: a.ID,
			Label: fmt.Sprintf("%s - %s", a.Name, status),
			DisplayInfo: map[string]interface{}{
				"评价活动": a.Name,
				"评价描述": a.Description,
				"活动状态": status,
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		MaxSelections: 1,
		Items:         items,
		Statistics: map[string]interface{}{
			"评价活动": statistics.Value("评价活动"),
			"活动状态": statistics.Value("活动状态"),
			"评价描述": statistics.Value("评价描述"),
		},
	}, nil
}

// ================ 列表 ================

// EvaluationActivityListReq 评价活动列表请求
type EvaluationActivityListReq struct {
	Name      string `json:"name" form:"name" widget:"name:评价活动名称;type:input"`
	Status    string `json:"status" form:"status" gorm:"-" widget:"name:状态;type:select;options:未开始,进行中,已结束;options_colors:909399,409EFF,67C23A"`
	CreatedBy string `json:"created_by" form:"created_by" widget:"name:创建人;type:user"`
	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// EvaluationActivityList 评价活动管理
func EvaluationActivityList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req EvaluationActivityListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&EvaluationActivity{})
	if req.Name != "" {
		queryDB = queryDB.Where("name LIKE ?", "%"+req.Name+"%")
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

	now := time.Now()
	switch req.Status {
	case "未开始":
		queryDB = queryDB.Where("start_time > ?", now)
	case "进行中":
		queryDB = queryDB.Where("start_time <= ? AND end_time > ?", now, now)
	case "已结束":
		queryDB = queryDB.Where("end_time <= ?", now)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var items []EvaluationActivity
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}

	for i := range items {
		items[i].Status = getActivityStatus(items[i].StartTime, items[i].EndTime)
		dimParams := EvaluationDimension{ActivityID: items[i].ID}
		items[i].DimensionLink, _ = ctx.BuildFunctionUrlWithText("evaluation_dimension_list.table", dimParams, "管理评价维度")
		if items[i].Status == "进行中" {
			subParams := EvaluationSubmitReq{ActivityID: items[i].ID}
			items[i].EvalLink, _ = ctx.BuildFunctionUrlWithText("evaluation_submit.form", subParams, "提交评价")
		} else {
			chartParams := EvaluationDimensionScoreReq{ActivityID: items[i].ID}
			items[i].EvalLink, _ = ctx.BuildFunctionUrlWithText("evaluation_dimension_score.chart", chartParams, "查看统计")
		}
	}

	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// EvaluationActivityListTemplate 评价活动管理配置
var EvaluationActivityListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "评价活动管理",
		Desc:         "维护评价活动的名称、时间范围和状态。",
		Request:      &EvaluationActivityListReq{},
		CreateTables: []interface{}{&EvaluationActivity{}, &EvaluationDimension{}, &EvaluationRecord{}, &EvaluationScoreDetail{}},
	},
	AutoCrudTable: &EvaluationActivity{},

	OnTableAddRow: func(ctx *app.Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error) {
		var row EvaluationActivity
		if err := ctx.ShouldBindValidate(&row); err != nil {
			return nil, err
		}
		row.CreatedBy = ctx.GetRequestUser()
		row.EvalCount = 0
		row.AverageScore = 0
		if err := ctx.GetGormDB().Create(&row).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[EvaluationActivity-Add] 创建失败, req: %+v, err: %v", row, err)
			return nil, fmt.Errorf("[系统错误]-[EvaluationActivity-Add] 创建失败, err: %w", err)
		}
		return &callback.OnTableAddRowResp{Data: &row}, nil
	},

	OnTableUpdateRow: func(ctx *app.Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error) {
		updates := req.ChangedFields()
		if err := ctx.GetGormDB().Model(&EvaluationActivity{}).Where("id = ?", req.GetId()).Updates(updates).Error; err != nil {
			logger.Errorf(ctx, "[系统错误]-[EvaluationActivity-Update] 更新失败, id: %v, err: %v", req.GetId(), err)
			return nil, fmt.Errorf("[系统错误]-[EvaluationActivity-Update] 更新失败, err: %w", err)
		}
		return &callback.OnTableUpdateRowResp{}, nil
	},

	OnTableDeleteRows: func(ctx *app.Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error) {
		db := ctx.GetGormDB()
		err := db.Transaction(func(tx *gorm.DB) error {
			// 删除活动下所有维度
			if err := tx.Where("activity_id IN ?", req.GetIds()).Delete(&EvaluationDimension{}).Error; err != nil {
				return err
			}
			// 删除活动下所有评分明细
			if err := tx.Where("activity_id IN ?", req.GetIds()).Delete(&EvaluationScoreDetail{}).Error; err != nil {
				return err
			}
			// 删除活动下所有评价记录
			if err := tx.Where("activity_id IN ?", req.GetIds()).Delete(&EvaluationRecord{}).Error; err != nil {
				return err
			}
			return tx.Where("id IN ?", req.GetIds()).Delete(&EvaluationActivity{}).Error
		})
		if err != nil {
			logger.Errorf(ctx, "[系统错误]-[EvaluationActivity-Delete] 删除失败, ids: %v, err: %v", req.GetIds(), err)
			return nil, fmt.Errorf("[系统错误]-[EvaluationActivity-Delete] 删除失败, err: %w", err)
		}
		return &callback.OnTableDeleteRowsResp{}, nil
	},
}

func init() {
	packageContext.GET("evaluation_activity_list.table", EvaluationActivityList, EvaluationActivityListTemplate)
}
