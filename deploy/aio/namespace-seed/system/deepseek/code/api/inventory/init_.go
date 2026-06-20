package inventory

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/inventory",
	Name:        "进销存管理系统",
	Desc:        "管理商品、供应商、客户资料，支持采购入库与销售出库，自动更新库存并记录流水，提供采购、销售、库存统计图表。",
}
