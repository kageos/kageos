package app

type TableTemplate struct {
	BaseConfig
	AutoCrudTable     interface{} `json:"auto_crud_table"`
	OnTableAddRow     OnTableAddRow
	OnTableUpdateRow  OnTableUpdateRow
	OnTableDeleteRows OnTableDeleteRows
}

func (t *TableTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *TableTemplate) TemplateType() TemplateType {
	return TemplateTypeTable
}

// EffectiveAutoCrudTable 返回实际用于 Table schema 的列表 Model。
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
