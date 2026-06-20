package runtime

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/user_1210227080_coding_0_1_0/runtime",
	Name:        "运行时工具",
	Desc:        "面向工作台编排和临时数据处理的脚本运行能力，当前支持 Python 和 Lua。复杂 CLI 编排建议优先使用 Python subprocess。",
}
