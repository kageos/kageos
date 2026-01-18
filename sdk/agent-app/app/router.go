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
// 返回格式：{full_path}，例如：/luobei/testgroup/tools/pdftools
// full_path 是 package 的完整路径（包含 user/app 前缀）
// 不再需要 group_code，直接使用 PackagePath
func (a *routerInfo) BuildSourceCodeFilePath() string {
	if a.Options == nil {
		return ""
	}

	// 获取 user 和 app（从 env 中获取）
	user := env.User
	app := env.App

	// PackagePath 是 package 路径，例如：tools/pdftools（不包含前导斜杠）
	packagePath := strings.Trim(a.Options.PackagePath, "/")

	// 构建完整路径：/{user}/{app}/{package_path}
	// 例如：/luobei/testgroup/tools/pdftools
	if packagePath == "" {
		return fmt.Sprintf("/%s/%s", user, app)
	}
	return fmt.Sprintf("/%s/%s/%s", user, app, packagePath)
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
