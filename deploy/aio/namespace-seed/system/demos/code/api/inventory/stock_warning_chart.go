package inventory

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

type StockWarningReq struct {
	Category string `json:"category" form:"category" widget:"name:商品分类;type:select;options:电子产品,办公用品,食品饮料,日用百货,其他"`
}

func StockWarningChart(ctx *app.Context, resp response.Response) error {
	var req StockWarningReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()
	queryDB := db.Model(&Product{}).Where("status = ? AND current_stock < safety_stock", "正常")
	if req.Category != "" {
		queryDB = queryDB.Where("category = ?", req.Category)
	}
	var products []Product
	if err := queryDB.Find(&products).Error; err != nil {
		return err
	}

	xAxis := make([]string, 0, len(products))
	currentStockData := make([]interface{}, 0, len(products))
	safetyStockData := make([]interface{}, 0, len(products))
	for _, p := range products {
		xAxis = append(xAxis, p.ProductName)
		currentStockData = append(currentStockData, p.CurrentStock)
		safetyStockData = append(safetyStockData, p.SafetyStock)
	}

	c := &chart.BarChart{
		Title: "库存预警",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "当前库存", Data: currentStockData},
			{Name: "安全库存", Data: safetyStockData},
		},
		Metadata: map[string]interface{}{
			"预警商品数": len(products),
		},
	}
	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("stock_warning.chart", StockWarningChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "库存预警",
			Request:  &StockWarningReq{},
			Response: &chart.BarChart{},
		},
	})
}
