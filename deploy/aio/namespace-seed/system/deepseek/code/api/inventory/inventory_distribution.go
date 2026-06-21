package inventory

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// InventoryDistReq 库存分布查询请求
type InventoryDistReq struct {
	Status string `json:"status" form:"status" widget:"name:状态;type:select;options:正常,停用;options_colors:4CAF50,F56C6C"`
}

var InventoryDistTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "库存分布",
		Request:  &InventoryDistReq{},
		Response: &chart.PieChart{},
	},
	ChartType: app.ChartTypePie,
}

// InventoryDistChart 库存分布饼图
func InventoryDistChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req InventoryDistReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	queryDB := db.Model(&Product{})
	if req.Status != "" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}

	type CategoryStat struct {
		Category string
		Total    int64
	}
	var stats []CategoryStat
	if err := queryDB.Select("category, SUM(stock) as total").Group("category").Scan(&stats).Error; err != nil {
		return err
	}

	pieData := make([]interface{}, 0, len(stats))
	var grandTotal int64
	for _, s := range stats {
		pieData = append(pieData, map[string]interface{}{
			"name":  s.Category,
			"value": s.Total,
		})
		grandTotal += s.Total
	}

	c := &chart.PieChart{
		Title:  "库存分布",
		Series: []chart.ChartSeries{{Name: "库存总量", Data: pieData}},
		Metadata: map[string]interface{}{
			"总库存量": grandTotal,
			"分类数":  len(stats),
		},
	}
	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("inventory_distribution.chart", InventoryDistChart, InventoryDistTemplate)
}
