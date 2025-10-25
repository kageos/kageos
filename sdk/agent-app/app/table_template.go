package app

type TableTemplate struct {
	BaseConfig
	AutoCrudTable     interface{}       `json:"auto_crud_table"`
	OnTableAddRow     OnTableAddRow     `json:"on_table_add_row"`
	OnTableUpdateRows OnTableUpdateRows `json:"on_table_update_rows"`
	OnTableDeleteRows OnTableDeleteRows `json:"on_table_delete_rows"`
}

func (t *TableTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *TableTemplate) TemplateType() TemplateType {
	return TemplateTypeTable
}
