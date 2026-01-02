package app

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
)

const (
	CallbackTypeOnTableAddRow         = "OnTableAddRow"
	CallbackTypeOnTableUpdateRow      = "OnTableUpdateRow"
	CallbackTypeOnTableDeleteRows     = "OnTableDeleteRows"
	CallbackTypeOnTableCreateInBatches = "OnTableCreateInBatches" // 系统内置批量创建回调
	CallbackTypeOnPageLoad            = "OnPageLoad"
	CallbackTypeOnSelectFuzzy         = "OnSelectFuzzy"
)

type OnTableAddRow func(ctx *Context, req *callback.OnTableAddRowReq) (*callback.OnTableAddRowResp, error)

// OnTableDeleteRows 当返回前端的数据是table类型时候，前端会把数据渲染成表格，这时候表格数据会有删除的行为，实现这个函数用来删除数据
type OnTableDeleteRows func(ctx *Context, req *callback.OnTableDeleteRowsReq) (*callback.OnTableDeleteRowsResp, error)

// OnTableUpdateRow 当返回前端的数据是table类型时候，前端会把数据渲染成表格，这时候表格数据会有更新的行为，实现这个函数用来更新数据
type OnTableUpdateRow func(ctx *Context, req *callback.OnTableUpdateRowReq) (*callback.OnTableUpdateRowResp, error)

// OnApiCreate 假如某次更新app时候，新增了这个api，则会出发点这个回调
type OnApiCreate func(ctx *Context, req *callback.OnApiCreateReq) (*callback.OnApiCreateResp, error)

// 前端访问该函数时候触发该回调
type OnPageLoad func(ctx *Context, req *callback.OnPageLoadReq) (*callback.OnPageLoadResp, error)

// OnSelectFuzzy 只有select组件才有的，当在select输入框输入关键字时候，如果有这个回调的话，会触发这个回调
type OnSelectFuzzy func(ctx *Context, req *callback.OnSelectFuzzyReq) (*callback.OnSelectFuzzyResp, error)
