package inventory

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// 销售趋势统计请求
type SalesTrendReq struct {
	StartDate string `json:"start_date" form:"start_date" widget:"name:开始日期;type:datetime"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:结束日期;type:datetime"`
}

// 销售趋势统计
func SalesTrendChart(ctx *app.Context, resp response.Response) error {
	var req SalesTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 默认时间范围：最近30天
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()
	if req.StartDate != "" {
		if t, err := time.Parse("2006-01-02", req.StartDate[:10]); err == nil {
			startDate = t
		}
	}
	if req.EndDate != "" {
		if t, err := time.Parse("2006-01-02", req.EndDate[:10]); err == nil {
			endDate = t.Add(24*time.Hour - time.Second)
		}
	}

	// 按日期分组统计销售额和毛利
	type DateStat struct {
		Date        string
		SalesAmount float64
		Profit      float64
	}
	var stats []DateStat

	// SQLite 日期格式化
	dateExpr := "strftime('%Y-%m-%d', outbound_time)"
	err := db.Model(&SalesOutbound{}).
		Select(fmt.Sprintf("%s as date, SUM(sales_amount) as sales_amount, SUM(profit) as profit", dateExpr)).
		Where("outbound_time >= ? AND outbound_time <= ?", startDate, endDate).
		Group("date").
		Order("date ASC").
		Scan(&stats).Error
	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[SalesTrendChart] 统计失败, err: %v", err)
		return err
	}

	// 构造日期列表和销售数据、毛利数据
	dateLabels := make([]string, 0)
	salesData := make([]interface{}, 0)
	profitData := make([]interface{}, 0)

	// 补全缺失日期
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dateLabels = append(dateLabels, dateStr)
		found := false
		for _, s := range stats {
			if s.Date == dateStr {
				salesData = append(salesData, s.SalesAmount)
				profitData = append(profitData, s.Profit)
				found = true
				break
			}
		}
		if !found {
			salesData = append(salesData, 0)
			profitData = append(profitData, 0)
		}
	}

	c := &chart.LineChart{
		Title: "销售趋势统计",
		XAxis: dateLabels,
		Series: []chart.ChartSeries{
			{Name: "销售额", Data: salesData},
			{Name: "毛利", Data: profitData},
		},
	}
	return resp.Chart(c).Build()
}

var SalesTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "销售趋势统计",
		Request:  &SalesTrendReq{},
		Response: &chart.LineChart{},
	},
	ChartType: app.ChartTypeLine,
}

func init() {
	packageContext.GET("sales_trend_statistics.chart", SalesTrendChart, SalesTrendChartTemplate)
}
