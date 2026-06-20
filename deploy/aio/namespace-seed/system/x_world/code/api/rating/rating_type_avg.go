// rating_type_avg.go
// 各类型平均评分图表：柱状图展示各类型的平均评分

package rating

import (
	"fmt"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 图表请求结构 ================

// RatingTypeAvgReq 各类型平均评分请求
type RatingTypeAvgReq struct {
	ObjectType string `json:"object_type" form:"object_type" widget:"name:类型筛选;type:select;options:全部,电影,图书,音乐,餐厅,酒店,商品,服务,课程,景点,其他;options_colors:909399,E91E63,9C27B0,673AB7,3F51B5,2196F3,00BCD4,009688,4CAF50,8BC34A,FF9800"`
}

// ================ 图表处理函数 ================

// RatingTypeAvgChart 各类型平均评分柱状图
func RatingTypeAvgChart(ctx *app.Context, resp response.Response) error {
	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	var req RatingTypeAvgReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	type TypeAvg struct {
		ObjectType  string  `json:"object_type" gorm:"column:object_type"`
		AvgRating   float64 `json:"avg_rating" gorm:"column:avg_rating"`
		RatingCount int     `json:"rating_count" gorm:"column:rating_count"`
	}

	var results []TypeAvg
	queryDB := db.Model(&RatingObject{}).Select("object_type, AVG(average_rating) as avg_rating, SUM(rating_count) as rating_count").
		Group("object_type").Order("avg_rating DESC")

	if req.ObjectType != "" && req.ObjectType != "全部" {
		queryDB = queryDB.Where("object_type = ?", req.ObjectType)
	}

	if err := queryDB.Scan(&results).Error; err != nil {
		return fmt.Errorf("查询评分数据失败: %v", err)
	}

	// 固定分类顺序
	typeOrder := []string{"电影", "图书", "音乐", "餐厅", "酒店", "商品", "服务", "课程", "景点", "其他"}
	typeMap := make(map[string]TypeAvg)
	for _, r := range results {
		typeMap[r.ObjectType] = r
	}

	var xAxis []string
	var seriesData []interface{}
	var totalCount int

	for _, t := range typeOrder {
		if avg, ok := typeMap[t]; ok {
			xAxis = append(xAxis, t)
			seriesData = append(seriesData, avg.AvgRating)
			totalCount += avg.RatingCount
		}
	}

	// 如果筛选了单个类型，展示该类型详情
	metadata := map[string]interface{}{
		"总评价人次": totalCount,
	}
	if req.ObjectType != "" && req.ObjectType != "全部" {
		metadata["当前类型"] = req.ObjectType
	}

	c := &chart.BarChart{
		Title:    "各类型平均评分",
		XAxis:    xAxis,
		Series:   []chart.ChartSeries{{Name: "平均评分", Data: seriesData}},
		Metadata: metadata,
	}

	return resp.Chart(c).Build()
}

// RatingTypeAvgTemplate 各类型平均评分配置
var RatingTypeAvgTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "各类型平均评分",
		Desc:     `柱状图展示各评价类型的平均评分分布`,
		Tags:     []string{"评分系统", "评分统计"},
		Request:  &RatingTypeAvgReq{},
		Response: &chart.BarChart{},
	},
	ChartType: app.ChartTypeBar,
}

// ================ API 注册 ================

func init() {
	packageContext.GET("rating_type_avg.chart", RatingTypeAvgChart, RatingTypeAvgTemplate)
}
