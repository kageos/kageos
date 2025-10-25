package app

type FormTemplate struct {
	BaseConfig
}

func (t *FormTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *FormTemplate) TemplateType() TemplateType {
	return TemplateTypeForm
}
