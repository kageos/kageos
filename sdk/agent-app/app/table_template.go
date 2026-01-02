package app

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
)

type TableTemplate struct {
	BaseConfig
	AutoCrudTable          interface{} `json:"auto_crud_table"`
	OnTableAddRow          OnTableAddRow
	OnTableUpdateRow       OnTableUpdateRow
	OnTableDeleteRows      OnTableDeleteRows
	// OnTableCreateInBatches 是系统内置的回调，不需要用户实现
	// 系统会自动通过反射获取 AutoCrudTable 结构，批量插入数据库
	OnTableCreateInBatches func(ctx *Context, req *callback.OnTableCreateInBatchesReq) (*callback.OnTableCreateInBatchesResp, error) `json:"-"`
}

func (t *TableTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *TableTemplate) TemplateType() TemplateType {
	return TemplateTypeTable
}
