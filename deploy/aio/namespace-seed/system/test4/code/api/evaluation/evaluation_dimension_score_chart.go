package evaluation

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 评价维度得分分布 ================

// EvaluationDimensionScoreReq 维度得分分布请求
type EvaluationDimensionScoreReq struct {
	ActivityID int `json:"activity_id" form:"activity_id" widget:"name:评价活动;type:select" validate:"required" callback:"OnSelectFuzzy"`
}

// EvaluationDimensionScoreChart 评价维度得分分布（柱状图）
func EvaluationDimensionScoreChart(ctx *app.Context, resp response.Response) error {
	var req EvaluationDimensionScoreReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()

	// 查询活动名称
	var activity EvaluationActivity
	if err := db.Where("id = ?", req.ActivityID).First(&activity).Error; err != nil {
		return fmt.Errorf("评价活动不存在")
	}

	// 按维度聚合：平均得分、评价人数（去重 evaluator）
	type dimStat struct {
		DimensionName string  `json:"dimension_name"`
		AvgScore      float64 `json:"avg_score"`
		EvalCount     int64   `json:"eval_count"`
	}
	var stats []dimStat
	if err := db.Table("evaluation_score_detail").
		Select("dimension_name, ROUND(AVG(score),1) as avg_score, COUNT(DISTINCT evaluator) as eval_count").
		Where("activity_id = ? AND deleted_at IS NULL", req.ActivityID).
		Group("dimension_name").
		Order("dimension_name").
		Scan(&stats).Error; err != nil {
		return fmt.Errorf("[系统错误]-[EvaluationDimensionScoreChart] 查询维度得分失败, err: %v", err)
	}

	labels := make([]string, 0, len(stats))
	avgData := make([]interface{}, 0, len(stats))
	countData := make([]interface{}, 0, len(stats))
	for _, s := range stats {
		labels = append(labels, s.DimensionName)
		avgData = append(avgData, s.AvgScore)
		countData = append(countData, s.EvalCount)
	}

	c := &chart.BarChart{
		Title: fmt.Sprintf("评价维度得分分布 - %s", activity.Name),
		XAxis: labels,
		Series: []chart.ChartSeries{
			{Name: "平均得分", Data: avgData},
			{Name: "评价人数", Data: countData},
		},
		Metadata: map[string]interface{}{
			"评价活动": activity.Name,
			"维度数量": len(stats),
		},
	}
	return resp.Chart(c).Build()
}

// EvaluationDimensionScoreChartTemplate 维度得分分布配置
var EvaluationDimensionScoreChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "评价维度得分分布",
		Request:  &EvaluationDimensionScoreReq{},
		Response: &chart.BarChart{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"activity_id": evaluationOnSelectFuzzyActivity,
		},
	},
	ChartType: app.ChartTypeBar,
}

func init() {
	packageContext.GET("evaluation_dimension_score.chart", EvaluationDimensionScoreChart, EvaluationDimensionScoreChartTemplate)
}
