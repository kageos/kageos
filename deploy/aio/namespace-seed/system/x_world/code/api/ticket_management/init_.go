package ticket_management

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/ticket_management",
	Name:        "工单管理系统",
	Desc:        "工单管理系统 - 管理工单提交、处理、状态流转和列表筛选",
}
