package dto

import (
	"time"

	"github.com/kageos/kageos/pkg/access"
)

type AssignTeamRoleReq struct {
	ResourcePath string          `json:"resource_path" binding:"required"`
	Username     string          `json:"username" binding:"required"`
	RoleCode     access.RoleCode `json:"role_code" binding:"required"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
}

type BatchAssignTeamRoleReq struct {
	ResourcePaths []string          `json:"resource_paths" binding:"required"`
	Usernames     []string          `json:"usernames" binding:"required"`
	RoleCodes     []access.RoleCode `json:"role_codes" binding:"required"`
	ExpiresAt     *time.Time        `json:"expires_at,omitempty"`
}

type RemoveTeamRoleReq struct {
	ResourcePath string          `json:"resource_path" binding:"required"`
	Username     string          `json:"username" binding:"required"`
	RoleCode     access.RoleCode `json:"role_code,omitempty"`
}

type TeamMemberAccessResp struct {
	Members []access.MemberAccess `json:"members"`
}

type MyPermissionsResp struct {
	ResourcePath  string               `json:"resource_path"`
	RoleCodes     []access.RoleCode    `json:"role_codes,omitempty"`
	Permissions   access.PermissionSet `json:"permissions"`
	InheritedFrom string               `json:"inherited_from,omitempty"`
	ExpiresAt     *time.Time           `json:"expires_at,omitempty"`
}
