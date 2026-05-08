package app

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback"
)

type TableTemplate struct {
	BaseConfig
	AutoCrudTable     interface{} `json:"auto_crud_table"`
	OnTableAddRow     OnTableAddRow
	OnTableUpdateRow  OnTableUpdateRow
	OnTableDeleteRows OnTableDeleteRows
	// OnTableCreateInBatches 是系统内置的回调，不需要用户实现
	// 系统会自动通过反射获取有效 AutoCrudTable 结构，批量插入数据库
	OnTableCreateInBatches func(ctx *Context, req *callback.OnTableCreateInBatchesReq) (*callback.OnTableCreateInBatchesResp, error) `json:"-"`
}

func (t *TableTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *TableTemplate) TemplateType() TemplateType {
	return TemplateTypeTable
}

// EffectiveAutoCrudTable 返回实际用于 Table schema 和系统内置批量导入的列表 Model。
// 显式 AutoCrudTable 优先；如果业务代码忘记配置，则降级使用 CreateTables 中第一张非空表。
func (t *TableTemplate) EffectiveAutoCrudTable() interface{} {
	if t == nil {
		return nil
	}
	if t.AutoCrudTable != nil {
		return t.AutoCrudTable
	}
	for _, table := range t.CreateTables {
		if table != nil {
			return table
		}
	}
	return nil
}
