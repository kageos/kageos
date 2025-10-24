package app

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
)

type OnTableAddRow func(ctx *Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error)

// OnTableDeleteRows 当返回前端的数据是table类型时候，前端会把数据渲染成表格，这时候表格数据会有删除的行为，实现这个函数用来删除数据
type OnTableDeleteRows func(ctx *Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error)

// OnTableUpdateRows 当返回前端的数据是table类型时候，前端会把数据渲染成表格，这时候表格数据会有更新的行为，实现这个函数用来更新数据
type OnTableUpdateRows func(ctx *Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error)
