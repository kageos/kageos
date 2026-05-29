package dto

type OpenAPITokenInfo struct {
	ID                int64   `json:"id"`
	Name              string  `json:"name"`
	TokenPrefix       string  `json:"token_prefix"`
	ExpiresAt         *string `json:"expires_at,omitempty"`
	RevokedAt         *string `json:"revoked_at,omitempty"`
	LastUsedAt        *string `json:"last_used_at,omitempty"`
	LastUsedIP        string  `json:"last_used_ip,omitempty"`
	LastUsedUserAgent string  `json:"last_used_user_agent,omitempty"`
	CreatedAt         string  `json:"created_at"`
}

type CreateOpenAPITokenReq struct {
	Name      string `json:"name" binding:"required" example:"kageos-hub-production"`
	ExpiresAt string `json:"expires_at,omitempty" example:"2027-01-01T00:00:00Z"`
}

type CreateOpenAPITokenResp struct {
	Token       OpenAPITokenInfo `json:"token"`
	SecretToken string           `json:"secret_token"`
}

type ListOpenAPITokensResp struct {
	Tokens []OpenAPITokenInfo `json:"tokens"`
}
