package evaluation

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// EvaluationScoreDetail 维度评分明细表
type EvaluationScoreDetail struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:提交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	RecordID      int     `json:"record_id" gorm:"column:record_id;index" widget:"name:评价记录ID;type:integer" hide:"create,update"`
	ActivityID    int     `json:"activity_id" gorm:"column:activity_id;index" widget:"name:活动ID;type:integer" hide:"create,update"`
	ActivityName  string  `json:"activity_name" gorm:"column:activity_name" widget:"name:所属评价;type:text" hide:"create,update"`
	Topic         string  `json:"topic" gorm:"column:topic" widget:"name:评价主题;type:text" hide:"create,update"`
	DimensionName string  `json:"dimension_name" gorm:"column:dimension_name" widget:"name:评价维度;type:text" hide:"create,update"`
	Score         float64 `json:"score" gorm:"column:score" widget:"name:评分;type:rate;max:5" hide:"create,update"`
	Evaluator     string  `json:"evaluator" gorm:"column:evaluator" widget:"name:评分人;type:user" hide:"create,update"`
}

func (EvaluationScoreDetail) TableName() string {
	return "evaluation_score_detail"
}

// ================ 列表 ================

// EvaluationScoreDetailListReq 维度评分明细列表请求
type EvaluationScoreDetailListReq struct {
	Activity      string `json:"activity" form:"activity" widget:"name:所属评价;type:input"`
	Topic         string `json:"topic" form:"topic" widget:"name:评价主题;type:input"`
	DimensionName string `json:"dimension_name" form:"dimension_name" widget:"name:评价维度;type:input"`
	Evaluator     string `json:"evaluator" form:"evaluator" widget:"name:评分人;type:user"`
	StartTime     string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime       string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// EvaluationScoreDetailList 维度评分明细查询
func EvaluationScoreDetailList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req EvaluationScoreDetailListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&EvaluationScoreDetail{})
	if req.Activity != "" {
		queryDB = queryDB.Where("activity_name LIKE ?", "%"+req.Activity+"%")
	}
	if req.Topic != "" {
		queryDB = queryDB.Where("topic LIKE ?", "%"+req.Topic+"%")
	}
	if req.DimensionName != "" {
		queryDB = queryDB.Where("dimension_name LIKE ?", "%"+req.DimensionName+"%")
	}
	if req.Evaluator != "" {
		queryDB = queryDB.Where("evaluator = ?", req.Evaluator)
	}
	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("created_at DESC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var items []EvaluationScoreDetail
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// EvaluationScoreDetailListTemplate 维度评分明细查询配置（只读）
var EvaluationScoreDetailListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "维度评分明细",
		Desc:         "只读查看每条评价记录中各维度的评分明细。",
		Request:      &EvaluationScoreDetailListReq{},
		CreateTables: []interface{}{&EvaluationScoreDetail{}},
	},
	AutoCrudTable: &EvaluationScoreDetail{},
}

func init() {
	packageContext.GET("evaluation_score_detail_list.table", EvaluationScoreDetailList, EvaluationScoreDetailListTemplate)
}
