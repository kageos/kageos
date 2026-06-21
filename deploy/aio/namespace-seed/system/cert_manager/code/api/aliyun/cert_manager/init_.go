package cert_manager

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/aliyun/cert_manager",
	Name:        "阿里云证书管家",
	Desc:        "通过阿里云云解析 DNS-01 自动签发、归档、巡检和续期 Let's Encrypt 证书；只管理证书文件，不自动部署。",
}
