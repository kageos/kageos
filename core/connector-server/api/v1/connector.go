package v1

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/connector-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type ConnectorAPI struct {
	connectorService *service.ConnectorService
}

func NewConnectorAPI(connectorService *service.ConnectorService) *ConnectorAPI {
	return &ConnectorAPI{connectorService: connectorService}
}

func (a *ConnectorAPI) CreateConnection(c *gin.Context) {
	var req dto.CreateConnectorConnectionReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	info, err := a.connectorService.CreateConnection(contextx.ToContext(c), req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.CreateConnectorConnectionResp{Connection: *info})
}

func (a *ConnectorAPI) ListConnections(c *gin.Context) {
	items, err := a.connectorService.ListConnections(contextx.ToContext(c), c.Query("provider"))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.ListConnectorConnectionsResp{Connections: items})
}

func (a *ConnectorAPI) DeleteConnection(c *gin.Context) {
	if err := a.connectorService.DeleteConnection(contextx.ToContext(c), c.Param("connection_id")); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

func (a *ConnectorAPI) RevokeConnection(c *gin.Context) {
	if err := a.connectorService.RevokeConnection(contextx.ToContext(c), c.Param("connection_id")); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

func (a *ConnectorAPI) BindDirectory(c *gin.Context) {
	var req dto.BindConnectorDirectoryReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	info, err := a.connectorService.BindDirectory(contextx.ToContext(c), req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.BindConnectorDirectoryResp{Binding: *info})
}

func (a *ConnectorAPI) ListDirectoryBindings(c *gin.Context) {
	items, err := a.connectorService.ListDirectoryBindings(contextx.ToContext(c), c.Query("resource_path"), c.Query("provider"))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.ListConnectorDirectoryBindingsResp{Bindings: items})
}

func (a *ConnectorAPI) DeleteDirectoryBinding(c *gin.Context) {
	resourcePath := strings.TrimSpace(c.Query("resource_path"))
	provider := strings.TrimSpace(c.Query("provider"))
	if err := a.connectorService.DeleteDirectoryBinding(contextx.ToContext(c), resourcePath, provider); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

func (a *ConnectorAPI) ResolveDirectoryBinding(c *gin.Context) {
	resp, err := a.connectorService.ResolveDirectoryBindingWithScopes(
		contextx.ToContext(c),
		c.Query("resource_path"),
		c.Query("provider"),
		parseScopeQuery(c.Query("required_scopes")),
	)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

func (a *ConnectorAPI) Proxy(c *gin.Context) {
	var req dto.ConnectorProxyReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	resp, err := a.connectorService.Proxy(contextx.ToContext(c), req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

func (a *ConnectorAPI) StartOAuth(c *gin.Context) {
	var req dto.StartConnectorOAuthReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	resp, err := a.connectorService.StartOAuth(contextx.ToContext(c), req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

func (a *ConnectorAPI) OAuthCallback(c *gin.Context) {
	resp, redirectAfter, err := a.connectorService.CompleteOAuthCallback(
		contextx.ToContext(c),
		c.Query("code"),
		c.Query("state"),
		firstNonEmpty(c.Query("error"), c.Query("error_description")),
	)
	if err != nil {
		if target := service.OAuthCallbackRedirect(redirectAfter, "error", "", err.Error()); target != "" {
			c.Redirect(http.StatusFound, target)
			return
		}
		response.FailWithMessage(c, err.Error())
		return
	}
	if target := service.OAuthCallbackRedirect(redirectAfter, "success", resp.Connection.ConnectionID, ""); target != "" {
		c.Redirect(http.StatusFound, target)
		return
	}
	response.OkWithData(c, resp)
}

func (a *ConnectorAPI) RefreshOAuthToken(c *gin.Context) {
	resp, err := a.connectorService.RefreshOAuthToken(contextx.ToContext(c), c.Param("connection_id"))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, resp)
}

func (a *ConnectorAPI) ListOAuthProviders(c *gin.Context) {
	items, err := a.connectorService.ListOAuthProviders(contextx.ToContext(c))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.ListConnectorOAuthProvidersResp{Providers: items})
}

func (a *ConnectorAPI) GetOAuthProvider(c *gin.Context) {
	info, err := a.connectorService.GetOAuthProvider(contextx.ToContext(c), c.Param("provider"))
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.GetConnectorOAuthProviderResp{Provider: *info})
}

func (a *ConnectorAPI) UpsertOAuthProvider(c *gin.Context) {
	var req dto.UpsertConnectorOAuthProviderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	if req.Code == "" {
		req.Code = c.Param("provider")
	}
	info, err := a.connectorService.UpsertOAuthProvider(contextx.ToContext(c), req)
	if err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.OkWithData(c, &dto.UpsertConnectorOAuthProviderResp{Provider: *info})
}

func (a *ConnectorAPI) DeleteOAuthProvider(c *gin.Context) {
	if err := a.connectorService.DeleteOAuthProvider(contextx.ToContext(c), c.Param("provider")); err != nil {
		response.FailWithMessage(c, err.Error())
		return
	}
	response.Ok(c)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func parseScopeQuery(value string) []string {
	scopes := make([]string, 0)
	for _, part := range strings.Fields(strings.ReplaceAll(value, ",", " ")) {
		part = strings.TrimSpace(part)
		if part != "" {
			scopes = append(scopes, part)
		}
	}
	return scopes
}
