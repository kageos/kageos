package table

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/table",
	Name:        "表格工具",
	Desc:        "面向表格数据的处理能力，统一承载 CSV、TSV、Excel 的读取、转换、清洗、合并、查询和生成。",
}
