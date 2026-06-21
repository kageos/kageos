package document

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/doc/document",
	Name:        "文档工具",
	Desc:        "面向办公文档和资料抽取的处理能力，支持文档转 Markdown、文档对比、Markdown/HTML 转交付文档。",
}
