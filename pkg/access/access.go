package access

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type Action string

const (
	ActionRead   Action = "read"
	ActionWrite  Action = "write"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionAdmin  Action = "admin"
	ActionOwner  Action = "owner"
)

type RoleCode string

const (
	RoleOwner  RoleCode = "owner"
	RoleAdmin  RoleCode = "admin"
	RoleMember RoleCode = "member"
	RoleViewer RoleCode = "viewer"
)

type PermissionSet map[Action]bool

type Assignment struct {
	TenantUser   string
	App          string
	Username     string
	ResourcePath string
	RoleCode     RoleCode
	ExpiresAt    *time.Time
	CreatedBy    string
}

type Result struct {
	ResourcePath  string        `json:"resource_path"`
	RoleCodes     []RoleCode    `json:"role_codes,omitempty"`
	Permissions   PermissionSet `json:"permissions"`
	InheritedFrom string        `json:"inherited_from,omitempty"`
	ExpiresAt     *time.Time    `json:"expires_at,omitempty"`
	Assignments   []Assignment  `json:"-"`
}

type Checker interface {
	Check(ctx context.Context, tenantUser, app, username, resourcePath string, action Action) error
	Can(ctx context.Context, tenantUser, app, username, resourcePath string, action Action) (bool, error)
	Resolve(ctx context.Context, tenantUser, app, username, resourcePath string) (*Result, error)
}

type Manager interface {
	Assign(ctx context.Context, req AssignRoleRequest) error
	BatchAssign(ctx context.Context, req BatchAssignRoleRequest) error
	Remove(ctx context.Context, req RemoveRoleRequest) error
	ListMembers(ctx context.Context, tenantUser, app, resourcePath string) ([]MemberAccess, error)
}

type AssignRoleRequest struct {
	TenantUser   string
	App          string
	Username     string
	ResourcePath string
	RoleCode     RoleCode
	ExpiresAt    *time.Time
	CreatedBy    string
}

type BatchAssignRoleRequest struct {
	TenantUser    string
	App           string
	Usernames     []string
	ResourcePaths []string
	RoleCodes     []RoleCode
	ExpiresAt     *time.Time
	CreatedBy     string
}

type RemoveRoleRequest struct {
	TenantUser   string
	App          string
	Username     string
	ResourcePath string
	RoleCode     RoleCode
	Actor        string
}

type MemberAccess struct {
	TenantUser     string        `json:"tenant_user"`
	App            string        `json:"app"`
	Username       string        `json:"username"`
	ResourcePath   string        `json:"resource_path"`
	RoleCode       RoleCode      `json:"role_code"`
	Permissions    PermissionSet `json:"permissions"`
	Source         string        `json:"source"`
	Direct         bool          `json:"direct"`
	InheritedFrom  string        `json:"inherited_from,omitempty"`
	TargetResource string        `json:"target_resource,omitempty"`
	ExpiresAt      *time.Time    `json:"expires_at,omitempty"`
	CreatedBy      string        `json:"created_by,omitempty"`
	CreatedAt      *time.Time    `json:"created_at,omitempty"`
	UpdatedAt      *time.Time    `json:"updated_at,omitempty"`
}

func NormalizeAction(action Action) Action {
	return Action(strings.ToLower(strings.TrimSpace(string(action))))
}

func NormalizeRoleCode(role RoleCode) RoleCode {
	return RoleCode(strings.ToLower(strings.TrimSpace(string(role))))
}

func NormalizeResourcePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return ""
	}
	return "/" + path
}

func IsSystemBuiltinPath(path string) bool {
	path = NormalizeResourcePath(path)
	return path == "/system" || strings.HasPrefix(path, "/system/")
}

func ParseUserApp(resourcePath string) (tenantUser, app string, err error) {
	parts := strings.Split(strings.Trim(NormalizeResourcePath(resourcePath), "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("resource_path 格式错误，无法解析 user 和 app: %s", resourcePath)
	}
	return parts[0], parts[1], nil
}

func AppRootPath(tenantUser, app string) string {
	tenantUser = strings.Trim(strings.TrimSpace(tenantUser), "/")
	app = strings.Trim(strings.TrimSpace(app), "/")
	if tenantUser == "" || app == "" {
		return ""
	}
	return "/" + tenantUser + "/" + app
}

func IsValidAction(action Action) bool {
	switch NormalizeAction(action) {
	case ActionRead, ActionWrite, ActionUpdate, ActionDelete, ActionAdmin, ActionOwner:
		return true
	default:
		return false
	}
}

func IsValidRoleCode(role RoleCode) bool {
	switch NormalizeRoleCode(role) {
	case RoleOwner, RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

func RolePermissions(role RoleCode) PermissionSet {
	switch NormalizeRoleCode(role) {
	case RoleOwner:
		return PermissionSet{
			ActionRead:   true,
			ActionWrite:  true,
			ActionUpdate: true,
			ActionDelete: true,
			ActionAdmin:  true,
			ActionOwner:  true,
		}
	case RoleAdmin:
		return PermissionSet{
			ActionRead:   true,
			ActionWrite:  true,
			ActionUpdate: true,
			ActionDelete: true,
			ActionAdmin:  true,
		}
	case RoleMember:
		return PermissionSet{
			ActionRead:   true,
			ActionWrite:  true,
			ActionUpdate: true,
		}
	case RoleViewer:
		return PermissionSet{
			ActionRead: true,
		}
	default:
		return EmptyPermissionSet()
	}
}

func EmptyPermissionSet() PermissionSet {
	return PermissionSet{
		ActionRead:   false,
		ActionWrite:  false,
		ActionUpdate: false,
		ActionDelete: false,
		ActionAdmin:  false,
		ActionOwner:  false,
	}
}

func MergePermissionSets(sets ...PermissionSet) PermissionSet {
	merged := EmptyPermissionSet()
	for _, set := range sets {
		for action, allowed := range set {
			if allowed {
				merged[NormalizeAction(action)] = true
			}
		}
	}
	return merged
}

func HasPermission(perms PermissionSet, action Action) bool {
	action = NormalizeAction(action)
	if action == "" {
		return false
	}
	if perms[ActionOwner] {
		return true
	}
	if action != ActionOwner && perms[ActionAdmin] {
		return true
	}
	return perms[action]
}

func ParentPaths(resourcePath string) []string {
	resourcePath = NormalizeResourcePath(resourcePath)
	if resourcePath == "" {
		return nil
	}

	parts := strings.Split(strings.Trim(resourcePath, "/"), "/")
	if len(parts) == 0 {
		return nil
	}

	paths := make([]string, 0, len(parts))
	for i := len(parts); i >= 1; i-- {
		paths = append(paths, "/"+strings.Join(parts[:i], "/"))
	}
	return paths
}

func PathApplies(assignedPath, resourcePath string) bool {
	assignedPath = NormalizeResourcePath(assignedPath)
	resourcePath = NormalizeResourcePath(resourcePath)
	if assignedPath == "" || resourcePath == "" {
		return false
	}
	return resourcePath == assignedPath || strings.HasPrefix(resourcePath, assignedPath+"/")
}

func Resolve(assignments []Assignment, resourcePath string, now time.Time) *Result {
	resourcePath = NormalizeResourcePath(resourcePath)
	result := &Result{
		ResourcePath: resourcePath,
		Permissions:  EmptyPermissionSet(),
	}
	if now.IsZero() {
		now = time.Now()
	}

	seenRoles := map[RoleCode]bool{}
	for _, assignment := range assignments {
		assignment.ResourcePath = NormalizeResourcePath(assignment.ResourcePath)
		assignment.RoleCode = NormalizeRoleCode(assignment.RoleCode)
		if !IsValidRoleCode(assignment.RoleCode) {
			continue
		}
		if assignment.ExpiresAt != nil && !assignment.ExpiresAt.After(now) {
			continue
		}
		if !PathApplies(assignment.ResourcePath, resourcePath) {
			continue
		}

		result.Permissions = MergePermissionSets(result.Permissions, RolePermissions(assignment.RoleCode))
		result.Assignments = append(result.Assignments, assignment)
		if !seenRoles[assignment.RoleCode] {
			result.RoleCodes = append(result.RoleCodes, assignment.RoleCode)
			seenRoles[assignment.RoleCode] = true
		}
		if result.InheritedFrom == "" || len(assignment.ResourcePath) > len(result.InheritedFrom) {
			result.InheritedFrom = assignment.ResourcePath
		}
		if assignment.ExpiresAt != nil {
			if result.ExpiresAt == nil || assignment.ExpiresAt.After(*result.ExpiresAt) {
				expiresAt := *assignment.ExpiresAt
				result.ExpiresAt = &expiresAt
			}
		}
	}
	if result.InheritedFrom == resourcePath {
		result.InheritedFrom = ""
	}
	return result
}
