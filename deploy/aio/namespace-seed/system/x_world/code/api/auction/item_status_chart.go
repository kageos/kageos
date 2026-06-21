package auction

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 拍品状态分布图 ================

// ItemStatusDistChart 拍品状态分布统计
func ItemStatusDistChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	// 查询所有拍品并计算状态
	type ItemStatus struct {
		ID           int
		SessionID    int
		CurrentPrice float64
		TopBidder    string
		Session      AuctionSession
	}

	var items []ItemStatus
	if err := db.Model(&AuctionItem{}).Preload("Session").Find(&items).Error; err != nil {
		return fmt.Errorf("查询拍品失败: %v", err)
	}

	// 按状态分组统计
	statusCount := map[string]int{
		"未开始": 0,
		"竞价中": 0,
		"已成交": 0,
		"已流拍": 0,
	}

	for _, item := range items {
		status := calculateItemStatus(item.Session, item.CurrentPrice, item.TopBidder)
		statusCount[status]++
	}

	// 转换为饼图数据
	pieData := make([]interface{}, 0)
	for status, count := range statusCount {
		if count > 0 {
			pieData = append(pieData, map[string]interface{}{
				"name":  status,
				"value": count,
			})
		}
	}

	// 如果没有数据，返回默认数据
	if len(pieData) == 0 {
		pieData = []interface{}{
			map[string]interface{}{"name": "无拍品", "value": 0},
		}
	}

	// 计算总数
	total := len(items)

	c := &chart.PieChart{
		Title: "拍品状态分布",
		Series: []chart.ChartSeries{{
			Name: "数量",
			Data: pieData,
		}},
		Metadata: map[string]interface{}{
			"拍品总数": total,
		},
	}

	return resp.Chart(c).Build()
}

// ItemStatusDistChartTemplate 拍品状态分布图表配置
var ItemStatusDistChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "拍品状态分布",
		Desc:     `按拍品状态查看数量占比`,
		Request:  nil,
		Response: &chart.PieChart{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("item_status_distribution.chart", ItemStatusDistChart, ItemStatusDistChartTemplate)
}
