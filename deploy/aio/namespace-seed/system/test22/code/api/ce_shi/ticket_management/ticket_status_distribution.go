package ticket_management

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// TicketStatusDistributionReq 工单状态分布请求
type TicketStatusDistributionReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:创建开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:创建结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// TicketStatusDistributionChart 工单状态分布图
func TicketStatusDistributionChart(ctx *app.Context, resp response.Response) error {
	var req TicketStatusDistributionReq
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

	var total int64
	if err := queryDB.Count(&total).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketStatusDistributionChart] 统计工单总数失败, err: %v", err)
		return fmt.Errorf("[系统错误] 统计工单总数失败: %w", err)
	}

	type StatusCount struct {
		Status string
		Count  int64
	}
	var stats []StatusCount
	if err := queryDB.Select("status, COUNT(*) as count").Group("status").Scan(&stats).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[TicketStatusDistributionChart] 按状态分组统计失败, err: %v", err)
		return fmt.Errorf("[系统错误] 按状态分组统计失败: %w", err)
	}

	statusMap := make(map[string]int64)
	for _, s := range stats {
		statusMap[s.Status] = s.Count
	}

	pieData := make([]interface{}, 0)
	// 按固定顺序输出
	for _, status := range []string{"待处理", "处理中", "已完成", "已关闭"} {
		count := statusMap[status]
		if count > 0 || len(stats) > 0 {
			pieData = append(pieData, map[string]interface{}{
				"name":  status,
				"value": count,
			})
		}
	}

	c := &chart.PieChart{
		Title: "工单状态分布",
		Series: []chart.ChartSeries{
			{
				Name: "工单状态",
				Data: pieData,
			},
		},
		Metadata: map[string]interface{}{
			"总工单数": total,
		},
	}

	return resp.Chart(c).Build()
}

// TicketStatusDistributionTemplate 工单状态分布配置
var TicketStatusDistributionTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "工单状态分布",
		Desc:     `按状态查看工单数量分布。`,
		Tags:     []string{"工单统计"},
		Request:  &TicketStatusDistributionReq{},
		Response: &chart.PieChart{},
	},
}

func init() {
	packageContext.GET("ticket_status_distribution.chart", TicketStatusDistributionChart, TicketStatusDistributionTemplate)
}
