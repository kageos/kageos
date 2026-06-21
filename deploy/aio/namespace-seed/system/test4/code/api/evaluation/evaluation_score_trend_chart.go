package evaluation

import (
	"fmt"
	"sort"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 评价得分趋势 ================

// EvaluationScoreTrendReq 评价得分趋势请求
type EvaluationScoreTrendReq struct {
	ActivityID int `json:"activity_id" form:"activity_id" widget:"name:评价活动;type:select" validate:"required" callback:"OnSelectFuzzy"`
}

// EvaluationScoreTrendChart 评价得分趋势（折线图）
func EvaluationScoreTrendChart(ctx *app.Context, resp response.Response) error {
	var req EvaluationScoreTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()

	// 查询活动名称
	var activity EvaluationActivity
	if err := db.Where("id = ?", req.ActivityID).First(&activity).Error; err != nil {
		return fmt.Errorf("评价活动不存在")
	}

	// 按日期聚合：每天的平均得分
	dateExpr, groupExpr := app.DateTimeBucketExpr(db, "created_at", app.TimeBucketDay)
	type dayStat struct {
		Date     string  `json:"date"`
		AvgScore float64 `json:"avg_score"`
	}
	var stats []dayStat
	if err := db.Table("evaluation_record").
		Select(fmt.Sprintf("%s as date, ROUND(AVG(average_score),1) as avg_score", dateExpr)).
		Where("activity_id = ? AND deleted_at IS NULL", req.ActivityID).
		Group(groupExpr).
		Order("date ASC").
		Scan(&stats).Error; err != nil {
		return fmt.Errorf("[系统错误]-[EvaluationScoreTrendChart] 查询得分趋势失败, err: %v", err)
	}

	// 按日期排序
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].Date < stats[j].Date
	})

	labels := make([]string, 0, len(stats))
	avgData := make([]interface{}, 0, len(stats))
	for _, s := range stats {
		labels = append(labels, s.Date)
		avgData = append(avgData, s.AvgScore)
	}

	c := &chart.LineChart{
		Title: fmt.Sprintf("评价得分趋势 - %s", activity.Name),
		XAxis: labels,
		Series: []chart.ChartSeries{
			{Name: "平均得分", Data: avgData},
		},
		Metadata: map[string]interface{}{
			"评价活动": activity.Name,
		},
	}
	return resp.Chart(c).Build()
}

// EvaluationScoreTrendChartTemplate 得分趋势配置
var EvaluationScoreTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "评价得分趋势",
		Request:  &EvaluationScoreTrendReq{},
		Response: &chart.LineChart{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"activity_id": evaluationOnSelectFuzzyActivity,
		},
	},
	ChartType: app.ChartTypeLine,
}

func init() {
	packageContext.GET("evaluation_score_trend.chart", EvaluationScoreTrendChart, EvaluationScoreTrendChartTemplate)
}
