package cert_manager

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/cloudflare/cert_manager",
	Name:        "Cloudflare 证书管家",
	Desc:        "通过 Cloudflare DNS-01 自动签发、归档、巡检和续期 Let's Encrypt 证书；只管理证书文件，不自动部署。",
}
