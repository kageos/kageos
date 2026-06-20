package evaluation

import (
	"github.com/kageos/kageos/pkg/gormx/query"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// EvaluationRecord 评价记录表
type EvaluationRecord struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:提交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`

	ActivityID   int     `json:"activity_id" gorm:"column:activity_id;index" widget:"name:活动ID;type:integer" hide:"create,update"`
	ActivityName string  `json:"activity_name" gorm:"column:activity_name" widget:"name:评价活动;type:text" hide:"create,update"`
	Topic        string  `json:"topic" gorm:"column:topic" widget:"name:评价主题;type:text" hide:"create,update"`
	Evaluator    string  `json:"evaluator" gorm:"column:evaluator" widget:"name:评价人;type:user" hide:"create,update"`
	AverageScore float64 `json:"average_score" gorm:"column:average_score;type:decimal(4,2)" widget:"name:平均得分;type:float;precision:1;min:0;max:5" hide:"create,update"`
	Comment      string  `json:"comment" gorm:"column:comment;type:text" widget:"name:总体评价;type:text_area" hide:"create,update"`
}

func (EvaluationRecord) TableName() string {
	return "evaluation_record"
}

// ================ 列表 ================

// EvaluationRecordListReq 评价记录列表请求
type EvaluationRecordListReq struct {
	Activity  string `json:"activity" form:"activity" widget:"name:评价活动;type:input"`
	Topic     string `json:"topic" form:"topic" widget:"name:评价主题;type:input"`
	Evaluator string `json:"evaluator" form:"evaluator" widget:"name:评价人;type:user"`
	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// EvaluationRecordList 评价记录查询
func EvaluationRecordList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req EvaluationRecordListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&EvaluationRecord{})
	if req.Activity != "" {
		queryDB = queryDB.Where("activity_name LIKE ?", "%"+req.Activity+"%")
	}
	if req.Topic != "" {
		queryDB = queryDB.Where("topic LIKE ?", "%"+req.Topic+"%")
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

	var items []EvaluationRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&items).Error; err != nil {
		return err
	}

	return resp.Table(response.TableResult{
		Items:      items,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// EvaluationRecordListTemplate 评价记录查询配置（只读）
var EvaluationRecordListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "评价记录查询",
		Desc:         "只读查看用户提交的评价记录。",
		Request:      &EvaluationRecordListReq{},
		CreateTables: []interface{}{&EvaluationRecord{}},
	},
	AutoCrudTable: &EvaluationRecord{},
}

func init() {
	packageContext.GET("evaluation_record_list.table", EvaluationRecordList, EvaluationRecordListTemplate)
}
