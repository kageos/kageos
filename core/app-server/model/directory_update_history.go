package model

import (
	"encoding/json"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// DirectoryUpdateHistory 目录更新历史记录表
// 一张表同时支持App视角和目录视角的查询
type DirectoryUpdateHistory struct {
	models.Base
	AppID         int64  `json:"app_id" gorm:"column:app_id;index:idx_app_version;index:idx_app_dir;comment:应用ID"`
	AppVersion    string `json:"app_version" gorm:"type:varchar(50);index:idx_app_version;comment:对应的应用版本号（如 v101）"`
	AppVersionNum int    `json:"app_version_num" gorm:"index:idx_app_version;comment:应用版本号（数字部分，如 101）"`
	
	FullCodePath  string `json:"full_code_path" gorm:"type:varchar(500);index:idx_app_dir;index:idx_dir_version;comment:目录完整路径（如 /luobei/app/hr/attendance）"`
	DirVersion    string `json:"dir_version" gorm:"type:varchar(50);index:idx_dir_version;comment:目录版本号（如 v1, v2）"`
	DirVersionNum int    `json:"dir_version_num" gorm:"index:idx_dir_version;comment:目录版本号（数字部分，如 1, 2）"`
	
	// 变更摘要（JSON格式存储，只记录关键信息）
	AddedAPIs   json.RawMessage `json:"added_apis" gorm:"type:json;column:added_apis;comment:新增的API摘要列表"`
	UpdatedAPIs json.RawMessage `json:"updated_apis" gorm:"type:json;column:updated_apis;comment:更新的API摘要列表"`
	DeletedAPIs json.RawMessage `json:"deleted_apis" gorm:"type:json;column:deleted_apis;comment:删除的API摘要列表"`
	
	// 统计信息
	AddedCount   int `json:"added_count" gorm:"column:added_count;comment:新增API数量"`
	UpdatedCount int `json:"updated_count" gorm:"column:updated_count;comment:更新API数量"`
	DeletedCount int `json:"deleted_count" gorm:"column:deleted_count;comment:删除API数量"`
	
	// 变更摘要（详情），可能是大模型返回的摘要信息，也可能是用户的变更需求
	Summary string `json:"summary" gorm:"type:text;column:summary;comment:变更摘要（详情）"`
	
	// 变更需求（用户在前端输入的）
	Requirement string `json:"requirement" gorm:"type:text;column:requirement;comment:变更需求（用户输入）"`
	
	// 变更描述（大模型输出的）
	ChangeDescription string `json:"change_description" gorm:"type:text;column:change_description;comment:变更描述（大模型输出）"`
	
	// 变更耗时（毫秒）
	Duration int64 `json:"duration" gorm:"column:duration;comment:变更耗时（毫秒）"`
	
	// Git 相关信息
	GitCommitHash string `json:"git_commit_hash" gorm:"type:varchar(40);column:git_commit_hash;comment:Git提交哈希（用于回滚）"`
	
	// 操作信息
	UpdatedBy string `json:"updated_by" gorm:"column:updated_by;comment:更新人"`
}

func (DirectoryUpdateHistory) TableName() string {
	return "directory_update_history"
}

// ApiSummary API摘要信息（用于历史记录）
type ApiSummary struct {
	Code         string `json:"code"`          // 函数代码
	Name         string `json:"name"`          // 函数名称
	Desc         string `json:"desc"`         // 函数描述
	Router       string `json:"router"`        // 路由路径
	Method       string `json:"method"`        // HTTP方法
	FullCodePath string `json:"full_code_path"` // 完整代码路径
	TemplateType string `json:"template_type"` // 模板类型（如 form、table、chart）
}

// GetAddedAPIs 解析新增的API列表
func (h *DirectoryUpdateHistory) GetAddedAPIs() ([]*ApiSummary, error) {
	var apis []*ApiSummary
	if len(h.AddedAPIs) == 0 {
		return apis, nil
	}
	err := json.Unmarshal(h.AddedAPIs, &apis)
	return apis, err
}

// GetUpdatedAPIs 解析更新的API列表
func (h *DirectoryUpdateHistory) GetUpdatedAPIs() ([]*ApiSummary, error) {
	var apis []*ApiSummary
	if len(h.UpdatedAPIs) == 0 {
		return apis, nil
	}
	err := json.Unmarshal(h.UpdatedAPIs, &apis)
	return apis, err
}

// GetDeletedAPIs 解析删除的API列表
func (h *DirectoryUpdateHistory) GetDeletedAPIs() ([]*ApiSummary, error) {
	var apis []*ApiSummary
	if len(h.DeletedAPIs) == 0 {
		return apis, nil
	}
	err := json.Unmarshal(h.DeletedAPIs, &apis)
	return apis, err
}

