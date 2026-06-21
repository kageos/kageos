package archive

import (
	"github.com/kageos/kageos-sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: "/archive",
	Name:        "压缩包工具",
	Desc:        "通用压缩包处理能力，支持查看和解出 ZIP、TAR、TAR.GZ、TGZ、TAR.XZ 等常见资料包。",
}
