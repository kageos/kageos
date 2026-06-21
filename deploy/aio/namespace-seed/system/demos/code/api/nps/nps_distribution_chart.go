// nps_distribution_chart.go
// 评分类型分布图表

package nps

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/callback"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 请求结构 ================

// NpsDistributionReq 分布分析请求
type NpsDistributionReq struct {
	QuestionnaireName string `json:"questionnaire_name" form:"questionnaire_name" widget:"name:问卷名称;type:select" callback:"OnSelectFuzzy"`
	CreatedFrom       string `json:"created_from" form:"created_from" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	CreatedTo         string `json:"created_to" form:"created_to" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// ================ 问卷模糊搜索回调（Chart用） ================

// npsChartOnSelectFuzzyQuestionnaire 问卷模糊搜索回调
func npsChartOnSelectFuzzyQuestionnaire(ctx *app.Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error) {
	db := ctx.GetGormDB()
	if db == nil {
		return nil, fmt.Errorf("数据库连接失败")
	}

	var questionnaires []NpsQuestionnaire

	if req.IsByValue() {
		db = db.Where("id = ?", req.GetValue()).Limit(1)
	} else if req.IsByValues() {
		db = db.Where("id in ?", req.GetValues())
	} else {
		keyword := req.Keyword()
		db = db.Where("name LIKE ?", "%"+keyword+"%").Limit(20)
	}

	db.Find(&questionnaires)

	items := make([]*callback.SelectFuzzyItem, 0)
	for _, q := range questionnaires {
		items = append(items, &callback.SelectFuzzyItem{
			Value: q.ID,
			Label: q.Name,
			DisplayInfo: map[string]interface{}{
				"问卷名称": q.Name,
				"状态":   getQuestionnaireStatus(q.StartTime, q.EndTime),
			},
		})
	}

	return &callback.OnSelectFuzzyResp{
		Items: items,
	}, nil
}

// ================ 图表 Handler ================

// NpsDistributionChart 评分类型分布
func NpsDistributionChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req NpsDistributionReq
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
			return resp.Chart(&chart.PieChart{
				Title: "评分类型分布",
				Series: []chart.ChartSeries{
					{
						Name: "评分类型",
						Data: []interface{}{},
					},
				},
				Metadata: map[string]interface{}{
					"说明": "请选择问卷后查看分布",
				},
			}).Build()
		}
	}

	// 按时间范围筛选
	if req.CreatedFrom != "" {
		queryDB = queryDB.Where("created_at >= ?", req.CreatedFrom)
	}
	if req.CreatedTo != "" {
		queryDB = queryDB.Where("created_at <= ?", req.CreatedTo)
	}

	// 统计各类型数量
	type TypeStat struct {
		ScoreType string
		Count     int
	}

	var stats []TypeStat
	err := queryDB.
		Select("CASE WHEN score >= 9 THEN '推荐者' WHEN score >= 7 THEN '被动者' ELSE '贬低者' END as score_type, COUNT(*) as count").
		Group("score_type").
		Scan(&stats).Error

	if err != nil {
		return fmt.Errorf("查询分布数据失败: %w", err)
	}

	// 构造饼图数据
	pieData := make([]interface{}, 0)
	var totalCount int
	typeCounts := map[string]int{"推荐者": 0, "被动者": 0, "贬低者": 0}

	for _, s := range stats {
		typeCounts[s.ScoreType] = s.Count
		totalCount += s.Count
	}

	// 按固定顺序输出
	order := []string{"推荐者", "被动者", "贬低者"}
	for _, t := range order {
		count := typeCounts[t]
		pieData = append(pieData, map[string]interface{}{
			"name":  t,
			"value": count,
		})
	}

	return resp.Chart(&chart.PieChart{
		Title: "评分类型分布",
		Series: []chart.ChartSeries{
			{
				Name: "评分类型",
				Data: pieData,
			},
		},
		Metadata: map[string]interface{}{
			"总评分人数": totalCount,
			"推荐者数":  typeCounts["推荐者"],
			"被动者数":  typeCounts["被动者"],
			"贬低者数":  typeCounts["贬低者"],
		},
	}).Build()
}

// ================ ChartTemplate ================

var NpsDistributionChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "评分类型分布",
		Desc:     "查看评分类型分布比例：推荐者、被动者、贬低者各占百分比。",
		Tags:     []string{"NPS", "分布分析"},
		Request:  &NpsDistributionReq{},
		Response: &chart.PieChart{},
		OnSelectFuzzyMap: map[string]app.OnSelectFuzzy{
			"questionnaire_name": npsChartOnSelectFuzzyQuestionnaire,
		},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("nps_distribution_statistics.chart", NpsDistributionChart, NpsDistributionChartTemplate)
}
