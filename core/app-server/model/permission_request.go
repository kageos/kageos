package model

import (
	"time"

	"github.com/kageos/kageos/pkg/gormx/models"
)

const (
	PermissionRequestStatusPending   = "pending"
	PermissionRequestStatusApproved  = "approved"
	PermissionRequestStatusRejected  = "rejected"
	PermissionRequestStatusCancelled = "cancelled"
)

// WorkspacePermissionRequest records the lifecycle of a user's request for
// access to one workspace resource. Effective access continues to come only
// from WorkspaceRoleAssignment.
type WorkspacePermissionRequest struct {
	models.Base
	TenantUser       string     `json:"tenant_user" gorm:"type:varchar(100);not null;index:idx_permission_request_workspace;comment:workspace 所属用户"`
	App              string     `json:"app" gorm:"type:varchar(100);not null;index:idx_permission_request_workspace;comment:应用代码"`
	Requester        string     `json:"requester" gorm:"type:varchar(100);not null;index:idx_permission_request_requester;comment:申请用户"`
	ResourcePath     string     `json:"resource_path" gorm:"type:varchar(500);not null;index:idx_permission_request_workspace;comment:申请资源路径"`
	RequestedRole    string     `json:"requested_role" gorm:"type:varchar(30);not null;comment:申请角色 viewer/member"`
	Reason           string     `json:"reason" gorm:"type:text;not null;comment:申请理由"`
	Status           string     `json:"status" gorm:"type:varchar(30);not null;default:'pending';index:idx_permission_request_status;comment:pending/approved/rejected/cancelled"`
	PendingKey       *string    `json:"-" gorm:"type:char(64);uniqueIndex:idx_permission_request_pending_key;comment:待审批请求幂等键"`
	ReviewedBy       string     `json:"reviewed_by,omitempty" gorm:"type:varchar(100);index;comment:实际审批人"`
	ReviewedAt       *time.Time `json:"reviewed_at,omitempty" gorm:"type:datetime;comment:审批时间"`
	ReviewComment    string     `json:"review_comment,omitempty" gorm:"type:text;comment:审批意见"`
	RequestedExpires *time.Time `json:"requested_expires_at,omitempty" gorm:"type:datetime;comment:申请的授权到期时间"`
}

func (WorkspacePermissionRequest) TableName() string {
	return "workspace_permission_requests"
}
