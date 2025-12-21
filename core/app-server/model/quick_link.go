package model

import (
	"encoding/json"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// QuickLink 快链数据模型
// 用于保存表单状态、表格状态、图表筛选条件等，支持通过 URL 快速恢复
type QuickLink struct {
	models.Base
	CreatedBy      string          `json:"created_by" gorm:"column:created_by;type:varchar(255);not null;comment:创建者用户名"`
	Name           string          `json:"name" gorm:"column:name;type:varchar(255);not null;comment:快链名称"`
	FunctionRouter string          `json:"function_router" gorm:"column:function_router;type:varchar(500);not null;index;comment:函数路由"`
	FunctionMethod string          `json:"function_method" gorm:"column:function_method;type:varchar(10);not null;comment:函数HTTP方法"`
	TemplateType   string          `json:"template_type" gorm:"column:template_type;type:varchar(50);not null;comment:模板类型:form,table,chart"`
	RequestParams  json.RawMessage `json:"request_params" gorm:"type:json;comment:表单参数(完整的FieldValue结构)"`
	FieldMetadata  json.RawMessage `json:"field_metadata" gorm:"type:json;comment:字段元数据(editable,readonly,hint,highlight)"`
	Metadata       json.RawMessage `json:"metadata" gorm:"type:json;comment:其他元数据(table_state,chart_filters,response_params)"`
}

func (QuickLink) TableName() string {
	return "quick_link"
}

// GetRequestParams 获取请求参数（解析 JSON）
func (q *QuickLink) GetRequestParams() (map[string]interface{}, error) {
	if len(q.RequestParams) == 0 {
		return make(map[string]interface{}), nil
	}
	var params map[string]interface{}
	err := json.Unmarshal(q.RequestParams, &params)
	return params, err
}

// GetFieldMetadata 获取字段元数据（解析 JSON）
func (q *QuickLink) GetFieldMetadata() (map[string]interface{}, error) {
	if len(q.FieldMetadata) == 0 {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal(q.FieldMetadata, &metadata)
	return metadata, err
}

// GetMetadata 获取其他元数据（解析 JSON）
func (q *QuickLink) GetMetadata() (map[string]interface{}, error) {
	if len(q.Metadata) == 0 {
		return make(map[string]interface{}), nil
	}
	var metadata map[string]interface{}
	err := json.Unmarshal(q.Metadata, &metadata)
	return metadata, err
}

// SetRequestParams 设置请求参数（序列化为 JSON）
func (q *QuickLink) SetRequestParams(params map[string]interface{}) error {
	data, err := json.Marshal(params)
	if err != nil {
		return err
	}
	q.RequestParams = data
	return nil
}

// SetFieldMetadata 设置字段元数据（序列化为 JSON）
func (q *QuickLink) SetFieldMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	q.FieldMetadata = data
	return nil
}

// SetMetadata 设置其他元数据（序列化为 JSON）
func (q *QuickLink) SetMetadata(metadata map[string]interface{}) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	q.Metadata = data
	return nil
}

