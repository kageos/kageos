// nps_trend_chart.go
// NPS趋势分析图表

package nps

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 请求结构 ================

// NpsTrendReq 趋势分析请求
type NpsTrendReq struct {
	QuestionnaireName string `json:"questionnaire_name" form:"questionnaire_name" widget:"name:问卷名称;type:select" callback:"OnSelectFuzzy"`
}

// ================ 图表 Handler ================

// NpsTrendChart NPS趋势分析
func NpsTrendChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req NpsTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&NpsScoreRecord{})

	// 按问卷名称筛选
	if req.QuestionnaireName != "" {
		var questionnaireIDs []int
		if err := db.Model(&NpsQuestionnaire{}).
			Where("name LIKE ?", "%"+req.QuestionnaireName+"%").
			Pluck("id", &questionnaireIDs).Error; err == nil && len(questionnaireIDs) > 0 {
			queryDB = queryDB.Where("questionnaire_id IN ?", questionnaireIDs)
		} else {
			// 无匹配问卷，返回空数据
			return resp.Chart(&chart.LineChart{
				Title: "NPS趋势分析",
				XAxis: []string{},
				Series: []chart.ChartSeries{
					{Name: "NPS分数", Data: []interface{}{}},
					{Name: "推荐者数", Data: []interface{}{}},
					{Name: "被动者数", Data: []interface{}{}},
					{Name: "贬低者数", Data: []interface{}{}},
				},
				Metadata: map[string]interface{}{
					"说明": "请选择问卷后查看趋势",
				},
			}).Build()
		}
	}

	// 按日期分组统计
	type DailyStat struct {
		Date       string
		Total      int
		Promoters  int // 推荐者 9-10
		Passives   int // 被动者 7-8
		Detractors int // 贬低者 0-6
	}

	var stats []DailyStat
	err := queryDB.
		Select("DATE(created_at) as date, COUNT(*) as total, " +
			"SUM(CASE WHEN score >= 9 THEN 1 ELSE 0 END) as promoters, " +
			"SUM(CASE WHEN score >= 7 AND score <= 8 THEN 1 ELSE 0 END) as passives, " +
			"SUM(CASE WHEN score <= 6 THEN 1 ELSE 0 END) as detractors").
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&stats).Error

	if err != nil {
		return fmt.Errorf("查询趋势数据失败: %w", err)
	}

	// 构造图表数据
	dates := make([]string, 0)
	npsScores := make([]interface{}, 0)
	promoters := make([]interface{}, 0)
	passives := make([]interface{}, 0)
	detractors := make([]interface{}, 0)

	var totalPromoters, totalPassives, totalDetractors, totalCount int

	for _, s := range stats {
		dates = append(dates, s.Date)
		totalPromoters += s.Promoters
		totalPassives += s.Passives
		totalDetractors += s.Detractors
		totalCount += s.Total

		// 计算当日 NPS
		var npsScore float64
		if s.Total > 0 {
			npsScore = float64(s.Promoters)*100/float64(s.Total) - float64(s.Detractors)*100/float64(s.Total)
		}
		npsScores = append(npsScores, npsScore)
		promoters = append(promoters, s.Promoters)
		passives = append(passives, s.Passives)
		detractors = append(detractors, s.Detractors)
	}

	// 计算总 NPS
	var overallNps float64
	if totalCount > 0 {
		overallNps = float64(totalPromoters)*100/float64(totalCount) - float64(totalDetractors)*100/float64(totalCount)
		overallNps = float64(int(overallNps*100+0.5)) / 100
	}

	return resp.Chart(&chart.LineChart{
		Title: "NPS趋势分析",
		XAxis: dates,
		Series: []chart.ChartSeries{
			{Name: "NPS分数", Data: npsScores},
			{Name: "推荐者数", Data: promoters},
			{Name: "被动者数", Data: passives},
			{Name: "贬低者数", Data: detractors},
		},
		Metadata: map[string]interface{}{
			"总评分人数": totalCount,
			"推荐者数":  totalPromoters,
			"被动者数":  totalPassives,
			"贬低者数":  totalDetractors,
			"NPS分数": overallNps,
			"说明":    "NPS = 推荐者占比 - 贬低者占比，范围 -100 到 100",
		},
	}).Build()
}

// ================ ChartTemplate ================

var NpsTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "NPS趋势分析",
		Desc:     "按日期查看 NPS 分数变化趋势，NPS = 推荐者占比 - 贬低者占比。",
		Tags:     []string{"NPS", "趋势分析"},
		Request:  &NpsTrendReq{},
		Response: &chart.LineChart{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"questionnaire_name": npsChartOnSelectFuzzyQuestionnaire,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("nps_trend_statistics.chart", NpsTrendChart, NpsTrendChartTemplate)
}
