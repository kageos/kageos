package model

import (
	"encoding/json"

	"github.com/kageos/kageos/pkg/gormx/models"
)

type OperateLog struct {
	models.Base
	TenantUser    string          `json:"tenant_user" gorm:"type:varchar(100);not null;index:idx_operate_log_scope;comment:workspace 所属用户"`
	CompanyCode   string          `json:"company_code" gorm:"type:varchar(64);index:idx_operate_log_company;comment:企业代码"`
	App           string          `json:"app" gorm:"type:varchar(100);not null;index:idx_operate_log_scope;comment:应用代码"`
	ActorUser     string          `json:"actor_user" gorm:"type:varchar(100);not null;index:idx_operate_log_actor;comment:实际操作者"`
	Action        string          `json:"action" gorm:"type:varchar(100);not null;index:idx_operate_log_action;comment:稳定事件枚举"`
	ResourceType  string          `json:"resource_type" gorm:"type:varchar(50);index;comment:资源类型"`
	ResourcePath  string          `json:"resource_path" gorm:"type:varchar(500);index:idx_operate_log_scope;comment:资源路径"`
	ResourceName  string          `json:"resource_name" gorm:"type:varchar(255);comment:资源名称"`
	TargetUser    string          `json:"target_user" gorm:"type:varchar(100);index;comment:被操作用户"`
	TargetID      string          `json:"target_id" gorm:"type:varchar(100);index;comment:被操作对象 ID"`
	Summary       string          `json:"summary" gorm:"type:varchar(1000);comment:人类可读摘要"`
	DetailsJSON   json.RawMessage `json:"details_json" gorm:"type:json;comment:扩展结构化详情"`
	OldValuesJSON json.RawMessage `json:"old_values_json" gorm:"type:json;comment:变更前结构化数据"`
	NewValuesJSON json.RawMessage `json:"new_values_json" gorm:"type:json;comment:变更后结构化数据"`
	Status        string          `json:"status" gorm:"type:varchar(30);index;comment:success/failed"`
	Source        string          `json:"source" gorm:"type:varchar(80);comment:来源 browser/agent/openapi"`
	IPAddress     string          `json:"ip_address" gorm:"type:varchar(50);comment:IP 地址"`
	UserAgent     string          `json:"user_agent" gorm:"type:varchar(500);comment:User Agent"`
	TraceID       string          `json:"trace_id" gorm:"type:varchar(100);index;comment:追踪 ID"`
}

func (OperateLog) TableName() string {
	return "operate_logs"
}
