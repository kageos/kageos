package text

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/text",
	Name:        "文本工具",
	Desc:        "面向文本和中文语料的处理能力，支持分词、关键词提取等轻量文本分析任务。",
}
