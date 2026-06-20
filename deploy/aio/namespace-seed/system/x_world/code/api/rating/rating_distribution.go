// rating_distribution.go
// 评分分布图表：饼图展示各评分区间的分布

package rating

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 图表请求结构 ================

// RatingDistributionReq 评分分布请求
type RatingDistributionReq struct {
	ObjectType string `json:"object_type" form:"object_type" widget:"name:类型筛选;type:select;options:全部,电影,图书,音乐,餐厅,酒店,商品,服务,课程,景点,其他;options_colors:909399,E91E63,9C27B0,673AB7,3F51B5,2196F3,00BCD4,009688,4CAF50,8BC34A,FF9800"`
}

// ================ 图表处理函数 ================

// RatingDistributionChart 评分分布饼图
func RatingDistributionChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req RatingDistributionReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	type DistResult struct {
		RatingRange string `json:"rating_range" gorm:"column:rating_range"`
		Count       int    `json:"count" gorm:"column:count"`
	}

	var results []DistResult
	queryDB := db.Model(&RatingRecord{})

	if req.ObjectType != "" && req.ObjectType != "全部" {
		queryDB = queryDB.Joins("JOIN rating_object ON rating_record.object_id = rating_object.id").
			Where("rating_object.object_type = ?", req.ObjectType)
	}

	// 按评分区间分组统计
	if err := queryDB.Select(`
		CASE
			WHEN rating >= 4.5 THEN '4.5-5分（优秀）'
			WHEN rating >= 3.5 THEN '3.5-4.5分（良好）'
			WHEN rating >= 2.5 THEN '2.5-3.5分（一般）'
			WHEN rating >= 1.5 THEN '1.5-2.5分（较差）'
			ELSE '1-1.5分（很差）'
		END as rating_range,
		COUNT(*) as count
	`).Group("rating_range").Order("count DESC").Scan(&results).Error; err != nil {
		return fmt.Errorf("查询评分分布失败: %v", err)
	}

	// 固定顺序
	rangeOrder := []string{"4.5-5分（优秀）", "3.5-4.5分（良好）", "2.5-3.5分（一般）", "1.5-2.5分（较差）", "1-1.5分（很差）"}
	rangeMap := make(map[string]int)
	for _, r := range results {
		rangeMap[r.RatingRange] = r.Count
	}

	pieData := make([]interface{}, 0)
	var totalCount int
	for _, r := range rangeOrder {
		if count, ok := rangeMap[r]; ok {
			pieData = append(pieData, map[string]interface{}{
				"name":  r,
				"value": count,
			})
			totalCount += count
		}
	}

	metadata := map[string]interface{}{
		"总评价数": totalCount,
	}
	if req.ObjectType != "" && req.ObjectType != "全部" {
		metadata["当前类型"] = req.ObjectType
	}

	c := &chart.PieChart{
		Title:    "评分分布",
		Series:   []chart.ChartSeries{{Name: "评价数量", Data: pieData}},
		Metadata: metadata,
	}

	return resp.Chart(c).Build()
}

// RatingDistributionTemplate 评分分布配置
var RatingDistributionTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "评分分布",
		Desc:     `饼图展示各评分区间的评价数量分布`,
		Tags:     []string{"评分系统", "评分统计"},
		Request:  &RatingDistributionReq{},
		Response: &chart.PieChart{},
	},
	ChartType: app.ChartTypePie,
}

// ================ API 注册 ================

func init() {
	packageContext.GET("rating_distribution.chart", RatingDistributionChart, RatingDistributionTemplate)
}
