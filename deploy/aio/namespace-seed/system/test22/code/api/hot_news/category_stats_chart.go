package hot_news

import (
	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// CategoryStatsReq 分类分布统计查询请求
type CategoryStatsReq struct {
	// 发布状态筛选（可选，不传则统计全部）
	Status string `json:"status" form:"status" widget:"name:发布状态;type:select;options:全部,草稿,已发布,已下架;options_colors:909399,909399,67C23A,F56C6C"`
}

// CategoryStatsChart 分类分布统计
func CategoryStatsChart(ctx *app.Context, resp response.Response) error {
	var req CategoryStatsReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	queryDB := db.Model(&News{})

	// 按发布状态筛选
	if req.Status != "" && req.Status != "全部" {
		queryDB = queryDB.Where("status = ?", req.Status)
	}

	// 按分类统计数量
	type CategoryStat struct {
		Category string
		Count    int
	}

	var stats []CategoryStat
	err := queryDB.
		Select("category, COUNT(*) as count").
		Group("category").
		Order("count DESC").
		Scan(&stats).Error

	if err != nil {
		logger.Errorf(ctx, "[CategoryStatsChart] 查询失败, err: %v", err)
		return err
	}

	// 构造饼图数据
	pieData := make([]interface{}, 0)
	var totalCount int64
	for _, s := range stats {
		pieData = append(pieData, map[string]interface{}{
			"name":  s.Category,
			"value": s.Count,
		})
		totalCount += int64(s.Count)
	}

	c := &chart.PieChart{
		Title: "分类分布统计",
		Series: []chart.ChartSeries{
			{Name: "新闻数量", Data: pieData},
		},
		Metadata: map[string]interface{}{
			"新闻总数": totalCount,
			"分类数量": len(stats),
		},
	}

	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("category_stats.chart", CategoryStatsChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "分类分布统计",
			Request:  &CategoryStatsReq{},
			Response: &chart.PieChart{},
		},
		ChartType: app.ChartTypePie,
	})
}
