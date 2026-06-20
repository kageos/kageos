package database

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/database_0_1_0/database",
	Name:        "数据库工具",
	Desc:        "面向轻量结构化数据的数据库能力，当前支持 SQLite 结构查看、CSV 入库和只读 SQL 查询。",
}
