package cert_manager

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/cert_manager",
	Name:        "智能证书管理",
	Desc:        "管理域名证书资产、证书文件、到期巡检、续期任务和负责人提醒；不自动部署证书。",
}
