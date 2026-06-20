package image

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/img/image",
	Name:        "图片工具",
	Desc:        "面向图片文件的处理能力，统一承载格式转换、缩放裁剪、压缩优化、缩略图、对比图和拼版图生成。",
}
