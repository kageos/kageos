package inventory

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 采购趋势（Chart） ================

type PurchaseTrendReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

func PurchaseTrendChart(ctx *app.Context, resp response.Response) error {
	var req PurchaseTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	queryDB := db.Model(&PurchaseRecord{})
	if req.StartTime != "" {
		queryDB = queryDB.Where("inbound_time >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("inbound_time <= ?", req.EndTime)
	}

	dateExpr, groupExpr := app.DateTimeBucketExpr(db, "inbound_time", app.TimeBucketDay)
	type purchaseStat struct {
		Date   string  `json:"date"`
		Amount float64 `json:"amount"`
		Qty    int     `json:"qty"`
	}
	var stats []purchaseStat
	err := queryDB.Select(fmt.Sprintf("%s as date, SUM(total_amount) as amount, SUM(qty) as qty", dateExpr)).
		Group(groupExpr).
		Order("date").
		Scan(&stats).Error
	if err != nil {
		return fmt.Errorf("[系统错误]-[PurchaseTrendChart] 统计失败, req: %+v, err: %w", req, err)
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
		Title: "采购趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{Name: "采购金额", Data: amountData},
			{Name: "入库数量", Data: qtyData},
		},
		Metadata: map[string]interface{}{
			"总采购金额": totalAmount,
			"总入库数量": totalQty,
		},
	}
	return resp.Chart(c).Build()
}

var PurchaseTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:    "采购趋势",
		Request: &PurchaseTrendReq{},
	},
	ChartType: app.ChartTypeLine,
}

func init() {
	packageContext.GET("purchase_trend.chart", PurchaseTrendChart, PurchaseTrendChartTemplate)
}
