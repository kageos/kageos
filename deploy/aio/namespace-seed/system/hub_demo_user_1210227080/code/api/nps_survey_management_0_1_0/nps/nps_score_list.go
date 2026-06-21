// nps_score_list.go
// NPS评分记录：只读查看用户提交的评分记录

package nps

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// NpsScoreRecord NPS评分记录表
type NpsScoreRecord struct {
	ID        int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:记录ID;type:ID" hide:"create,update"`                                                      // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:提交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`                                                                                              // 软删除字段
	CreatedBy string         `json:"created_by" gorm:"column:created_by" widget:"name:提交人;type:user" hide:"create,update"`                                                              // 前端仅在列表展示，不进入新增/编辑表单。

	QuestionnaireID   int               `json:"questionnaire_id" gorm:"column:questionnaire_id;index" widget:"name:问卷ID;type:select" callback:"OnSelectFuzzy"`
	Questionnaire     *NpsQuestionnaire `json:"-" widget:"-" gorm:"foreignKey:QuestionnaireID"`
	QuestionnaireName string            `json:"questionnaire_name" gorm:"-" widget:"name:问卷名称;type:input" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。

	Score     int    `json:"score" gorm:"column:score;comment:评分0-10" widget:"name:评分;type:integer;min:0;max:10;step:1;unit:分" hide:"create,update"`
	ScoreType string `json:"score_type" gorm:"-" widget:"name:评分类型;type:select;options:推荐者,被动者,贬低者;options_colors:67C23A,409EFF,F56C6C" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	Reason    string `json:"reason" gorm:"column:reason;comment:推荐理由" widget:"name:推荐理由;type:text_area" hide:"create,update"`
}

func (NpsScoreRecord) TableName() string {
	return "nps_score_record"
}

// ================ 辅助函数 ================

// getScoreType 根据评分获取评分类型
func getScoreType(score int) string {
	if score >= 9 {
		return "推荐者"
	} else if score >= 7 {
		return "被动者"
	}
	return "贬低者"
}

// ================ 列表请求结构 ================

// NpsScoreListReq 评分记录列表请求
type NpsScoreListReq struct {
	QuestionnaireName string `json:"questionnaire_name" form:"questionnaire_name" gorm:"-" widget:"name:问卷名称;type:input"`
	ScoreType         string `json:"score_type" form:"score_type" gorm:"-" widget:"name:评分类型;type:select;options:推荐者,被动者,贬低者;options_colors:67C23A,409EFF,F56C6C"`
	CreatedBy         string `json:"created_by" form:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"`
	CreatedFrom       string `json:"created_from" form:"created_from" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedTo         string `json:"created_to" form:"created_to" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`

	query.PageSortReq `widget:"-"`
}

// ================ 评分记录列表 Handler ================

// NpsScoreList 评分记录列表
func NpsScoreList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req NpsScoreListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&NpsScoreRecord{}).Preload("Questionnaire")

	// 按问卷名称筛选
	if req.QuestionnaireName != "" {
		var questionnaireIDs []int
		if err := db.Model(&NpsQuestionnaire{}).
			Where("name LIKE ?", "%"+req.QuestionnaireName+"%").
			Pluck("id", &questionnaireIDs).Error; err == nil && len(questionnaireIDs) > 0 {
			queryDB = queryDB.Where("questionnaire_id IN ?", questionnaireIDs)
		} else {
			queryDB = queryDB.Where("1 = 0")
		}
	}

	// 按评分类型筛选
	if req.ScoreType != "" {
		switch req.ScoreType {
		case "推荐者":
			queryDB = queryDB.Where("score >= 9")
		case "被动者":
			queryDB = queryDB.Where("score >= 7 AND score <= 8")
		case "贬低者":
			queryDB = queryDB.Where("score <= 6")
		}
	}

	// 按创建人筛选
	if req.CreatedBy != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatedBy)
	}

	// 按创建时间范围筛选
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

	var records []NpsScoreRecord
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&records).Error; err != nil {
		return err
	}

	// 返回前填充关联字段和计算字段
	for i := range records {
		if records[i].Questionnaire != nil {
			records[i].QuestionnaireName = records[i].Questionnaire.Name
		}
		records[i].ScoreType = getScoreType(records[i].Score)
	}

	return resp.Table(response.TableResult{
		Items:      records,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// ================ 问卷模糊搜索回调 ================

// npsOnSelectFuzzyQuestionnaire 问卷模糊搜索回调
func npsOnSelectFuzzyQuestionnaire(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var questionnaires []NpsQuestionnaire

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("name LIKE ?", "%"+keyword+"%").Limit(20)
	}

	db.Find(&questionnaires)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, q := range questionnaires {
		status := getQuestionnaireStatus(q.StartTime, q.EndTime)
		items = append(items, &callback.SelectFuzzyItem{
			Value: q.ID,
			Label: fmt.Sprintf("%s - %s", q.Name, status),
			DisplayInfo: map[string]interface{}{
				"问卷名称": q.Name,
				"问卷说明": q.Description,
				"状态":   status,
				"时间范围": fmt.Sprintf("%s - %s",
					q.StartTime.Time().Format("2006-01-02 15:04"),
					q.EndTime.Time().Format("2006-01-02 15:04")),
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		Items: items,
	}, nil
}

// ================ TableTemplate（只读） ================

var NpsScoreListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "NPS评分记录",
		Desc:         "只读查看用户提交的评分记录。",
		Tags:         []string{"NPS", "评分记录"},
		Request:      &NpsScoreListReq{},
		CreateTables: []interface{}{&NpsScoreRecord{}},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"questionnaire_id": npsOnSelectFuzzyQuestionnaire,
		},
	},
	AutoCrudTable: &NpsScoreRecord{},
	// 评分记录只读，不配置增删改回调
}

// ================ API 注册 ================

func init() {
	packageContext.GET("nps_score_list.table", NpsScoreList, NpsScoreListTemplate)
}
