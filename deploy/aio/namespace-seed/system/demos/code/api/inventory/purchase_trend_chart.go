package inventory

import (
	"time"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
	"github.com/kageos/kageos-sdk/agent-app/types"
)

type PurchaseTrendReq struct {
	StartTime types.Time `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   types.Time `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func PurchaseTrendChart(ctx *app.Context, resp response.Response) error {
	var req PurchaseTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}
	db := ctx.GetGormDB()

	// 默认查询最近 30 天
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
	err := db.Model(&PurchaseOrder{}).
		Select("DATE(purchase_time) as date, SUM(total_amount) as amount, COUNT(*) as order_count").
		Where("status = ? AND purchase_time BETWEEN ? AND ?", "已入库", startTime, endTime).
		Group("DATE(purchase_time)").
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
		Title: "采购趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "采购金额", Data: amountData},
			{Name: "入库单数", Data: countData},
		},
	}
	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("purchase_trend.chart", PurchaseTrendChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "采购趋势",
			Request:  &PurchaseTrendReq{},
			Response: &chart.LineChart{},
		},
	})
}
