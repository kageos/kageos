package inventory

import (
	"fmt"

	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 销售趋势（Chart） ================

type SalesTrendReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func SalesTrendChart(ctx *app.Context, resp response.Response) error {
	var req SalesTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	queryDB := db.Model(&SalesRecord{})
	if req.StartTime != "" {
		queryDB = queryDB.Where("outbound_time >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("outbound_time <= ?", req.EndTime)
	}

	dateExpr, groupExpr := app.DateTimeBucketExpr(db, "outbound_time", app.TimeBucketDay)
	type salesStat struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
		Qty    int     `json:"qty"`
	}
	var stats []salesStat
	err := queryDB.Select(fmt.Sprintf("%s as date, SUM(total_amount) as amount, SUM(qty) as qty", dateExpr)).
		Group(groupExpr).
		Order("date").
		Scan(&stats).Error
	if err != nil {
		return fmt.Errorf("[系统错误]-[SalesTrendChart] 统计失败, req: %+v, err: %w", req, err)
	}

	var xAxis []string
	var amountData []interface{}
	var qtyData []interface{}
	var totalAmount float64
	var totalQty int
	for _, s := range stats {
		xAxis = append(xAxis, s.Date)
		amountData = append(amountData, s.Amount)
		qtyData = append(qtyData, s.Qty)
		totalAmount += s.Amount
		totalQty += s.Qty
	}

	c := &chart.LineChart{
		Title: "销售趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "销售金额", Data: amountData},
			{Name: "出库数量", Data: qtyData},
		},
		Metadata: map[string]interface{}{
			"总销售金额": totalAmount,
			"总出库数量": totalQty,
		},
	}
	return resp.Chart(c).Build()
}

var SalesTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "销售趋势",
		Request: &SalesTrendReq{},
	},
	ChartType: app.ChartTypeLine,
}

func init() {
	packageContext.GET("sales_trend.chart", SalesTrendChart, SalesTrendChartTemplate)
}
