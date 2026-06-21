package evaluation

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
	"gorm.io/gorm"
)

// ================ 请求/响应结构 ================

// EvaluationScoreRow 维度评分子表行
type EvaluationScoreRow struct {
	DimensionID   int    `json:"dimension_id" widget:"name:评价维度;type:integer" validate:"required"`
	DimensionName string `json:"dimension_name" widget:"name:维度名称;type:text"`
	Score         int    `json:"score" widget:"name:评分(1-5);type:integer;min:1;max:5;step:1" validate:"required,min=1,max=5"`
}

// EvaluationSubmitReq 提交评价请求
type EvaluationSubmitReq struct {
	ActivityID int                  `json:"activity_id" widget:"name:评价活动;type:select" validate:"required" callback:"OnSelectFuzzy"`
	Topic      string               `json:"topic" widget:"name:评价主题;type:input;placeholder:例如某位员工、某个产品" validate:"required,min=1,max=100"`
	Scores     []EvaluationScoreRow `json:"scores" widget:"name:维度评分;type:table" validate:"required,min=1"`
	Comment    string               `json:"comment" widget:"name:总体评价;type:text_area;placeholder:对本次评价的补充说明或意见"`
}

// EvaluationSubmitResp 提交评价响应
type EvaluationSubmitResp struct {
	Result       string  `json:"result" widget:"name:提交结果;type:text"`
	SubmitTime   string  `json:"submit_time" widget:"name:提交时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	AverageScore float64 `json:"average_score" widget:"name:平均得分;type:float;precision:1;min:0;max:5"`
}

// ================ 提交评价 ================

// EvaluationSubmit 提交评价入口
func EvaluationSubmit(ctx *app.Context, resp response.Response) error {
	var req EvaluationSubmitReq
	if err := ctx.ShouldBindValidate(&req); err != nil {
		return err
	}
	res, err := DoEvaluationSubmit(ctx, &req)
	if err != nil {
		return err
	}
	return resp.Form(res).Build()
}

// DoEvaluationSubmit 提交评价业务逻辑
func DoEvaluationSubmit(ctx *app.Context, req *EvaluationSubmitReq) (*EvaluationSubmitResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		logger.Errorf(ctx, "[系统错误]-[DoEvaluationSubmit] 数据库连接失败, req: %+v", req)
		return nil, fmt.Errorf("[系统错误]-[DoEvaluationSubmit] 数据库连接失败, req: %+v", req)
	}

	userInfo := ctx.GetRequestUser()
	now := time.Now()

	// 1. 查询评价活动
	var activity EvaluationActivity
	if err := db.Where("id = ?", req.ActivityID).First(&activity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("评价活动不存在")
		}
		logger.Errorf(ctx, "[系统错误]-[DoEvaluationSubmit] 查询活动失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoEvaluationSubmit] 查询活动失败, req: %+v, err: %w", req, err)
	}

	// 2. 校验活动状态为进行中
	status := getActivityStatus(activity.StartTime, activity.EndTime)
	if status != "进行中" {
		return nil, fmt.Errorf("评价活动当前状态为「%s」，仅进行中的活动可以提交评价", status)
	}

	// 3. 校验活动维度数量 >= 2
	var dimCount int64
	if err := db.Model(&EvaluationDimension{}).Where("activity_id = ?", req.ActivityID).Count(&dimCount).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoEvaluationSubmit] 查询维度数量失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoEvaluationSubmit] 查询维度数量失败, err: %w", err)
	}
	if dimCount < 2 {
		return nil, fmt.Errorf("该评价活动下的评价维度不足 2 个，无法开始评价，请先添加评价维度")
	}

	// 4. 校验重复提交：同一评价人对同一评价活动和评价主题只能提交一次
	var existCount int64
	if err := db.Model(&EvaluationRecord{}).Where("activity_id = ? AND topic = ? AND evaluator = ?", req.ActivityID, req.Topic, userInfo).Count(&existCount).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoEvaluationSubmit] 查询重复评价失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoEvaluationSubmit] 查询重复评价失败, err: %w", err)
	}
	if existCount > 0 {
		return nil, fmt.Errorf("您已对「%s」在「%s」活动中提交过评价，不能重复提交", req.Topic, activity.Name)
	}

	// 5. 校验评分合法性并计算平均分
	totalScore := 0
	for _, s := range req.Scores {
		if s.Score < 1 || s.Score > 5 {
			return nil, fmt.Errorf("维度「%s」的评分必须在 1-5 之间，当前值: %d", s.DimensionName, s.Score)
		}
		totalScore += s.Score
	}
	avgScore := math.Round(float64(totalScore)/float64(len(req.Scores))*10) / 10

	// 6. 事务：创建评价记录 + 评分明细 + 更新活动统计
	var record EvaluationRecord
	err := db.Transaction(func(tx *gorm.DB) error {
		record = EvaluationRecord{
			ActivityID:   req.ActivityID,
			ActivityName: activity.Name,
			Topic:        req.Topic,
			Evaluator:    userInfo,
			AverageScore: avgScore,
			Comment:      req.Comment,
		}
		if err := tx.Create(&record).Error; err != nil {
			return fmt.Errorf("创建评价记录失败: %v", err)
		}

		// 创建评分明细
		details := make([]*EvaluationScoreDetail, 0, len(req.Scores))
		for _, s := range req.Scores {
			details = append(details, &EvaluationScoreDetail{
				RecordID:      record.ID,
				ActivityID:    req.ActivityID,
				ActivityName:  activity.Name,
				Topic:         req.Topic,
				DimensionName: s.DimensionName,
				Score:         float64(s.Score),
				Evaluator:     userInfo,
			})
		}
		if len(details) > 0 {
			if err := tx.Create(&details).Error; err != nil {
				return fmt.Errorf("创建评分明细失败: %v", err)
			}
		}

		// 更新活动统计：评价人数+1，平均得分重新计算
		var newCount int64
		tx.Model(&EvaluationRecord{}).Where("activity_id = ?", req.ActivityID).Count(&newCount)

		var totalAvg float64
		tx.Model(&EvaluationRecord{}).Where("activity_id = ?", req.ActivityID).
			Select("COALESCE(AVG(average_score),0)").Scan(&totalAvg)
		totalAvg = math.Round(totalAvg*10) / 10

		if err := tx.Model(&EvaluationActivity{}).Where("id = ?", req.ActivityID).
			Updates(map[string]interface{}{
				"eval_count":    newCount,
				"average_score": totalAvg,
			}).Error; err != nil {
			return fmt.Errorf("更新活动统计失败: %v", err)
		}
		return nil
	})

	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[DoEvaluationSubmit] 事务失败, req: %+v, err: %v", req, err)
		return nil, fmt.Errorf("[系统错误]-[DoEvaluationSubmit] 事务失败, req: %+v, err: %w", req, err)
	}

	return &EvaluationSubmitResp{
		Result:       "评价提交成功",
		SubmitTime:   types.Time(now).Time().Format("2006-01-02 15:04:05"),
		AverageScore: avgScore,
	}, nil
}

// EvaluationSubmitTemplate 提交评价配置
var EvaluationSubmitTemplate = &app.FormTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "提交评价",
		Desc:     "用户选择进行中的评价活动，对各维度进行 1-5 分打分并提交评价意见。",
		Request:  &EvaluationSubmitReq{},
		Response: &EvaluationSubmitResp{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"activity_id": evaluationOnSelectFuzzyActivityForSubmit,
		},
	},
}

func init() {
	packageContext.POST("evaluation_submit.form", EvaluationSubmit, EvaluationSubmitTemplate)
}
