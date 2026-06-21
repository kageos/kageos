package agent_debate

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/gormx/query"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 数据模型 ================

// DebateMatch 辩论赛记录表
type DebateMatch struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`                                                        // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:创建时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	UpdatedAt     types.Time     `json:"updated_at" gorm:"column:updated_at;type:datetime;autoUpdateTime" widget:"name:更新时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	Topic         string         `json:"topic" gorm:"column:topic;comment:辩题" widget:"name:辩题;type:input" validate:"required"`
	PositiveSide  string         `json:"positive_side" gorm:"column:positive_side;comment:正方" widget:"name:正方;type:input" hide:"create,update"`
	NegativeSide  string         `json:"negative_side" gorm:"column:negative_side;comment:反方" widget:"name:反方;type:input" hide:"create,update"`
	TotalRounds   int            `json:"total_rounds" gorm:"column:total_rounds;comment:总回合数" widget:"name:总回合数;type:integer" hide:"create,update"`
	Status        string         `json:"status" gorm:"column:status;comment:状态" widget:"name:状态;type:select;options:待开始,进行中,已完成;options_colors:909399,409EFF,67C23A" hide:"create,update"`
	PositiveScore float64        `json:"positive_score" gorm:"column:positive_score;comment:正方最终得分" widget:"name:正方最终得分;type:float;min:0;max:10;precision:1" hide:"create,update"`
	NegativeScore float64        `json:"negative_score" gorm:"column:negative_score;comment:反方最终得分" widget:"name:反方最终得分;type:float;min:0;max:10;precision:1" hide:"create,update"`
	MatchResult   string         `json:"match_result" gorm:"column:match_result;comment:对决结果" widget:"name:对决结果;type:input" hide:"create,update"`
	JudgeScore    float64        `json:"judge_score" gorm:"column:judge_score;comment:裁判评分" widget:"name:裁判评分;type:float;min:0;max:10;precision:1" hide:"create,update"`
	RoundsLink    string         `json:"rounds_link" gorm:"-" widget:"name:回合详情;type:link;target:_blank" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	SpeechLink    string         `json:"speech_link" gorm:"-" widget:"name:发言;type:link;target:_blank" hide:"create,update"`   // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedBy     string         `json:"created_by" gorm:"column:created_by" widget:"name:创建人;type:user" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (DebateMatch) TableName() string {
	return "debate_match"
}

// ================ 辩论赛记录列表 ================

// DebateMatchListReq 辩论赛记录列表请求
type DebateMatchListReq struct {
	Topic       string `json:"topic" form:"topic" widget:"name:辩题;type:input"`
	Status      string `json:"status" form:"status" widget:"name:状态;type:select;options:待开始,进行中,已完成;options_colors:909399,409EFF,67C23A"`
	CreatorName string `json:"creator_name" form:"creator_name" widget:"name:创建人;type:user"`

	query.PageSortReq `widget:"-"`
}

// DebateMatchList 辩论赛记录列表
func DebateMatchList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req DebateMatchListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&DebateMatch{})
	if req.Topic != "" {
		queryDB = queryDB.Where("topic LIKE ?", "%"+req.Topic+"%")
	}
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}
	if req.CreatorName != "" {
		queryDB = queryDB.Where("created_by = ?", req.CreatorName)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var matches []DebateMatch
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&matches).Error; err != nil {
		return err
	}

	for i := range matches {
		// 回合详情链接
		params := DebateRound{DebateMatchID: matches[i].ID}
		matches[i].RoundsLink, _ = ctx.BuildFunctionUrlWithText("debate_round_list.table", params, "查看回合详情")

		// 发言链接
		submitParams := SubmitSpeechReq{DebateMatchID: matches[i].ID}
		matches[i].SpeechLink, _ = ctx.BuildFunctionUrlWithText("submit_speech.form", submitParams, "提交发言")
	}

	return resp.Table(response.TableResult{
		Items:      matches,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// DebateMatchListTemplate 辩论赛记录配置
var DebateMatchListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "辩论赛记录",
		Desc:         `记录每场辩论赛的基本信息和结果`,
		Tags:         []string{"辩论赛", "记录管理"},
		Request:      &DebateMatchListReq{},
		CreateTables: []interface{}{&DebateMatch{}, &DebateRound{}},
	},
	AutoCrudTable: &DebateMatch{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("debate_match_list.table", DebateMatchList, DebateMatchListTemplate)
}
