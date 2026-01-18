package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// Role 角色模型
// ⭐ 角色按资源类型分组，每个资源类型有自己独立的角色集合
type Role struct {
	models.Base
	Name         string `json:"name" gorm:"type:varchar(100);not null;comment:角色名称"`
	Code         string `json:"code" gorm:"type:varchar(50);not null;index:idx_resource_code;comment:角色编码"`
	ResourceType string `json:"resource_type" gorm:"type:varchar(20);not null;index:idx_resource_code;comment:资源类型：directory、table、form、chart、app（角色适用的资源类型）"`
	Description  string `json:"description" gorm:"type:varchar(500);comment:角色描述"`
	IsSystem     bool   `json:"is_system" gorm:"type:tinyint(1);not null;default:0;comment:是否系统预设角色"`
	IsDefault    bool   `json:"is_default" gorm:"type:tinyint(1);not null;default:0;comment:是否默认角色（用于权限申请时的默认推荐）"`
	CreatedBy    string `json:"created_by" gorm:"type:varchar(100);comment:创建者用户名"`

	// 关联字段
	Permissions []*RolePermission `json:"permissions,omitempty" gorm:"foreignKey:RoleID;references:ID"`
}

func (*Role) TableName() string {
	return "role"
}

// RolePermission 角色权限模型
type RolePermission struct {
	models.Base
	RoleID   int64 `json:"role_id" gorm:"not null;index;comment:角色ID"`
	ActionID int64 `json:"action_id" gorm:"not null;index;comment:权限点ID（外键关联到 action 表）"`

	// 关联字段
	Role        *Role   `json:"role,omitempty" gorm:"foreignKey:RoleID;references:ID"`
	ActionModel *Action `json:"action_model,omitempty" gorm:"foreignKey:ActionID;references:ID"`

	// ⭐ 计算字段（用于 JSON 序列化，从 ActionModel 获取）
	ResourceType string `json:"resource_type" gorm:"-"`
	Action       string `json:"action" gorm:"-"`
}

// AfterFind GORM 钩子：查询后自动填充计算字段
func (rp *RolePermission) AfterFind(tx *gorm.DB) error {
	if rp.ActionModel != nil {
		rp.ResourceType = rp.ActionModel.ResourceType
		rp.Action = rp.ActionModel.Code
	}
	return nil
}

func (*RolePermission) TableName() string {
	return "role_permission"
}

// RoleAssignment 角色分配模型（统一表，支持用户和组织架构）
type RoleAssignment struct {
	models.Base
	User         string       `json:"user" gorm:"type:varchar(100);not null;index:idx_user_app;index:idx_user_app_subject;comment:租户用户名"`
	App          string       `json:"app" gorm:"type:varchar(100);not null;index:idx_user_app;index:idx_user_app_subject;comment:应用代码"`
	SubjectType  string       `json:"subject_type" gorm:"type:varchar(20);not null;index:idx_subject_type_subject;index:idx_user_app_subject;comment:权限主体类型：user（用户）、department（组织架构）"`
	Subject      string       `json:"subject" gorm:"type:varchar(150);not null;index:idx_subject_type_subject;index:idx_user_app_subject;comment:权限主体：用户名或组织架构路径"`
	RoleID       int64        `json:"role_id" gorm:"not null;index;comment:角色ID"`
	ResourcePath string       `json:"resource_path" gorm:"type:varchar(150);not null;index;comment:资源路径（角色生效范围）"`
	StartTime    models.Time  `json:"start_time" gorm:"type:datetime;not null;default:CURRENT_TIMESTAMP;index;comment:生效开始时间"`
	EndTime      *models.Time `json:"end_time" gorm:"type:datetime;index;comment:生效结束时间"`
	CreatedBy    string       `json:"created_by" gorm:"type:varchar(100);comment:创建者用户名"`

	// 关联字段
	Role *Role `json:"role,omitempty" gorm:"foreignKey:RoleID;references:ID"`
}

func (*RoleAssignment) TableName() string {
	return "role_assignment"
}

// IsEffective 检查角色分配是否生效
func (r *RoleAssignment) IsEffective(now time.Time) bool {
	if now.Before(time.Time(r.StartTime)) {
		return false
	}
	if r.EndTime != nil && now.After(time.Time(*r.EndTime)) {
		return false
	}
	return true
}
