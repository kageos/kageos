package ocr

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/ocr",
	Name:        "OCR 工具",
	Desc:        "面向图片文字识别的 OCR 能力，支持中文、英文和中英混合图片识别。PDF OCR 请优先使用 /pdf/ocr.form。",
}
