package printdrop

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/printdrop",
	Name:        "PrintDrop 打印文件管理系统",
	Desc:        "打印店文件上传与订单管理，支持选择打印规格、状态追踪。",
}
