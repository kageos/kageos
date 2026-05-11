package model

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

type WorkflowDefinition struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	Name            string `json:"name" gorm:"size:255;not null;index;comment:工作流名称"`
	Description     string `json:"description" gorm:"type:text;comment:工作流描述"`
	AppID           int64  `json:"app_id" gorm:"index;default:0;comment:所属应用ID"`
	FullCodePath    string `json:"full_code_path" gorm:"size:500;index;comment:服务树资源路径"`
	Status          string `json:"status" gorm:"size:20;not null;index;comment:draft/enabled/disabled"`
	LatestVersionID int64  `json:"latest_version_id" gorm:"index;default:0;comment:最新发布版本ID"`
	CreatedBy       string `json:"created_by" gorm:"size:255;index;comment:创建人"`
	UpdatedBy       string `json:"updated_by" gorm:"size:255;comment:最后更新人"`

	DraftDefinitionJSON json.RawMessage `json:"draft_definition_json" gorm:"column:draft_definition;type:json;comment:草稿定义JSON"`
}

func (WorkflowDefinition) TableName() string {
	return "workflow_definition"
}

type WorkflowDefinitionVersion struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

	WorkflowID       int64           `json:"workflow_id" gorm:"not null;index;uniqueIndex:idx_workflow_version;comment:工作流ID"`
	Version          int             `json:"version" gorm:"not null;uniqueIndex:idx_workflow_version;comment:版本号"`
	DefinitionJSON   json.RawMessage `json:"definition_json" gorm:"type:json;not null;comment:发布定义JSON"`
	InputSchemaJSON  json.RawMessage `json:"input_schema_json" gorm:"type:json;comment:输入schema"`
	OutputSchemaJSON json.RawMessage `json:"output_schema_json" gorm:"type:json;comment:输出schema"`
	Status           string          `json:"status" gorm:"size:20;not null;index;comment:draft/published/archived"`
	CreatedBy        string          `json:"created_by" gorm:"size:255;index;comment:发布人"`
}

func (WorkflowDefinitionVersion) TableName() string {
	return "workflow_definition_version"
}
