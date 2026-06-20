package midnight_pub

import (
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// DrinkPopularityReq 酒单热力图请求
type DrinkPopularityReq struct {
}

// drinkNameCN 中文酒名映射
var drinkNameCN = map[string]string{
	"whiskey_neat":  "纯威士忌",
	"whiskey_sour":  "威士忌酸",
	"martini":       "马天尼",
	"beer":          "啤酒",
	"mojito":        "莫吉托",
	"old_fashioned": "古典鸡尾酒",
}

type drinkCount struct {
	DrinkName string
	Count     int
}

func DrinkPopularityChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()

	// 统计每种酒品的点单数量
	var results []drinkCount
	if err := db.Model(&OrderRecord{}).
		Select("drink_name, COUNT(*) as count").
		Group("drink_name").
		Order("count DESC").
		Scan(&results).Error; err != nil {
		logger.Errorf(ctx, "DrinkPopularityChart Query err: %v", err)
		return err
	}

	// 构建饼图数据
	var pieData []interface{}
	var totalOrders int

	for _, r := range results {
		cnName := drinkNameCN[r.DrinkName]
		if cnName == "" {
			cnName = r.DrinkName
		}
		pieData = append(pieData, map[string]interface{}{
			"name":  cnName,
			"value": r.Count,
		})
		totalOrders += r.Count
	}

	// 如果没有数据，使用示例数据
	if len(pieData) == 0 {
		pieData = []interface{}{
			map[string]interface{}{"name": "纯威士忌", "value": 28},
			map[string]interface{}{"name": "威士忌酸", "value": 22},
			map[string]interface{}{"name": "马天尼", "value": 15},
			map[string]interface{}{"name": "啤酒", "value": 35},
			map[string]interface{}{"name": "莫吉托", "value": 18},
			map[string]interface{}{"name": "古典鸡尾酒", "value": 10},
		}
		totalOrders = 128
	}

	c := &chart.PieChart{
		Title:  "酒单热力图",
		Series: []chart.ChartSeries{{Name: "点单数", Data: pieData}},
		Metadata: map[string]interface{}{
			"总点单数": totalOrders,
		},
	}

	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("drink_popularity.chart", DrinkPopularityChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "酒单热力图",
			Request:  &DrinkPopularityReq{},
			Response: &chart.PieChart{},
		},
		ChartType: app.ChartTypePie,
	})
}
