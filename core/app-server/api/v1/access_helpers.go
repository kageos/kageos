package v1

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/app-server/service"
	"github.com/kageos/kageos/pkg/access"
	"github.com/kageos/kageos/pkg/contextx"
)

func requireAccess(c *gin.Context, teamAccessService *service.TeamAccessService, resourcePath string, action access.Action) error {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return err
	}
	return teamAccessService.Check(contextx.ToContext(c), tenantUser, app, contextx.GetRequestUser(c), resourcePath, action)
}

// requireWorkspaceDataAccess applies explicit authorization first and then the
// authenticated open-collaboration fallback. Use it only for business-data
// surfaces (Form/Table/Chart and workspace content reads), never for workspace
// administration, source code, schedules, credentials, sharing, or audit logs.
func requireWorkspaceDataAccess(c *gin.Context, teamAccessService *service.TeamAccessService, resourcePath string, action access.Action) error {
	resourcePath = access.NormalizeResourcePath(resourcePath)
	tenantUser, app, err := access.ParseUserApp(resourcePath)
	if err != nil {
		return err
	}
	return teamAccessService.CheckWorkspaceData(contextx.ToContext(c), tenantUser, app, contextx.GetRequestUser(c), resourcePath, action)
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
