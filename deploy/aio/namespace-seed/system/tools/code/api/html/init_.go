package html

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/html",
	Name:        "html工具",
	Desc:        "静态 HTML 页面生成工具，适合一次性报告、展示页、数据看板、Markdown 文档、交互式表格、时间线、日历、Mermaid 图表等纯展示场景。生成结果为可直接访问的 HTML 文件，不适合需要实时查库或后台状态变更的应用。",
}
