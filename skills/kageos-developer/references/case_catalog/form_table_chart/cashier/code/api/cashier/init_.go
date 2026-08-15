package cashier

import "github.com/kageos/kageos-sdk/agent-app/app"

var packageContext = &app.PackageContext{
	RouterGroup: "/cashier",
	Name:        "收银系统",
	Desc:        "商品管理、收银结账、支付流水和销售趋势统计。",
}
