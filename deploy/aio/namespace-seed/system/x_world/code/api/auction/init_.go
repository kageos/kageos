package auction

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/auction",
	Name:        "拍卖会系统",
	Desc:        "拍卖会系统 - 管理拍卖场次、拍品、竞价记录和成交统计",
}
