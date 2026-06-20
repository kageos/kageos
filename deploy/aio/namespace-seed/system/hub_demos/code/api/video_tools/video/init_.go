package video

import (
	"github.com/kageos/kageos/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/video_tools/video",
	Name:        "视频工具",
	Desc:        "面向视频文件的处理能力，支持信息读取、转码、压缩、裁剪、拼接、抽帧、缩略图总览和 GIF 生成。",
}
