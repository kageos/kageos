package app

import "github.com/kageos/kageos/dto"

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
	Code               string              `json:"code"`
	Name               string              `json:"name"`
	Desc               string              `json:"desc"`
	Tags               []string            `json:"tags"`
	CreateTables       []interface{}       `json:"create_tables"`
	Connectors         []string            `json:"connectors,omitempty"`
	ConnectorEndpoints []ConnectorEndpoint `json:"connector_endpoints,omitempty"`
	OnApiCreate        OnApiCreate         `json:"on_api_create"`

	// 请求响应
	Request  interface{} `json:"request"`
	Response interface{} `json:"response"`

	OnSelectFuzzyMap map[string]OnSelectFuzzy `json:"-"`
}

type ConnectorEndpoint = dto.ConnectorEndpoint
