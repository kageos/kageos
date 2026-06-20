package json

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/json",
	Name:        "JSON 工具",
	Desc:        "JSON 校验、格式化、压缩、jq 查询和 JSON 转 CSV 能力，适合工作台处理接口返回、配置文件和结构化数据。",
}
