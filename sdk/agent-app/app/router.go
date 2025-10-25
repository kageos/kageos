package app

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"
	"strings"
)

type HandleFunc func(ctx *Context, resp response.Response) error
type routerInfo struct {
	HandleFunc HandleFunc
	Router     string
	Method     string

	Template Templater
}

// 取路由的最后一段当作code
func (a *routerInfo) getCode() string {
	trim := strings.Trim(a.Router, "/")
	split := strings.Split(trim, "/")
	return split[len(split)-1]
}
