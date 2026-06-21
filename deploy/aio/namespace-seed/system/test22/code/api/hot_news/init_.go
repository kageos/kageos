package hot_news

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/hot_news",
	Name:        "热点新闻管理系统",
	Desc:        "热点新闻管理系统 - 管理新闻的创建、编辑、分类、上下架状态，并统计发布趋势和分类分布。",
}
