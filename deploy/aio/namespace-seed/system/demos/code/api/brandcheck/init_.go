package brandcheck

import "github.com/kageos/kageos-sdk/agent-app/app"

var packageContext = &app.PackageContext{
	RouterGroup: "/brandcheck",
	Name:        "品牌可用性检查",
	Desc:        "面向创业项目命名的品牌占用验证工具，检查域名、GitHub、npm、PyPI、Docker Hub 等常见平台的可用性。",
}
