package inventory

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/inventory",
	Name:        "进销存管理系统",
	Desc:        "管理商品、供应商、采购入库和销售出库，实时跟踪库存变动，并提供销售和采购统计看板。",
}
