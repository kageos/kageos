package hot_news

import (
	"fmt"
	"time"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// NewsTrendReq 新闻发布趋势查询请求
type NewsTrendReq struct {
	StartTime string `json:"start_time" form:"start_time" widget:"name:开始时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
	EndTime   string `json:"end_time" form:"end_time" widget:"name:结束时间;type:datetime;format:YYYY-MM-DD HH:mm:ss"`
}

// NewsTrendChart 新闻发布趋势统计
func NewsTrendChart(ctx *app.Context, resp response.Response) error {
	var req NewsTrendReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()

	// 默认时间范围：最近7天
	startTime := time.Now().AddDate(0, 0, -7)
	endTime := time.Now()

	if req.StartTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", req.StartTime); err == nil {
			startTime = t
		}
	}
	if req.EndTime != "" {
		if t, err := time.Parse("2006-01-02 15:04:05", req.EndTime); err == nil {
			endTime = t
		}
	}

	// 按日期分组统计已发布新闻数量
	type DailyStat struct {
		Date  string
		Count int
	}

	var stats []DailyStat
	err := db.Model(&News{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("status = ?", "已发布").
		Where("created_at >= ?", startTime).
		Where("created_at <= ?", endTime).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&stats).Error

	if err != nil {
		logger.Errorf(ctx, "[NewsTrendChart] 查询失败, err: %v", err)
		return err
	}

	// 生成日期序列，填充缺失日期的0值
	dateLabels := make([]string, 0)
	seriesData := make([]interface{}, 0)
	statMap := make(map[string]int)
	for _, s := range stats {
		statMap[s.Date] = s.Count
	}

	for d := startTime; !d.After(endTime); d = d.AddDate(0, 0, 1) {
		dateStr := d.Format("2006-01-02")
		dateLabels = append(dateLabels, dateStr)
		if count, ok := statMap[dateStr]; ok {
			seriesData = append(seriesData, count)
		} else {
			seriesData = append(seriesData, 0)
		}
	}

	// 计算总发布数量
	var totalPublished int64
	db.Model(&News{}).Where("status = ?", "已发布").Count(&totalPublished)

	c := &chart.LineChart{
		Title:  "新闻发布趋势",
		XAxis:  dateLabels,
		Series: []chart.ChartSeries{{Name: "发布数量", Data: seriesData}},
		Metadata: map[string]interface{}{
			"总发布数量": totalPublished,
			"统计周期":  fmt.Sprintf("%s 至 %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")),
		},
	}

	return resp.Chart(c).Build()
}

func init() {
	packageContext.GET("news_trend.chart", NewsTrendChart, &app.ChartTemplate{
		BaseConfig: app.BaseConfig{
			Name:     "新闻发布趋势",
			Request:  &NewsTrendReq{},
			Response: &chart.LineChart{},
		},
		ChartType: app.ChartTypeLine,
	})
}
