package evaluation

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/evaluation",
	Name:        "评价系统",
	Desc:        "评价系统：创建评价活动并设置评价维度，用户对指定对象进行多维度打分评价，查看评价统计结果。",
}
