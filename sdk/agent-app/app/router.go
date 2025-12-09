package app

import (
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/env"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
)

type HandleFunc func(ctx *Context, resp response.Response) error
type routerInfo struct {
	HandleFunc HandleFunc
	Options    *RegisterOptions
	Router     string
	Method     string
	Template   Templater
}

// BuildSourceCodeFilePath 构建源代码文件路径（用于 SourceCode 表的 Code 字段）
// 返回格式：{full_path}/{group_code}，例如：/luobei/testgroup/plugins/tools_cashier
// full_path 是 package 的完整路径（包含 user/app 前缀）
// group_code 是函数组代码
func (a *routerInfo) BuildSourceCodeFilePath() string {
	if a.Options == nil || a.Options.RouterGroup == nil {
		return ""
	}

	// 获取 user 和 app（从 env 中获取）
	// 注意：这里需要导入 env 包
	user := env.User
	app := env.App

	// PackagePath 是 package 路径，例如：plugins 或 crm/ticket（不包含前导斜杠）
	packagePath := strings.Trim(a.Options.PackagePath, "/")

	// GroupCode 是函数组代码，例如：tools_cashier 或 crm_ticket
	groupCode := a.Options.RouterGroup.GroupCode

	// 构建完整路径：/{user}/{app}/{package_path}/{group_code}
	// 例如：/luobei/testgroup/plugins/tools_cashier
	if packagePath == "" {
		return fmt.Sprintf("/%s/%s/%s", user, app, groupCode)
	}
	return fmt.Sprintf("/%s/%s/%s/%s", user, app, packagePath, groupCode)
}

// 取路由的最后一段当作code
func (a *routerInfo) getCode() string {
	trim := strings.Trim(a.Router, "/")
	split := strings.Split(trim, "/")
	return split[len(split)-1]
}

func (a *routerInfo) IsDefaultRouter() bool {
	t := strings.Trim(a.Router, "/")
	return strings.HasPrefix(t, "_")
}
