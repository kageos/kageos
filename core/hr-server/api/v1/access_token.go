package v1

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kageos/kageos/core/hr-server/service"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/ginx/response"
)

// AccessToken exposes the HR-owned persistent session authority to the gateway.
type AccessToken struct {
	authService *service.AuthService
}

func NewAccessToken(authService *service.AuthService) *AccessToken {
	return &AccessToken{authService: authService}
}

// Validate is called by the API gateway on cache misses.
func (h *AccessToken) Validate(c *gin.Context) {
	rawToken := strings.TrimSpace(c.GetHeader(contextx.TokenHeader))
	if rawToken == "" {
		response.NoAuth(c, "未提供访问令牌")
		return
	}
	principal, err := h.authService.ValidateAccessToken(rawToken)
	if err != nil {
		response.NoAuth(c, "访问令牌无效、已过期或已吊销")
		return
	}
	response.OkWithData(c, principal)
}
