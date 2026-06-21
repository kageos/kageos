package message

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/message",
	Name:        "消息工具",
	Desc:        "面向工作台通知和交付的消息发送能力，支持向用户、部门或用户加部门发送 Markdown、HTML、纯文本消息。",
}
