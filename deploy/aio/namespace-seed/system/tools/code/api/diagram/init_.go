package diagram

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/diagram",
	Name:        "图示工具",
	Desc:        "面向流程图、架构图、关系图的图示生成能力，支持结构化 JSON 和 DOT 描述渲染为 PNG、SVG、PDF。",
}
