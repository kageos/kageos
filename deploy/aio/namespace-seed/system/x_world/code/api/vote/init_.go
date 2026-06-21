package vote

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/vote",
	Name:        "投票系统",
	Desc:        "投票系统 - 管理投票主题、选项和投票记录，支持用户提交投票并查看结果统计。",
}
