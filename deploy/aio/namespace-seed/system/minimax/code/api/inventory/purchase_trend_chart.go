package inventory

import (
	"fmt"
	"time"

	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// 采购趋势统计请求
type PurchaseTrendReq struct {
	StartDate string `json:"start_date" form:"start_date" widget:"name:开始日期;type:datetime"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:结束日期;type:datetime"`
}

// 采购趋势统计
func PurchaseTrendChart(ctx *app.Context, resp response.Response) error {
	var req PurchaseTrendReq
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

	// 按日期分组统计采购金额
	type DateStat struct {
		Date   string
		Amount float64
	}
	var stats []DateStat

	// SQLite 日期格式化
	dateExpr := "strftime('%Y-%m-%d', inbound_time)"
	err := db.Model(&PurchaseInbound{}).
		Select(fmt.Sprintf("%s as date, SUM(amount) as amount", dateExpr)).
		Where("inbound_time >= ? AND inbound_time <= ?", startDate, endDate).
		Group("date").
		Order("date ASC").
		Scan(&stats).Error
	if err != nil {
		logger.Errorf(ctx, "[系统错误]-[PurchaseTrendChart] 统计失败, err: %v", err)
		return err
	}

	// 构造日期列表和数据列表
	dateLabels := make([]string, 0)
	amountData := make([]interface{}, 0)

	// 补全缺失日期
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dateLabels = append(dateLabels, dateStr)
		found := false
		for _, s := range stats {
			if s.Date == dateStr {
				amountData = append(amountData, s.Amount)
				found = true
				break
			}
		}
		if !found {
			amountData = append(amountData, 0)
		}
	}

	c := &chart.LineChart{
		Title:  "采购趋势统计",
		XAxis:  dateLabels,
		Series: []chart.ChartSeries{{Name: "采购金额", Data: amountData}},
	}
	return resp.Chart(c).Build()
}

var PurchaseTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "采购趋势统计",
		Request:  &PurchaseTrendReq{},
		Response: &chart.LineChart{},
	},
	ChartType: app.ChartTypeLine,
}

func init() {
	packageContext.GET("purchase_trend_statistics.chart", PurchaseTrendChart, PurchaseTrendChartTemplate)
}
