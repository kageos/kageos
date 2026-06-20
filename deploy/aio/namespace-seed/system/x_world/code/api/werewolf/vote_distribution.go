// vote_distribution.go
// 投票分布图表

package werewolf

import (
	"fmt"
	"strings"

	"github.com/kageos/kageos/sdk/agent-app/app"
	"github.com/kageos/kageos/sdk/agent-app/chart"
	"github.com/kageos/kageos/sdk/agent-app/response"
)

// ================ 请求结构 ================

// VoteDistributionReq 投票分布请求
type VoteDistributionReq struct {
	RoomNo string `json:"room_no" widget:"name:房间号;type:input"`
	Round  int    `json:"round" widget:"name:轮次;type:integer"`
}

// ================ 投票分布图表 ================

// VoteDistributionChart 投票分布图表
func VoteDistributionChart(ctx *app.Context, resp response.Response) error {
	var req VoteDistributionReq
	if err := ctx.ShouldBind(&req); err != nil {
		return err
	}

	db := ctx.GetGormDB()
	if db == nil {
		return fmt.Errorf("数据库连接失败")
	}

	// 查询投票记录
	var records []GameRecord
	query := db.Model(&GameRecord{}).
		Where("room_no = ? AND phase = ? AND content LIKE ?", req.RoomNo, "投票", "投票给%")

	if req.Round > 0 {
		query = query.Where("round = ?", req.Round)
	}

	if err := query.Find(&records).Error; err != nil {
		return fmt.Errorf("查询投票记录失败: %w", err)
	}

	// 统计每个被投票者的票数
	voteCountMap := make(map[string]int)
	for _, r := range records {
		// 解析 "投票给XXX" 提取被投票者
		target := strings.TrimPrefix(r.Content, "投票给")
		if target != r.Content { // 说明前缀匹配成功
			voteCountMap[target]++
		}
	}

	// 构建图表数据
	var playerNames []string
	var voteCounts []interface{}

	for name, count := range voteCountMap {
		playerNames = append(playerNames, name)
		voteCounts = append(voteCounts, count)
	}

	// 按票数降序排序
	for i := 0; i < len(playerNames)-1; i++ {
		for j := i + 1; j < len(playerNames); j++ {
			if voteCounts[j].(int) > voteCounts[i].(int) {
				playerNames[i], playerNames[j] = playerNames[j], playerNames[i]
				voteCounts[i], voteCounts[j] = voteCounts[j], voteCounts[i]
			}
		}
	}

	if len(playerNames) == 0 {
		playerNames = []string{"暂无投票"}
		voteCounts = []interface{}{0}
	}

	c := &chart.BarChart{
		Title:  "投票分布",
		XAxis:  playerNames,
		Series: []chart.ChartSeries{{Name: "票数", Data: voteCounts}},
		Metadata: map[string]interface{}{
			"总票数": len(records),
		},
	}

	return resp.Chart(c).Build()
}

// VoteDistributionChartTemplate 投票分布图表配置
var VoteDistributionChartTemplate = &app.ChartTemplate{
	BaseConfig: app.BaseConfig{
		Name:     "投票分布",
		Desc:     `展示当前轮次各玩家的得票情况`,
		Tags:     []string{"狼人杀", "图表统计"},
		Request:  &VoteDistributionReq{},
		Response: &chart.BarChart{},
	},
}

// ================ API 注册 ================

func init() {
	packageContext.GET("vote_distribution.chart", VoteDistributionChart, VoteDistributionChartTemplate)
}
