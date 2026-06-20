// nps_score_submit.go
// 提交NPS评分表单

package nps

import (
	"errors"
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/callback"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// NpsScoreSubmitReq 提交评分请求
type NpsScoreSubmitReq struct {
	QuestionnaireID int    `json:"questionnaire_id" widget:"name:选择问卷;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Score           int    `json:"score" widget:"name:评分;type:rate;max:10" validate:"required,min=0,max=10"`
	Reason          string `json:"reason" widget:"name:推荐理由;type:text_area;placeholder:选填，推荐该产品的理由或改进建议" validate:"max=500"`
}

// NpsScoreSubmitResp 提交评分响应
type NpsScoreSubmitResp struct {
	Success    bool   `json:"success" widget:"name:是否成功;type:switch"`
	Message    string `json:"message" widget:"name:提交结果;type:text_area"`
	ScoreType  string `json:"score_type" widget:"name:评分类型;type:input"`
	SubmitTime string `json:"submit_time" widget:"name:提交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// ================ 辅助函数 ================

// getScoreType 根据评分获取评分类型
func getScoreTypeFromScore(score int) string {
	if score >= 9 {
		return "推荐者"
	} else if score >= 7 {
		return "被动者"
	}
	return "贬低者"
}

// ================ 问卷模糊搜索回调（进行中问卷） ================

// npsOnSelectFuzzyActiveQuestionnaire 进行中问卷模糊搜索回调
func npsOnSelectFuzzyActiveQuestionnaire(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[npsOnSelectFuzzyActiveQuestionnaire] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[npsOnSelectFuzzyActiveQuestionnaire]：数据库连接失败")
	}

	now := time.Now()
	var questionnaires []NpsQuestionnaire

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("name LIKE ? AND start_time <= ? AND end_time > ?",
			"%"+keyword+"%", now, now).
			Limit(20)
	}

	db.Find(&questionnaires)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, q := range questionnaires {
		items = append(items, &callback.SelectFuzzyItem{
			Value: q.ID,
			Label: q.Name,
			DisplayInfo: map[string]interface{}{
				"问卷名称": q.Name,
				"问卷说明": q.Description,
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

// ================ 提交评分 Handler ================

// NpsScoreSubmit 提交评分入口
func NpsScoreSubmit(ctx *app.Context, resp response.Response) error {
	var req NpsScoreSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}

	res, err := DoNpsScoreSubmit(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoNpsScoreSubmit 提交评分业务逻辑
func DoNpsScoreSubmit(ctx *app.Context, req *NpsScoreSubmitReq) (*NpsScoreSubmitResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoNpsScoreSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoNpsScoreSubmit]：数据库连接失败")
	}

	userInfo := ctx.GetRequestUser()
	now := time.Now()

	// 检查问卷是否存在且处于进行中状态
	var questionnaire NpsQuestionnaire
	if err := db.Where("id = ?", req.QuestionnaireID).First(&questionnaire).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("问卷不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoNpsScoreSubmit] 查询问卷失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoNpsScoreSubmit]：查询问卷失败, err: %w", err)
	}

	status := getQuestionnaireStatus(questionnaire.StartTime, questionnaire.EndTime)
	if status != "进行中" {
		return nil, fmt.Errorf("该问卷状态为「%s」，当前不可提交评分", status)
	}

	// 检查是否已提交过
	var count int64
	if err := db.Model(&NpsScoreRecord{}).
		Where("questionnaire_id = ? AND created_by = ?", req.QuestionnaireID, userInfo).
		Count(&count).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoNpsScoreSubmit] 查询已提交记录失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoNpsScoreSubmit]：查询已提交记录失败, err: %w", err)
	}
	if count > 0 {
		return nil, fmt.Errorf("您已提交过该问卷的评分，每人每个问卷只能提交一次")
	}

	// 创建评分记录
	record := &NpsScoreRecord{
		QuestionnaireID: req.QuestionnaireID,
		Score:           req.Score,
		Reason:          req.Reason,
		CreatedBy:       userInfo,
	}
	record.CreatedAt = types.Time(now)

	if err := db.Create(record).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoNpsScoreSubmit] 创建评分记录失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoNpsScoreSubmit]：创建评分记录失败, err: %w", err)
	}

	return &NpsScoreSubmitResp{
		Success:    true,
		Message:    "评分成功",
		ScoreType:  getScoreTypeFromScore(req.Score),
		SubmitTime: now.Format("2006-01-02 15:04:05"),
	}, nil
}

// ================ FormTemplate ================

var NpsScoreSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交NPS评分",
		Desc:     "用户选择进行中的问卷后提交 0-10 分评分，自动判定评分类型并记录。",
		Tags:     []string{"NPS", "评分提交"},
		Request:  &NpsScoreSubmitReq{},
		Response: &NpsScoreSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"questionnaire_id": npsOnSelectFuzzyActiveQuestionnaire,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.POST("nps_score_submit.form", NpsScoreSubmit, NpsScoreSubmitTemplate)
}
