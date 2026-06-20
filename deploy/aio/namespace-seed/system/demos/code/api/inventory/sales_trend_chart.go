package inventory

import (
	"time"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
	"github.com/kageos/kageos/sdk/agent-app/types"
)

type SalesTrendReq struct {
	StartTime types.Time `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   types.Time `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func SalesTrendChart(ctx *app.Context, resp response.Response) error {
	var req SalesTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()

	startTime := req.StartTime.Time()
	endTime := req.EndTime.Time()
	if startTime.IsZero() {
		startTime = time.Now().AddDate(0, 0, -30)
	}
	if endTime.IsZero() {
		endTime = time.Now()
	}

	type DailyStats struct {
		Date       string  `json:"date"`
		Amount     float64 `json:"amount"`
		OrderCount int     `json:"order_count"`
	}

	var stats []DailyStats
	err := db.Model(&SalesOrder{}).
		Select("DATE(sales_time) as date, SUM(total_amount) as amount, COUNT(*) as order_count").
		Where("status = ? AND sales_time BETWEEN ? AND ?", "已出库", startTime, endTime).
		Group("DATE(sales_time)").
		Order("date ASC").
		Scan(&stats).Error
	if err != nil {
		return err
	}

	xAxis := make([]string, 0, len(stats))
	amountData := make([]interface{}, 0, len(stats))
	countData := make([]interface{}, 0, len(stats))
	for _, s := range stats {
		xAxis = append(xAxis, s.Date)
		amountData = append(amountData, s.Amount)
		countData = append(countData, s.OrderCount)
	}

	c := &chart.LineChart{
		Title: "销售趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "销售额", Data: amountData},
			{Name: "出库单数", Data: countData},
		},
	}
	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("sales_trend.chart", SalesTrendChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "销售趋势",
			Request:  &SalesTrendReq{},
			Response: &chart.LineChart{},
		},
	})
}
