package file

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/file",
	Name:        "文件处理",
	Desc:        "通用文件能力，支持文件体检、哈希校验和远程文件下载；压缩包处理请优先使用 /archive。",
}
