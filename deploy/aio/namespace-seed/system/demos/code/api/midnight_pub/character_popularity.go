package midnight_pub

import (
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// CharacterPopularityReq 角色人气榜请求
type CharacterPopularityReq struct {
}

func CharacterPopularityChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()

	// 查询所有角色并按出场次数排序
	var characters []PubCharacter
	if err := db.Model(&PubCharacter{}).Order("appear_count DESC").Find(&characters).Error; err != nil {
		logger.Errorf(ctx, "CharacterPopularityChart Query err: %v", err)
		return err
	}

	// 构建图表数据
	var names []string
	var counts []interface{}

	for _, char := range characters {
		names = append(names, char.CharacterName)
		counts = append(counts, char.AppearCount)
	}

	// 如果没有数据，使用示例数据
	if len(names) == 0 {
		names = []string{"程序员小李", "文艺青年阿诗", "神秘人"}
		counts = []interface{}{45, 38, 12}
	}

	// 计算总出场次数
	var totalAppearances int
	for _, char := range characters {
		totalAppearances += char.AppearCount
	}
	if totalAppearances == 0 {
		totalAppearances = 95
	}

	c := &chart.BarChart{
		Title:  "角色人气榜",
		XAxis:  names,
		Series: []chart.ChartSeries{{Name: "出场次数", Data: counts}},
		Metadata: map[string]interface{}{
			"总出场次数": totalAppearances,
		},
	}

	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("character_popularity.chart", CharacterPopularityChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "角色人气榜",
			Request:  &CharacterPopularityReq{},
			Response: &chart.BarChart{},
		},
		ChartType: app.ChartTypeBar,
	})
}
