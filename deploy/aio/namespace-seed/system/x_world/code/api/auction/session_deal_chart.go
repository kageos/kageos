package auction

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 场次成交分布图 ================

// SessionDealDistReq 场次成交分布请求
type SessionDealDistReq struct {
	StartDate string `json:"start_date" form:"start_date" widget:"name:成交开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndDate   string `json:"end_date" form:"end_date" widget:"name:成交结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// SessionDealDistChart 场次成交分布统计
func SessionDealDistChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req SessionDealDistReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	// 按场次分组统计成交额
	query := `
		SELECT
			s.name as session_name,
			COALESCE(SUM(d.price), 0) as total_amount
		FROM auction_session s
		LEFT JOIN auction_deal_record d ON s.id = d.session_id
		WHERE 1=1
	`
	args := []interface{}{}

	if req.StartDate != "" {
		query += " AND d.created_at >= ?"
		args = append(args, req.StartDate)
	}
	if req.EndDate != "" {
		query += " AND d.created_at <= ?"
		args = append(args, req.EndDate)
	}

	query += " GROUP BY s.id, s.name ORDER BY total_amount DESC"

	type SessionStat struct {
		SessionName string  `json:"session_name"`
		TotalAmount float64 `json:"total_amount"`
	}

	var stats []SessionStat
	if err := db.Raw(query, args...).Scan(&stats).Error; err != nil {
		return fmt.Errorf("查询场次成交记录失败: %v", err)
	}

	// 转换为图表数据
	xAxis := make([]string, 0)
	seriesData := make([]interface{}, 0)
	totalAmount := float64(0)

	for _, s := range stats {
		xAxis = append(xAxis, s.SessionName)
		seriesData = append(seriesData, s.TotalAmount)
		totalAmount += s.TotalAmount
	}

	// 如果没有数据，返回空图表
	if len(xAxis) == 0 {
		xAxis = []string{"暂无数据"}
		seriesData = []interface{}{0}
	}

	c := &chart.BarChart{
		Title:  "场次成交分布",
		XAxis:  xAxis,
		Series: []chart.ChartSeries{{Name: "成交额(元)", Data: seriesData}},
		Metadata: map[string]interface{}{
			"总成交额": fmt.Sprintf("%.2f 元", totalAmount),
		},
	}

	return resp.Chart(c).Build()
}

// SessionDealDistChartTemplate 场次成交分布图表配置
var SessionDealDistChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "场次成交分布",
		Desc:     `按场次查看成交额分布`,
		Request:  &SessionDealDistReq{},
		Response: &chart.BarChart{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("session_deal_distribution.chart", SessionDealDistChart, SessionDealDistChartTemplate)
}
