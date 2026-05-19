package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type WorkspaceRoleAssignment struct {
	models.Base
	TenantUser   string     `json:"tenant_user" gorm:"type:varchar(100);not null;index:idx_team_access_scope;index:idx_team_access_user;comment:workspace 所属用户"`
	App          string     `json:"app" gorm:"type:varchar(100);not null;index:idx_team_access_scope;index:idx_team_access_user;comment:应用代码"`
	Username     string     `json:"username" gorm:"type:varchar(100);not null;index:idx_team_access_user;comment:被授权用户名"`
	ResourcePath string     `json:"resource_path" gorm:"type:varchar(500);not null;index:idx_team_access_scope;comment:授权资源路径 full_code_path"`
	RoleCode     string     `json:"role_code" gorm:"type:varchar(30);not null;index:idx_team_access_scope;comment:固定角色 owner/admin/member/viewer"`
	ExpiresAt    *time.Time `json:"expires_at" gorm:"type:datetime;index;comment:到期时间，空表示永久有效"`
}

func (WorkspaceRoleAssignment) TableName() string {
	return "workspace_role_assignments"
}
