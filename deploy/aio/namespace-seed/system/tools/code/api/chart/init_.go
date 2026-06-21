package chart

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/chart",
	Name:        "图表工具",
	Desc:        "面向数据可视化的图表生成能力，支持折线、柱状、饼图、分布图和组合图等常见展示。",
}
