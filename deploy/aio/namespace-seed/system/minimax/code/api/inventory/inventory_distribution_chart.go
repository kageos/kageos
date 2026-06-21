package inventory

import (
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// 库存分布统计请求
type InventoryDistributionReq struct {
	Category string `json:"category" form:"category" widget:"name:商品分类;type:select;options:全部,饮料,零食,日用品,其他;options_colors:909399,67C23A,FF9800,409EFF,9E9E9E"`
}

// 库存分布统计
func InventoryDistributionChart(ctx *app.Context, resp response.Response) error {
	var req InventoryDistributionReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	queryDB := db.Model(&InventoryLedger{})
	if req.Category != "" && req.Category != "全部" {
		queryDB = queryDB.Where("category = ?", req.Category)
	}

	var items []InventoryLedger
	if err := queryDB.Order("stock_qty DESC").Find(&items).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[InventoryDistributionChart] 查询失败, err: %v", err)
		return err
	}

	// 构造数据
	productNames := make([]string, 0)
	stockData := make([]interface{}, 0)
	for _, item := range items {
		productNames = append(productNames, item.ProductName)
		stockData = append(stockData, item.StockQty)
	}

	c := &chart.BarChart{
		Title:  "库存分布统计",
		XAxis:  productNames,
		Series: []chart.ChartSeries{{Name: "库存数量", Data: stockData}},
	}
	return resp.Chart(c).Build()
}

var InventoryDistributionChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "库存分布统计",
		Request:  &InventoryDistributionReq{},
		Response: &chart.BarChart{},
	},
	ChartType: app.ChartTypeBar,
}

func init() {
	packageContext.GET("inventory_distribution_statistics.chart", InventoryDistributionChart, InventoryDistributionChartTemplate)
}
