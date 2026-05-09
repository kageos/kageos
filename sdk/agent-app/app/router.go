package app

import (
	"strings"

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
