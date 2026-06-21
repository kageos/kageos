package agent_debate

import (
	"errors"
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// SubmitSpeechReq 提交辩论发言请求
type SubmitSpeechReq struct {
	DebateMatchID int    `json:"debate_match_id" widget:"name:辩论赛ID;type:integer"`
	Round         int    `json:"round" widget:"name:回合;type:integer;min:1;render_default:1"`
	UserIdentity  string `json:"user_identity" widget:"name:您的身份;type:input;placeholder:请输入您的名字或昵称"`
	Stance        string `json:"stance" widget:"name:立场;type:select;options:正方,反方;options_colors:67C23A,F56C6C;render_default:正方"`
	SpeechContent string `json:"speech_content" widget:"name:您的观点;type:text_area;placeholder:请在这里写下您的观点、论据和论证过程..."`
}

// SubmitSpeechResp 提交辩论发言响应
type SubmitSpeechResp struct {
	DebateMatchID int    `json:"debate_match_id" widget:"name:辩论赛ID;type:integer"`
	UserIdentity  string `json:"user_identity" widget:"name:发言人身份;type:input"`
	Round         int    `json:"round" widget:"name:回合;type:integer"`
	Stance        string `json:"stance" widget:"name:立场;type:input"`
	SpeechContent string `json:"speech_content" widget:"name:发言内容;type:text_area"`
	Topic         string `json:"topic" widget:"name:辩题;type:input"`
	Message       string `json:"message" widget:"name:提示信息;type:text_area"`
}

// ================ 提交辩论发言 ================

// SubmitSpeech 提交辩论发言入口
func SubmitSpeech(ctx *app.Context, resp response.Response) error {
	var req SubmitSpeechReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoSubmitSpeech(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoSubmitSpeech 提交辩论发言业务逻辑
func DoSubmitSpeech(ctx *app.Context, req *SubmitSpeechReq) (*SubmitSpeechResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoSubmitSpeech] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoSubmitSpeech]：数据库连接失败")
	}

	// 校验必填字段
	if req.UserIdentity == "" {
		return nil, fmt.Errorf("请输入您的身份/名字")
	}
	if req.SpeechContent == "" {
		return nil, fmt.Errorf("请输入您的观点")
	}

	// 查询辩论赛记录
	var match DebateMatch
	if err := db.Where("id = ?", req.DebateMatchID).First(&match).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("辩论赛不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoSubmitSpeech] 查询辩论赛失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoSubmitSpeech]：查询辩论赛失败, err: %w", err)
	}

	// 检查辩论赛状态
	if match.Status == "已完成" {
		return nil, fmt.Errorf("辩论赛已结束，无法继续发言")
	}

	// 检查回合数是否超过总回合数
	if req.Round > match.TotalRounds {
		return nil, fmt.Errorf("回合数不能超过总回合数%d", match.TotalRounds)
	}

	// 获取当前系统用户
	speaker := ctx.GetRequestUser()

	// 创建回合记录
	round := &DebateRound{
		DebateMatchID: req.DebateMatchID,
		Round:         req.Round,
		UserIdentity:  req.UserIdentity,
		Speaker:       speaker,
		Stance:        req.Stance,
		SpeechContent: req.SpeechContent,
	}

	if err := db.Create(round).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoSubmitSpeech] 创建回合记录失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoSubmitSpeech]：创建回合记录失败, err: %w", err)
	}

	// 更新辩论赛状态为进行中
	if match.Status == "待开始" {
		db.Model(&match).Update("status", "进行中")
	}

	// 统计当前回合数
	var currentRoundCount int64
	db.Model(&DebateRound{}).Where("debate_match_id = ?", req.DebateMatchID).Count(&currentRoundCount)

	message := fmt.Sprintf("发言已提交！\n发言人：%s\n立场：%s\n回合：%d/%d\n\n当前已发言%d次。", req.UserIdentity, req.Stance, req.Round, match.TotalRounds, currentRoundCount)

	if currentRoundCount >= int64(match.TotalRounds*2) {
		message += "\n\n辩论已全部完成，请使用「裁判评分」表单进行评分。"
	}

	return &SubmitSpeechResp{
		DebateMatchID: req.DebateMatchID,
		UserIdentity:  req.UserIdentity,
		Round:         req.Round,
		Stance:        req.Stance,
		SpeechContent: req.SpeechContent,
		Topic:         match.Topic,
		Message:       message,
	}, nil
}

// SubmitSpeechTemplate 提交辩论发言配置
var SubmitSpeechTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交辩论发言",
		Desc:     `提交某场辩论赛的发言内容，表达您的观点和立场`,
		Tags:     []string{"辩论赛", "发言"},
		Request:  &SubmitSpeechReq{},
		Response: &SubmitSpeechResp{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("submit_speech.form", SubmitSpeech, SubmitSpeechTemplate)
}
