package ticket

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/ticket",
	Name:        "工单管理",
	Desc:        "工单管理系统",
}
