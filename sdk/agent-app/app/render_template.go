package app

type TemplateType string

const (
	TemplateTypeForm  TemplateType = "form"
	TemplateTypeTable TemplateType = "table"
	TemplateTypeChart TemplateType = "chart"
)

type Templater interface {
	GetBaseConfig() *BaseConfig
	TemplateType() TemplateType
}

type BaseConfig struct {
	// 名称配置
	Code         string        `json:"code"`
	Name         string        `json:"name"`
	Desc         string        `json:"desc"`
	Tags         []string      `json:"tags"`
	CreateTables []interface{} `json:"create_tables"`
	OnApiCreate  OnApiCreate   `json:"on_api_create"`

	// 请求响应
	Request  interface{} `json:"request"`
	Response interface{} `json:"response"`

	OnSelectFuzzyMap map[string]OnSelectFuzzy `json:"-"`
}
