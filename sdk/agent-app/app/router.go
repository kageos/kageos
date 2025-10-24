package app

import "github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response"

type HandleFunc func(ctx *Context, resp response.Response) error
type routerInfo struct {
	HandleFunc HandleFunc
	Router     string
	Method     string

	Template Templater
}
