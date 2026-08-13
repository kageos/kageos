package v1

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
)

func requireAccess(c *gin.Context, permissionService *service.PermissionService, resourcePath string, action access.Action) error {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return err
	}
	return permissionService.RequirePermission(contextx.ToContext(c), tenantUser, app, contextx.GetRequestUser(c), resourcePath, action)
}

// requireWorkstationAccess treats the AI workstation as a Member capability.
// It deliberately reuses the existing resource permissions instead of adding a
// second workstation-specific role: Viewer is denied, while Member, Admin, and
// Owner pass. Individual tools still enforce their own action permission.
func requireWorkstationAccess(c *gin.Context, permissionService *service.PermissionService, resourcePath string) error {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return err
	}
	result, err := permissionService.ResolvePermissions(
		contextx.ToContext(c), tenantUser, app, contextx.GetRequestUser(c), resourcePath,
	)
	if err != nil {
		return err
	}
	if !access.CoversRole(result.Permissions, access.RoleMember) {
		return fmt.Errorf("使用工作台至少需要 Member 权限: %s", resourcePath)
	}
	return nil
}

func normalizeFullCodePathParam(c *gin.Context) string {
	fullCodePath := c.Param("full-code-path")
	if fullCodePath == "" {
		return ""
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	return access.NormalizeResourcePath(fullCodePath)
}
