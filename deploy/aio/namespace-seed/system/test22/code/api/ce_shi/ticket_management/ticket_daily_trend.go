package ticket_management

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// TicketDailyTrendReq 每日工单趋势请求
type TicketDailyTrendReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// TicketDailyTrendChart 每日工单趋势图
func TicketDailyTrendChart(ctx *app.Context, resp response.Response) error {
	var req TicketDailyTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	queryDB := db.Model(&Ticket{})

	if req.StartTime != "" {
		queryDB = queryDB.Where("created_at >= ?", req.StartTime)
	}
	if req.EndTime != "" {
		queryDB = queryDB.Where("created_at <= ?", req.EndTime)
	}

	// 使用 SDK helper 获取日期分组表达式
	dateExpr, groupExpr := app.DateTimeBucketExpr(db, "created_at", app.TimeBucketDay)

	type DailyCount struct {
		Date  string
		Count int64
	}
	var stats []DailyCount

	if err := queryDB.
		Select(fmt.Sprintf("%s as date, COUNT(*) as count", dateExpr)).
		Group(groupExpr).
		Order("date ASC").
		Scan(&stats).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketDailyTrendChart] 按日期分组统计失败, err: %v", err)
		return fmt.Errorf("[系统错误] 按日期分组统计失败: %w", err)
	}

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketDailyTrendChart] 统计工单总数失败, err: %v", err)
	}

	xAxis := make([]string, 0, len(stats))
	seriesData := make([]interface{}, 0, len(stats))

	for _, s := range stats {
		xAxis = append(xAxis, s.Date)
		seriesData = append(seriesData, s.Count)
	}

	c := &chart.BarChart{
		Title: "每日工单趋势",
		XAxis: xAxis,
		Series: []chart.ChartSeries{
			{
				Name: "工单数量",
				Data: seriesData,
			},
		},
		Metadata: map[string]interface{}{
			"总工单数": total,
		},
	}

	return resp.Chart(c).Build()
}

// TicketDailyTrendTemplate 每日工单趋势配置
var TicketDailyTrendTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "每日工单趋势",
		Desc:     `按日期查看每日新增工单数量。`,
		Tags:     []string{"工单统计"},
		Request:  &TicketDailyTrendReq{},
		Response: &chart.BarChart{},
	},
}

func init() {
	packageContext.GET("ticket_daily_trend.chart", TicketDailyTrendChart, TicketDailyTrendTemplate)
}
