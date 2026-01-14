package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// ⭐ WorkspacePermission 已废弃，不再使用 workspace_permission 表，仅使用角色系统

// PermissionRequest 权限申请审批表（角色申请）
type PermissionRequest struct {
	models.Base
	AppID             int64        `json:"app_id" gorm:"column:app_id;not null;index:idx_app_status;comment:工作空间ID"`
	ApplicantUsername string       `json:"applicant_username" gorm:"column:applicant_username;type:varchar(100);not null;index:idx_applicant;comment:申请人用户名"`
	SubjectType       string       `json:"subject_type" gorm:"column:subject_type;type:varchar(20);not null;comment:权限主体类型"`
	Subject           string       `json:"subject" gorm:"column:subject;type:varchar(150);not null;comment:权限主体"`
	ResourcePath      string       `json:"resource_path" gorm:"column:resource_path;type:varchar(150);not null;index:idx_resource_status;comment:资源路径"`
	RoleID            int64        `json:"role_id" gorm:"column:role_id;not null;index;comment:角色ID"`
	StartTime         models.Time  `json:"start_time" gorm:"column:start_time;type:datetime;not null;comment:权限开始时间"`
	EndTime           *models.Time `json:"end_time" gorm:"column:end_time;type:datetime;comment:权限结束时间（NULL 表示永久）"`
	Reason            string       `json:"reason" gorm:"column:reason;type:text;comment:申请原因"`
	Status            string       `json:"status" gorm:"column:status;type:varchar(20);not null;default:'pending';index:idx_resource_status;index:idx_status;index:idx_app_status;comment:申请状态"`
	ApprovedAt        *models.Time `json:"approved_at" gorm:"column:approved_at;type:datetime;comment:审批时间"`
	ApprovedBy        string       `json:"approved_by" gorm:"column:approved_by;type:varchar(100);index:idx_approver;comment:审批人用户名"`
	RejectedAt        *models.Time `json:"rejected_at" gorm:"column:rejected_at;type:datetime;comment:拒绝时间"`
	RejectedBy        string       `json:"rejected_by" gorm:"column:rejected_by;type:varchar(100);comment:拒绝人用户名"`
	RejectReason      string       `json:"reject_reason" gorm:"column:reject_reason;type:text;comment:拒绝原因"`
	CancelledAt       *models.Time `json:"cancelled_at" gorm:"column:cancelled_at;type:datetime;comment:取消时间"`
	CancelledBy       string       `json:"cancelled_by" gorm:"column:cancelled_by;type:varchar(100);comment:取消人用户名"`
	RoleAssignmentID  *int64       `json:"role_assignment_id" gorm:"column:role_assignment_id;comment:关联的角色分配记录ID"`
}

func (*PermissionRequest) TableName() string {
	return "permission_request"
}

// 权限申请状态常量
const (
	PermissionRequestStatusPending   = "pending"   // 待审批
	PermissionRequestStatusApproved  = "approved"  // 已同意
	PermissionRequestStatusRejected  = "rejected"  // 已驳回
	PermissionRequestStatusCancelled = "cancelled" // 已取消
)

// PermissionGrantLog 授权记录表
type PermissionGrantLog struct {
	models.Base
	AppID           int64      `json:"app_id" gorm:"column:app_id;not null;index:idx_app_grantor;comment:工作空间ID"`
	GrantorUsername string     `json:"grantor_username" gorm:"column:grantor_username;type:varchar(100);not null;index:idx_grantor;index:idx_app_grantor;comment:授权人用户名"`
	GranteeType     string     `json:"grantee_type" gorm:"column:grantee_type;type:varchar(20);not null;index:idx_grantee;comment:被授权人类型"`
	Grantee         string     `json:"grantee" gorm:"column:grantee;type:varchar(150);not null;index:idx_grantee;comment:被授权人"`
	ResourcePath    string     `json:"resource_path" gorm:"column:resource_path;type:varchar(150);not null;index:idx_resource;comment:资源路径"`
	Action          string     `json:"action" gorm:"column:action;type:varchar(50);not null;comment:操作类型"`
	StartTime       time.Time  `json:"start_time" gorm:"column:start_time;type:datetime;not null;comment:权限开始时间"`
	EndTime         *time.Time `json:"end_time" gorm:"column:end_time;type:datetime;comment:权限结束时间（NULL 表示永久）"`
	GrantedAt       time.Time  `json:"granted_at" gorm:"column:granted_at;type:datetime;not null;default:CURRENT_TIMESTAMP;index:idx_grantor;index:idx_grantee;index:idx_resource;comment:授权时间"`
	RevokedAt       *time.Time `json:"revoked_at" gorm:"column:revoked_at;type:datetime;comment:撤销时间"`
	RevokedBy       string     `json:"revoked_by" gorm:"column:revoked_by;type:varchar(100);comment:撤销人用户名"`
	RevokeReason    string     `json:"revoke_reason" gorm:"column:revoke_reason;type:text;comment:撤销原因"`
	PermissionID    *int64     `json:"permission_id" gorm:"column:permission_id;comment:关联的权限记录ID"`
}

func (*PermissionGrantLog) TableName() string {
	return "permission_grant_log"
}

// ApprovalPolicy 审批策略配置表
type ApprovalPolicy struct {
	models.Base
	AppID            int64  `json:"app_id" gorm:"column:app_id;not null;index:idx_app_resource;comment:工作空间ID"`
	ResourcePath     string `json:"resource_path" gorm:"column:resource_path;type:varchar(150);not null;index:idx_resource_policy;index:idx_app_resource;comment:目录路径"`
	PolicyExpression string `json:"policy_expression" gorm:"column:policy_expression;type:varchar(150);not null;comment:审批策略表达式"`
	Description      string `json:"description" gorm:"column:description;type:varchar(150);comment:策略描述"`
	IsEnabled        bool   `json:"is_enabled" gorm:"column:is_enabled;type:tinyint(1);not null;default:1;index:idx_resource_policy;index:idx_app_resource;comment:是否启用"`
	UpdatedBy        string `json:"updated_by" gorm:"column:updated_by;type:varchar(100);comment:更新者用户名"`
}

func (*ApprovalPolicy) TableName() string {
	return "approval_policy"
}
