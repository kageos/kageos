package agent_debate

import (
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 请求/响应结构 ================

// CreateDebateReq 创建辩论赛请求
type CreateDebateReq struct {
	Topic  string `json:"topic" widget:"name:辩题;type:input;placeholder:请输入辩论话题，例如：996是否应该被抵制" validate:"required,min=2,max=500"`
	Rounds int    `json:"rounds" widget:"name:回合数;type:integer;min:1;max:5;step:1;render_default:3;unit:回合" validate:"min=1,max=5"`
}

// CreateDebateResp 创建辩论赛响应
type CreateDebateResp struct {
	DebateMatchID int    `json:"debate_match_id" widget:"name:辩论赛ID;type:integer"`
	Topic         string `json:"topic" widget:"name:辩题;type:input"`
	Status        string `json:"status" widget:"name:状态;type:select;options:待开始,进行中,已完成;options_colors:909399,409EFF,67C23A"`
	Message       string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 创建辩论赛 ================

// CreateDebate 创建辩论赛入口
func CreateDebate(ctx *app.Context, resp response.Response) error {
	var req CreateDebateReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoCreateDebate(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoCreateDebate 创建辩论赛业务逻辑
func DoCreateDebate(ctx *app.Context, req *CreateDebateReq) (*CreateDebateResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoCreateDebate] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoCreateDebate]：数据库连接失败")
	}

	// 默认3回合
	rounds := req.Rounds
	if rounds <= 0 {
		rounds = 3
	}

	// 创建辩论赛记录
	match := &DebateMatch{
		Topic:        req.Topic,
		PositiveSide: "感性派老王",
		NegativeSide: "理性派小李",
		TotalRounds:  rounds,
		Status:       "待开始",
	}

	if err := db.Create(match).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoCreateDebate] 创建辩论赛记录失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoCreateDebate]：创建辩论赛记录失败, req: %+v, err: %w", req, err)
	}

	message := fmt.Sprintf("辩论赛已创建！\n辩题：%s\n正方：感性派老王\n反方：理性派小李\n总回合数：%d\n\n提示：使用「提交辩论发言」表单让AI角色进行辩论，完成后使用「裁判评分」表单进行打分。", req.Topic, rounds)

	return &CreateDebateResp{
		DebateMatchID: match.ID,
		Topic:         req.Topic,
		Status:        "待开始",
		Message:       message,
	}, nil
}

// CreateDebateTemplate 创建辩论赛配置
var CreateDebateTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "创建辩论赛",
		Desc:     `用户输入辩题，创建一场新的辩论赛，系统自动分配正反方角色`,
		Tags:     []string{"辩论赛", "创建"},
		Request:  &CreateDebateReq{},
		Response: &CreateDebateResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("create_debate.form", CreateDebate, CreateDebateTemplate)
}
