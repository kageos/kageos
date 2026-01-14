package dto

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// ApplyPermissionReq 权限申请请求（角色申请）
type ApplyPermissionReq struct {
	ResourcePath string       `json:"resource_path" binding:"required"` // 资源路径（full-code-path）
	RoleID       int64        `json:"role_id" binding:"required"`       // 角色ID（必填）
	SubjectType  string       `json:"subject_type"`                     // 权限主体类型：user（用户）或 department（部门），可选，默认为 user
	Subject      string       `json:"subject"`                          // 权限主体：用户名或组织架构路径，可选，默认为当前用户
	Reason       string       `json:"reason"`                           // 申请理由（可选）
	EndTime      *models.Time `json:"end_time"`                         // 权限结束时间（nil 表示永久）
}

// ApplyPermissionResp 权限申请响应
type ApplyPermissionResp struct {
	ID      string `json:"id"`      // 申请ID（暂时返回空字符串，后续可以扩展为申请记录ID）
	Status  string `json:"status"`  // 申请状态（approved：已批准，pending：待审核）
	Message string `json:"message"` // 响应消息
}

// GetWorkspacePermissionsReq 获取工作空间权限请求
// ⭐ 支持传递用户和组织架构参数，使方法可复用（既可以获取当前用户权限，也可以获取其他用户权限）
type GetWorkspacePermissionsReq struct {
	User               string `json:"user" form:"user"`                                                     // 工作空间所属用户（必填）
	App                string `json:"app" form:"app"`                                                       // 工作空间应用代码（必填）
	Username           string `json:"username,omitempty" form:"username,omitempty"`                         // 用户名（可选，如果不提供则从 context 获取当前用户）
	DepartmentFullPath string `json:"department_full_path,omitempty" form:"department_full_path,omitempty"` // 组织架构路径（可选，如果不提供则从 context 获取）
}

// PermissionRecord 权限记录
type PermissionRecord struct {
	ID       int64  `json:"id"`       // 权限记录ID
	User     string `json:"user"`     // 用户名
	Resource string `json:"resource"` // 资源路径
	Action   string `json:"action"`   // 操作类型
	AppID    int64  `json:"app_id"`   // 应用ID
}

// GetWorkspacePermissionsResp 获取工作空间权限响应
// ⭐ 直接返回原始权限记录，让前端处理
type GetWorkspacePermissionsResp struct {
	Records []PermissionRecord `json:"records"` // 原始权限记录
}

// ============================================
// 权限申请和审批相关 DTO（企业版功能）
// ============================================

// CreatePermissionRequestReq 创建权限申请请求（API 层使用，角色申请）
type CreatePermissionRequestReq struct {
	AppID        int64        `json:"app_id" binding:"required"`        // 工作空间ID
	ResourcePath string       `json:"resource_path" binding:"required"` // 资源路径（full-code-path）
	RoleID       int64        `json:"role_id" binding:"required"`      // 角色ID（必填）
	SubjectType  string       `json:"subject_type" binding:"required"`  // 权限主体类型：user 或 department
	Subject      string       `json:"subject" binding:"required"`       // 权限主体：用户名或组织架构路径
	StartTime    models.Time  `json:"start_time"`                       // 权限开始时间（可选，默认为当前时间）
	EndTime      *models.Time `json:"end_time"`                         // 权限结束时间（nil 表示永久）
	Reason       string       `json:"reason"`                           // 申请原因（可选）
}

// InternalCreatePermissionRequestReq 内部创建权限申请请求（企业版内部使用，角色申请）
type InternalCreatePermissionRequestReq struct {
	User              string       `json:"user"`               // 租户用户名
	App               string       `json:"app"`                // 应用代码
	AppID             int64        `json:"app_id"`             // 工作空间ID（从 resourcePath 解析得到）
	ApplicantUsername string       `json:"applicant_username"` // 申请人用户名
	SubjectType       string       `json:"subject_type"`       // 权限主体类型
	Subject           string       `json:"subject"`            // 权限主体
	ResourcePath      string       `json:"resource_path"`      // 资源路径
	RoleID            int64        `json:"role_id"`            // 角色ID（必填）
	StartTime         models.Time  `json:"start_time"`         // 权限开始时间
	EndTime           *models.Time `json:"end_time"`           // 权限结束时间（nil 表示永久）
	Reason            string       `json:"reason"`             // 申请原因
}

// CreatePermissionRequestResp 创建权限申请响应
type CreatePermissionRequestResp struct {
	RequestID int64  `json:"request_id"` // 申请记录ID
	Status    string `json:"status"`     // 申请状态（pending：待审批）
	Message   string `json:"message"`    // 响应消息
}

// ApprovePermissionRequestReq 审批通过权限申请请求
type ApprovePermissionRequestReq struct {
	RequestID int64 `json:"request_id" binding:"required"` // 申请记录ID
}

// RejectPermissionRequestReq 审批拒绝权限申请请求
type RejectPermissionRequestReq struct {
	RequestID int64  `json:"request_id" binding:"required"` // 申请记录ID
	Reason    string `json:"reason"`                        // 拒绝原因（可选）
}

// GetPermissionRequestsReq 获取权限申请列表请求
type GetPermissionRequestsReq struct {
	AppID        int64  `json:"app_id" form:"app_id"`               // 工作空间ID（可选）
	Status       string `json:"status" form:"status"`               // 申请状态（pending、approved、rejected，可选）
	Applicant    string `json:"applicant" form:"applicant"`         // 申请人用户名（可选）
	ResourcePath string `json:"resource_path" form:"resource_path"` // 资源路径（可选）
	Page         int    `json:"page" form:"page"`                   // 页码（可选，默认1）
	PageSize     int    `json:"page_size" form:"page_size"`         // 每页数量（可选，默认20）
}

// PermissionRequestInfo 权限申请信息（角色申请）
type PermissionRequestInfo struct {
	ID                int64        `json:"id"`                 // 申请记录ID
	AppID             int64        `json:"app_id"`             // 工作空间ID
	ApplicantUsername string       `json:"applicant_username"` // 申请人用户名
	SubjectType       string       `json:"subject_type"`       // 权限主体类型
	Subject           string       `json:"subject"`            // 权限主体
	ResourcePath      string       `json:"resource_path"`      // 资源路径
	ResourceName      string       `json:"resource_name"`      // 资源名称（中文，从 service_tree 获取）
	RoleID            int64        `json:"role_id"`            // 角色ID
	RoleName          string       `json:"role_name"`          // 角色名称（从角色服务获取）
	StartTime         models.Time  `json:"start_time"`         // 权限开始时间
	EndTime           *models.Time `json:"end_time"`           // 权限结束时间（nil 表示永久）
	Reason            string       `json:"reason"`             // 申请原因
	Status            string       `json:"status"`             // 申请状态
	ApprovedAt        *models.Time `json:"approved_at"`        // 审批时间（可选）
	ApprovedBy        string       `json:"approved_by"`        // 审批人用户名（可选）
	RejectedAt        *models.Time `json:"rejected_at"`        // 拒绝时间（可选）
	RejectedBy        string       `json:"rejected_by"`        // 拒绝人用户名（可选）
	RejectReason      string       `json:"reject_reason"`      // 拒绝原因（可选）
	CreatedAt         models.Time  `json:"created_at"`         // 申请时间
	Approvers         []string     `json:"approvers"`          // 审批人列表（节点管理员，从 service_tree 获取）
}

// GetPermissionRequestsResp 获取权限申请列表响应
type GetPermissionRequestsResp struct {
	Total    int64                   `json:"total"`     // 总记录数
	Page     int                     `json:"page"`      // 当前页码
	PageSize int                     `json:"page_size"` // 每页数量
	Records  []PermissionRequestInfo `json:"records"`   // 申请记录列表
}
