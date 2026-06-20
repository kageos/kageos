package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

type AuthLoginProvider struct {
	providerService *service.AuthLoginProviderService
}

func NewAuthLoginProvider(providerService *service.AuthLoginProviderService) *AuthLoginProvider {
	return &AuthLoginProvider{providerService: providerService}
}

func (a *AuthLoginProvider) PublicMethods(c *gin.Context) {
	methods, err := a.providerService.ListLoginMethods()
	if err != nil {
		response.FailWithMessage(c, "获取登录方式失败: "+err.Error())
		return
	}
	response.OkWithData(c, &dto.ListLoginMethodsResp{Methods: methods})
}

func (a *AuthLoginProvider) List(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	providers, err := a.providerService.ListProviders()
	if err != nil {
		response.FailWithMessage(c, "获取登录方式配置失败: "+err.Error())
		return
	}
	response.OkWithData(c, &dto.ListAuthLoginProvidersResp{Providers: providers})
}

func (a *AuthLoginProvider) UpdateConfig(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	var req dto.UpdateAuthLoginProviderConfigReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	provider, err := a.providerService.UpdateConfig(c.Param("code"), req.Config, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, "保存登录方式配置失败: "+err.Error())
		return
	}
	response.OkWithData(c, provider)
}

func (a *AuthLoginProvider) SetEnabled(c *gin.Context) {
	if !requireSystemUser(c) {
		return
	}
	var req dto.UpdateAuthLoginProviderEnabledReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.FailWithMessage(c, "请求参数错误: "+err.Error())
		return
	}
	provider, err := a.providerService.SetEnabled(c.Param("code"), req.Enabled, contextx.GetRequestUser(c))
	if err != nil {
		response.FailWithMessage(c, "更新登录方式状态失败: "+err.Error())
		return
	}
	response.OkWithData(c, provider)
}
