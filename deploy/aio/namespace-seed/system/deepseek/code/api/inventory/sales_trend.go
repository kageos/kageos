package inventory

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// SalesTrendReq 销售趋势查询请求
type SalesTrendReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

var SalesTrendTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售趋势",
		Request:  &SalesTrendReq{},
		Response: &chart.LineChart{},
	},
	ChartType: app.ChartTypeLine,
}

// SalesTrendChart 销售趋势折线图
func SalesTrendChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	var req SalesTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	dateExpr, groupExpr := app.DateTimeBucketExpr(db, "order_date", app.TimeBucketDay)

	queryDB := db.Model(&SalesOrder{}).Where("status != ?", "已取消")
	if req.StartTime != "" {
		queryDB = queryDB.Where("order_date >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("order_date <= ?", req.EndTime)
	}

	type DailyStat struct {
		Date  string
		Total float64
	}
	var stats []DailyStat
	if err := queryDB.Select(fmt.Sprintf("%s as date, SUM(total_amount) as total", dateExpr)).
		Group(groupExpr).Order("date ASC").Scan(&stats).Error; err != nil {
		return err
	}

	dateLabels := make([]string, 0, len(stats))
	amountData := make([]interface{}, 0, len(stats))
	var grandTotal float64
	for _, s := range stats {
		dateLabels = append(dateLabels, s.Date)
		amountData = append(amountData, s.Total)
		grandTotal += s.Total
	}

	c := &chart.LineChart{
		Title: "销售趋势",
		XAxis: dateLabels,
		Series: []chart.ChartSeries{
			{Name: "销售金额", Data: amountData},
		},
		Metadata: map[string]interface{}{
			"销售总金额": grandTotal,
			"统计天数":  len(stats),
		},
	}
	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("sales_trend.chart", SalesTrendChart, SalesTrendTemplate)
}
