package dto

import (
	"time"

	"github.com/kageos/kageos/pkg/access"
)

type GrantRoleReq struct {
	ResourcePath string           `json:"resource_path" binding:"required"`
	Principal    access.Principal `json:"principal" binding:"required"`
	RoleCode     access.RoleCode  `json:"role_code" binding:"required"`
	ExpiresAt    *time.Time       `json:"expires_at,omitempty"`
}

type BatchGrantRolesReq struct {
	ResourcePaths []string           `json:"resource_paths" binding:"required"`
	Principals    []access.Principal `json:"principals" binding:"required"`
	RoleCodes     []access.RoleCode  `json:"role_codes" binding:"required"`
	ExpiresAt     *time.Time         `json:"expires_at,omitempty"`
}

type RevokeRoleReq struct {
	ResourcePath string           `json:"resource_path" binding:"required"`
	Principal    access.Principal `json:"principal" binding:"required"`
	RoleCode     access.RoleCode  `json:"role_code,omitempty"`
}

type PermissionAssignmentsResp struct {
	Assignments []access.RoleAssignmentView `json:"assignments"`
}

type MyPermissionsResp struct {
	ResourcePath  string               `json:"resource_path"`
	RoleCodes     []access.RoleCode    `json:"role_codes,omitempty"`
	Permissions   access.PermissionSet `json:"permissions"`
	InheritedFrom string               `json:"inherited_from,omitempty"`
	ExpiresAt     *time.Time           `json:"expires_at,omitempty"`
}
