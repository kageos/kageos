package sales_leads

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/sales_leads",
	Name:        "销售线索管理",
	Desc:        "销售线索管理系统 - 管理销售线索，跟进客户从初步接触到成交的全流程",
}
