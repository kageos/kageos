package v1

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
	"github.com/kageos/kageos/pkg/openapitoken"
)

func (u *User) ListOpenAPITokens(c *gin.Context) {
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}
	tokens, err := openapitoken.List(username)
	if err != nil {
		response.Internal(c, "查询 OpenAPI Token 失败: "+err.Error())
		return
	}
	items := make([]dto.OpenAPITokenInfo, 0, len(tokens))
	for _, token := range tokens {
		items = append(items, convertOpenAPIToken(token))
	}
	response.OkWithData(c, &dto.ListOpenAPITokensResp{Tokens: items})
}

func (u *User) CreateOpenAPIToken(c *gin.Context) {
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}
	var req dto.CreateOpenAPITokenReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	var expiresAt *time.Time
	if strings.TrimSpace(req.ExpiresAt) != "" {
		parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(req.ExpiresAt))
		if err != nil {
			response.BadRequest(c, "expires_at 必须是 RFC3339 时间格式")
			return
		}
		expiresAt = &parsed
	}
	ctx := contextx.ToContext(c)
	currentUser, err := u.userService.GetUserByUsername(ctx, username)
	if err != nil {
		response.Error(c, err)
		return
	}
	companyName := ""
	companyLogoURL := ""
	if currentUser.CompanyCode != "" {
		if companies, err := u.userService.GetCompaniesByCodes(ctx, []string{currentUser.CompanyCode}); err == nil && len(companies) > 0 {
			companyName = companies[0].Name
			companyLogoURL = companies[0].LogoURL
		}
	}
	result, err := openapitoken.Create(openapitoken.CreateInput{
		OwnerUsername:      username,
		OwnerUserID:        currentUser.ID,
		OwnerEmail:         currentUser.Email,
		CompanyCode:        currentUser.CompanyCode,
		CompanyName:        companyName,
		CompanyLogoURL:     companyLogoURL,
		DepartmentFullPath: currentUser.DepartmentFullPath,
		LeaderUsername:     currentUser.LeaderUsername,
		Name:               req.Name,
		ExpiresAt:          expiresAt,
	})
	if err != nil {
		response.Internal(c, "创建 OpenAPI Token 失败: "+err.Error())
		return
	}
	response.OkWithData(c, &dto.CreateOpenAPITokenResp{
		Token:       convertOpenAPIToken(result.Token),
		SecretToken: result.Secret,
	})
}

func (u *User) RevokeOpenAPIToken(c *gin.Context) {
	username := contextx.GetRequestUser(c)
	if username == "" {
		response.NoAuth(c, "未提供用户信息")
		return
	}
	var req struct {
		ID int64 `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}
	if err := openapitoken.Revoke(username, req.ID); err != nil {
		response.Internal(c, "吊销 OpenAPI Token 失败: "+err.Error())
		return
	}
	response.Ok(c)
}

func convertOpenAPIToken(token openapitoken.OpenAPIToken) dto.OpenAPITokenInfo {
	return dto.OpenAPITokenInfo{
		ID:                token.ID,
		Name:              token.Name,
		TokenPrefix:       token.TokenPrefix,
		ExpiresAt:         formatOptionalTime(token.ExpiresAt),
		RevokedAt:         formatOptionalTime(token.RevokedAt),
		LastUsedAt:        formatOptionalTime(token.LastUsedAt),
		LastUsedIP:        token.LastUsedIP,
		LastUsedUserAgent: token.LastUsedUserAgent,
		CreatedAt:         time.Time(token.CreatedAt).Format(time.RFC3339),
	}
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}
