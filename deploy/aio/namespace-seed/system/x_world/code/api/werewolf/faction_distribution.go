// faction_distribution.go
// 存活阵营分布图表

package werewolf

import (
	"fmt"

	"github.com/kageos/kageos-sdk/pkg/logger"
	"github.com/kageos/kageos-sdk/agent-app/app"
	"github.com/kageos/kageos-sdk/agent-app/chart"
	"github.com/kageos/kageos-sdk/agent-app/response"
)

// ================ 请求结构 ================

// FactionDistributionReq 存活阵营分布请求
type FactionDistributionReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input"`
}

// ================ 存活阵营分布图表 ================

// FactionDistributionChart 存活阵营分布图表
func FactionDistributionChart(ctx *app.Context, resp response.Response) error {
	var req FactionDistributionReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	// 查询所有该房间的玩家
	var players []Player
	if err := db.Where("room_no = ?", req.RoomNo).Find(&players).Error; err != nil {
		logger.Errorf(ctx, "[系统错误]-[FactionDistributionChart] 查询玩家失败, req: %+v, err: %v", req, err)
		return fmt.Errorf("查询玩家失败: %w", err)
	}

	logger.Infof(ctx, "[调试]-[FactionDistributionChart] 查询到 %d 个玩家, req: %+v", len(players), req)

	// 手动分组统计
	wolfCount := 0
	goodCount := 0
	for _, p := range players {
		if p.Role == "狼人" {
			wolfCount++
		} else {
			goodCount++
		}
	}

	pieData := make([]interface{}, 0)
	totalCount := wolfCount + goodCount

	if wolfCount > 0 {
		pieData = append(pieData, map[string]interface{}{
			"name":  "狼人",
			"value": wolfCount,
		})
	}
	if goodCount > 0 {
		pieData = append(pieData, map[string]interface{}{
			"name":  "好人",
			"value": goodCount,
		})
	}

	if len(pieData) == 0 {
		pieData = append(pieData, map[string]interface{}{
			"name":  "暂无存活玩家",
			"value": 0,
		})
	}

	c := &chart.PieChart{
		Title: "存活阵营分布",
		Series: []chart.ChartSeries{
			{Name: "阵营人数", Data: pieData},
		},
		Metadata: map[string]interface{}{
			"存活总人数": totalCount,
		},
	}

	return resp.Chart(c).Build()
}

// FactionDistributionChartTemplate 存活阵营分布图表配置
var FactionDistributionChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "存活阵营分布",
		Desc:     `展示当前存活玩家中各阵营的占比`,
		Tags:     []string{"狼人杀", "图表统计"},
		Request:  &FactionDistributionReq{},
		Response: &chart.PieChart{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("faction_distribution.chart", FactionDistributionChart, FactionDistributionChartTemplate)
}
