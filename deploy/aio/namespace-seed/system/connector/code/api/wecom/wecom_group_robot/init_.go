package wecom_group_robot

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/wecom/wecom_group_robot",
	Name:        "企业微信群机器人",
	Desc:        "企业微信群机器人 Webhook 配置、素材上传和多类型消息推送。",
}
