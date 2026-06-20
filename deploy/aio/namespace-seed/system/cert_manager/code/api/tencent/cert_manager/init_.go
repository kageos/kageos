package cert_manager

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/tencent/cert_manager",
	Name:        "腾讯云证书管家",
	Desc:        "通过腾讯云 DNSPod DNS-01 自动签发、归档、巡检和续期 Let's Encrypt 证书；只管理证书文件，不自动部署。",
}
