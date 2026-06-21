package nps

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/nps",
	Name:        "NPS 净推荐值调研系统",
	Desc:        "NPS 净推荐值调研系统：收集客户 0-10 分推荐意愿评分，自动计算 NPS 分数并查看趋势分析。",
}
