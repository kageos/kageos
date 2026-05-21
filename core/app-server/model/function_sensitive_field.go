package model

import "github.com/kageos/kageos/pkg/gormx/models"

type FunctionSensitiveField struct {
	models.Base
	TenantUser   string `json:"tenant_user" gorm:"type:varchar(100);not null;index:idx_function_sensitive_field_scope;comment:workspace 所属用户"`
	App          string `json:"app" gorm:"type:varchar(100);not null;index:idx_function_sensitive_field_scope;comment:应用代码"`
	FullCodePath string `json:"full_code_path" gorm:"type:varchar(500);not null;index:idx_function_sensitive_field_scope;comment:函数 full code path"`
	FunctionID   int64  `json:"function_id" gorm:"index;comment:function.ID"`
	SchemaType   string `json:"schema_type" gorm:"type:varchar(50);not null;comment:form/table/chart"`
	Section      string `json:"section" gorm:"type:varchar(50);not null;index:idx_function_sensitive_field_section;comment:request/response/fields"`
	FieldPath    string `json:"field_path" gorm:"type:varchar(500);not null;index:idx_function_sensitive_field_section;comment:schema 字段路径"`
	FieldCode    string `json:"field_code" gorm:"type:varchar(255);comment:字段 code"`
	FieldName    string `json:"field_name" gorm:"type:varchar(255);comment:字段展示名"`
	Source       string `json:"source" gorm:"type:varchar(50);not null;comment:schema/password"`
}

func (FunctionSensitiveField) TableName() string {
	return "function_sensitive_fields"
}
