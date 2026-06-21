package rfpfiller

import "github.com/kageos/kageos-sdk/agent-app/app"

var packageContext = &app.PackageContext{
	RouterGroup: "/rfpfiller",
	Name:        "安全问卷与 RFP 填写",
	Desc:        "上传或粘贴公司资料、产品文档和客户问卷，生成带置信度和引用来源的回答草稿。",
}
