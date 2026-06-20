package agent_debate

import (
	"errors"
	"fmt"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// JudgeScoreReq 裁判评分请求
type JudgeScoreReq struct {
	DebateMatchID int     `json:"debate_match_id" widget:"name:辩论赛ID;type:integer" validate:"required"`
	PositiveScore float64 `json:"positive_score" widget:"name:正方评分;type:float;min:0;max:10;step:0.5;render_default:8" validate:"required,gte=0,lte=10"`
	NegativeScore float64 `json:"negative_score" widget:"name:反方评分;type:float;min:0;max:10;step:0.5;render_default:8" validate:"required,gte=0,lte=10"`
	JudgeScore    float64 `json:"judge_score" widget:"name:综合评分;type:float;min:0;max:10;step:0.5;render_default:8" validate:"gte=0,lte=10"`
	Comment       string  `json:"comment" widget:"name:精彩点评;type:text_area;placeholder:请写下您的犀利点评..."`
}

// JudgeScoreResp 裁判评分响应
type JudgeScoreResp struct {
	DebateMatchID int     `json:"debate_match_id" widget:"name:辩论赛ID;type:integer"`
	PositiveScore float64 `json:"positive_score" widget:"name:正方最终得分;type:float;min:0;max:10"`
	NegativeScore float64 `json:"negative_score" widget:"name:反方最终得分;type:float;min:0;max:10"`
	MatchResult   string  `json:"match_result" widget:"name:对决结果;type:input"`
	JudgeScore    float64 `json:"judge_score" widget:"name:综合评分;type:float;min:0;max:10"`
	Comment       string  `json:"comment" widget:"name:点评内容;type:text_area"`
	Topic         string  `json:"topic" widget:"name:辩题;type:input"`
	Winner        string  `json:"winner" widget:"name:获胜方;type:input"`
}

// ================ 裁判评分 ================

// JudgeScore 裁判评分入口
func JudgeScore(ctx *app.Context, resp response.Response) error {
	var req JudgeScoreReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoJudgeScore(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoJudgeScore 裁判评分业务逻辑
func DoJudgeScore(ctx *app.Context, req *JudgeScoreReq) (*JudgeScoreResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoJudgeScore] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoJudgeScore]：数据库连接失败")
	}

	// 查询辩论赛记录
	var match DebateMatch
	if err := db.Where("id = ?", req.DebateMatchID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("辩论赛不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoJudgeScore] 查询辩论赛失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoJudgeScore]：查询辩论赛失败, err: %w", err)
	}

	// 检查辩论赛是否已完成
	if match.Status == "已完成" {
		return nil, fmt.Errorf("辩论赛已经评分结束，不能重复评分")
	}

	// 确定对决结果
	diff := req.PositiveScore - req.NegativeScore
	var matchResult string
	var winner string

	if diff >= 1.0 {
		matchResult = "正方获胜"
		winner = match.PositiveSide
	} else if diff <= -1.0 {
		matchResult = "反方获胜"
		winner = match.NegativeSide
	} else {
		matchResult = "平局"
		winner = "双方"
	}

	// 默认综合评分
	judgeScore := req.JudgeScore
	if judgeScore == 0 {
		judgeScore = (req.PositiveScore + req.NegativeScore) / 2
	}

	// 更新辩论赛记录
	updates := map[string]interface{}{
		"status":         "已完成",
		"positive_score": req.PositiveScore,
		"negative_score": req.NegativeScore,
		"match_result":   matchResult,
		"judge_score":    judgeScore,
	}

	if err := db.Model(&match).Updates(updates).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoJudgeScore] 更新辩论赛记录失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoJudgeScore]：更新辩论赛记录失败, err: %w", err)
	}

	// 点评内容
	comment := req.Comment
	if comment == "" {
		if matchResult == "正方获胜" {
			comment = fmt.Sprintf("正方 %s 的表现令人印象深刻！论点清晰有力，论据充分有据，赢得了裁判的认可！", match.PositiveSide)
		} else if matchResult == "反方获胜" {
			comment = fmt.Sprintf("反方 %s 展现了出色的逻辑思维！数据详实、论证严密，值得称赞！", match.NegativeSide)
		} else {
			comment = "双方势均力敌，各有千秋！这是一场精彩的对决！"
		}
	}

	return &JudgeScoreResp{
		DebateMatchID: req.DebateMatchID,
		PositiveScore: req.PositiveScore,
		NegativeScore: req.NegativeScore,
		MatchResult:   matchResult,
		JudgeScore:    judgeScore,
		Comment:       comment,
		Topic:         match.Topic,
		Winner:        winner,
	}, nil
}

// JudgeScoreTemplate 裁判评分配置
var JudgeScoreTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "裁判评分",
		Desc:     `辩论结束后，裁判对正反双方进行评分，并宣布最终结果`,
		Tags:     []string{"辩论赛", "评分"},
		Request:  &JudgeScoreReq{},
		Response: &JudgeScoreResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("judge_score.form", JudgeScore, JudgeScoreTemplate)
}
