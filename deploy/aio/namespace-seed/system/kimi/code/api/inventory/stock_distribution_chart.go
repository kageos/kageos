package inventory

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 商品库存分布（Chart） ================

func StockDistributionChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()

	type stockStat struct {
		Name string `json:"name"`
		Qty  int    `json:"qty"`
	}
	var stats []stockStat
	err := db.Model(&Product{}).
		Select("name as name, stock_qty as qty").
		Where("stock_qty > 0").
		Order("stock_qty DESC").
		Scan(&stats).Error
	if err != nil {
		return fmt.Errorf("[系统错误]-[StockDistributionChart] 统计失败, err: %w", err)
	}

	var xAxis []string
	var qtyData []interface{}
	var totalQty int
	for _, s := range stats {
		xAxis = append(xAxis, s.Name)
		qtyData = append(qtyData, s.Qty)
		totalQty += s.Qty
	}

	c := &chart.BarChart{
		Title: "商品库存分布",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "库存数量", Data: qtyData},
		},
		Metadata: map[string]interface{}{
			"总库存": totalQty,
		},
	}
	return resp.Chart(c).Build()
}

var StockDistributionChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "商品库存分布",
		Request: &struct{}{},
	},
	ChartType: app.ChartTypeBar,
}

func init() {
	packageContext.GET("stock_distribution.chart", StockDistributionChart, StockDistributionChartTemplate)
}
