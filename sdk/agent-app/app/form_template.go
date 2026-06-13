package app

type FormTemplate struct {
	BaseConfig
	Schedules []FormSchedule `json:"schedules,omitempty"`
}

func (t *FormTemplate) GetBaseConfig() *BaseConfig {
	return &t.BaseConfig
}

func (t *FormTemplate) TemplateType() TemplateType {
	return TemplateTypeForm
}
