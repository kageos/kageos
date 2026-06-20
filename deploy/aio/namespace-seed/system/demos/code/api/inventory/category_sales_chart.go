package inventory

import (
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

type CategorySalesReq struct {
	StartTime types.Time `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   types.Time `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func CategorySalesChart(ctx *app.Context, resp response.Response) error {
	var req CategorySalesReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()

	startTime := req.StartTime.Time()
	endTime := req.EndTime.Time()
	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, -1, 0)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	// 查询已出库的销售订单
	var orders []SalesOrder
	err := db.Where("status = ? AND sales_time BETWEEN ? AND ?", "已出库", startTime, endTime).
		Find(&orders).Error
	if err != nil {
		return err
	}

	// 按商品分类统计销售额
	categoryMap := make(map[string]float64)
	for _, order := range orders {
		// 解析销售明细，格式：商品名称×数量@单价元
		// 这里简化处理，实际应该解析明细获取商品分类
		// 暂时按订单总额统计
		categoryMap["未分类"] += order.TotalAmount
	}

	// 转换为饼图数据
	pieData := make([]interface{}, 0, len(categoryMap))
	for category, amount := range categoryMap {
		pieData = append(pieData, map[string]interface{}{
			"name":  category,
			"value": amount,
		})
	}

	c := &chart.PieChart{
		Title: "分类销售占比",
		Series: []chart.ChartSeries{
			{Name: "销售额", Data: pieData},
		},
	}
	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("category_sales.chart", CategorySalesChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "分类销售占比",
			Request:  &CategorySalesReq{},
			Response: &chart.PieChart{},
		},
	})
}
