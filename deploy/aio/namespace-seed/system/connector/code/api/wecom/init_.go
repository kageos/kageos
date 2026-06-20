package wecom

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/wecom",
	Name:        "企业微信",
	Desc:        "企业微信自建应用配置、连接测试和应用消息发送。",
}
