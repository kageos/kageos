package auction

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 成交额趋势图 ================

// DealTrendReq 成交额趋势请求
type DealTrendReq struct {
	SessionID string `json:"session_id" form:"session_id" widget:"name:所属场次;type:input"`
}

// DealTrendChart 成交额趋势统计
func DealTrendChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req DealTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	// 按日期分组统计成交额
	query := `
		SELECT
			DATE(created_at) as deal_date,
			SUM(price) as total_amount
		FROM auction_deal_record
		WHERE 1=1
	`
	args := []interface{}{}

	if req.SessionID != "" {
		query += " AND session_id = ?"
		args = append(args, req.SessionID)
	}

	query += " GROUP BY DATE(created_at) ORDER BY deal_date ASC"

	type DealStat struct {
		DealDate    string  `json:"deal_date"`
		TotalAmount float64 `json:"total_amount"`
	}

	var stats []DealStat
	if err := db.Raw(query, args...).Scan(&stats).Error; err != nil {
		return fmt.Errorf("查询成交记录失败: %v", err)
	}

	// 转换为图表数据
	xAxis := make([]string, 0)
	seriesData := make([]interface{}, 0)
	totalAmount := float64(0)

	for _, s := range stats {
		xAxis = append(xAxis, s.DealDate)
		seriesData = append(seriesData, s.TotalAmount)
		totalAmount += s.TotalAmount
	}

	// 如果没有数据，返回空图表
	if len(xAxis) == 0 {
		xAxis = []string{"暂无数据"}
		seriesData = []interface{}{0}
	}

	c := &chart.LineChart{
		Title:  "成交额趋势",
		XAxis:  xAxis,
		Series: []chart.ChartSeries{{Name: "成交额(元)", Data: seriesData}},
		Metadata: map[string]interface{}{
			"总成交额": fmt.Sprintf("%.2f 元", totalAmount),
		},
	}

	return resp.Chart(c).Build()
}

// DealTrendChartTemplate 成交额趋势图表配置
var DealTrendChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "成交额趋势",
		Desc:     `按日期查看每日成交额变化趋势`,
		Request:  &DealTrendReq{},
		Response: &chart.LineChart{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("deal_trend_statistics.chart", DealTrendChart, DealTrendChartTemplate)
}
