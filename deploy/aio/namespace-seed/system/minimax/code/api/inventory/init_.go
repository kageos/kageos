package inventory

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/inventory",
	Name:        "进销存管理系统",
	Desc:        "进销存管理系统 - 管理商品、供应商、客户信息，处理采购入库和销售出库业务，统计采购销售趋势和库存分布。",
}
