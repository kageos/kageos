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

// DebateRound 辩论回合记录表
type DebateRound struct {
	ID            int            `json:"id" gorm:"primaryKey;autoIncrement;column:id" widget:"name:ID;type:ID" hide:"create,update"`                                                        // 前端仅在列表展示，不进入新增/编辑表单。
	CreatedAt     types.Time     `json:"created_at" gorm:"column:created_at;type:datetime;autoCreateTime" widget:"name:发言时间;type:datetime;format:YYYY-MM-DD HH:mm:ss" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	DeletedAt     gorm.DeletedAt `json:"deleted_at" gorm:"index;column:deleted_at" widget:"-"`
	DebateMatchID int            `json:"debate_match_id" gorm:"column:debate_match_id;comment:辩论赛ID" widget:"name:辩论赛ID;type:integer"`
	Round         int            `json:"round" gorm:"column:round;comment:回合" widget:"name:回合;type:integer"`
	UserIdentity  string         `json:"user_identity" gorm:"column:user_identity;comment:发言人身份" widget:"name:发言人身份;type:input;placeholder:请输入您的名字或昵称"`
	Speaker       string         `json:"speaker" gorm:"column:speaker;comment:系统用户" widget:"name:提交人;type:user" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
	Stance        string         `json:"stance" gorm:"column:stance;comment:立场" widget:"name:立场;type:select;options:正方,反方;options_colors:67C23A,F56C6C"`
	SpeechContent string         `json:"speech_content" gorm:"column:speech_content;comment:发言内容" widget:"name:发言内容;type:text_area;placeholder:请在这里写下您的观点和论据..."`
	Match         *DebateMatch   `json:"-" gorm:"foreignKey:DebateMatchID" widget:"-"`
	Topic         string         `json:"topic" gorm:"-" widget:"name:辩题;type:text" hide:"create,update"` // 前端仅在列表展示，不进入新增/编辑表单。
}

func (DebateRound) TableName() string {
	return "debate_round"
}

// ================ 辩论回合记录列表 ================

// DebateRoundListReq 辩论回合记录列表请求
type DebateRoundListReq struct {
	DebateMatchID int    `json:"debate_match_id" form:"debate_match_id" widget:"name:辩论赛ID;type:integer"`
	Stance        string `json:"stance" form:"stance" widget:"name:立场;type:select;options:正方,反方;options_colors:67C23A,F56C6C"`
	Round         int    `json:"round" form:"round" widget:"name:回合;type:integer"`
	Speaker       string `json:"speaker" form:"speaker" widget:"name:提交人;type:user"`

	query.PageSortReq `widget:"-"`
}

// DebateRoundList 辩论回合记录列表
func DebateRoundList(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req DebateRoundListReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&DebateRound{}).Preload("Match")
	if req.DebateMatchID > 0 {
		queryDB = queryDB.Where("debate_match_id = ?", req.DebateMatchID)
	}
	if req.Stance != "" {
		queryDB = queryDB.Where("stance = ?", req.Stance)
	}
	if req.Round > 0 {
		queryDB = queryDB.Where("round = ?", req.Round)
	}
	if req.Speaker != "" {
		queryDB = queryDB.Where("speaker = ?", req.Speaker)
	}

	if order := req.PageSortReq.GetOrder(); order != "" {
		queryDB = queryDB.Order(order)
	} else {
		queryDB = queryDB.Order("debate_match_id ASC, round ASC")
	}
	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		return err
	}

	var rounds []DebateRound
	if err := queryDB.Offset(req.PageSortReq.GetOffset()).Limit(req.PageSortReq.GetLimit()).Find(&rounds).Error; err != nil {
		return err
	}

	for i := range rounds {
		if rounds[i].Match != nil && rounds[i].Match.ID > 0 {
			rounds[i].Topic = rounds[i].Match.Topic
		}
	}

	return resp.Table(response.TableResult{
		Items:      rounds,
		TotalCount: total,
		PageInfo:   &req.PageSortReq,
	}).Build()
}

// DebateRoundListTemplate 辩论回合记录配置
var DebateRoundListTemplate = &app.TableTemplate{
	BaseConfig: app.BaseConfig{
		Name:         "辩论回合记录",
		Desc:         `记录辩论赛中每个回合的发言内容`,
		Tags:         []string{"辩论赛", "回合记录"},
		Request:      &DebateRoundListReq{},
		CreateTables: []interface{}{&DebateRound{}},
	},
	AutoCrudTable: &DebateRound{},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("debate_round_list.table", DebateRoundList, DebateRoundListTemplate)
}
