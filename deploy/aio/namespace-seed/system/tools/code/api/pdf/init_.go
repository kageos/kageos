package pdf

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/pdf",
	Name:        "PDF 工具",
	Desc:        "面向 PDF 文件的处理能力，统一承载信息读取、压缩、拆分、抽页、合并、文本提取、图片提取、页面渲染、OCR 和页面总览图生成。",
}
