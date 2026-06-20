package wechat_articles

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/weixin/wechat_articles",
	Name:        "微信文章",
	Desc:        "主要是可以搜微信公众号文章和读取微信微信公众号内容",
}
