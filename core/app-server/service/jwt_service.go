package service

import "github.com/ai-agent-os/ai-agent-os/pkg/auth"

// JWTService 对 pkg/auth.JWTService 的类型别名，保持 app-server 内部 API 兼容
type JWTService = auth.JWTService

// JWTClaims 对 pkg/auth.JWTClaims 的类型别名
type JWTClaims = auth.JWTClaims

// NewJWTService 创建 JWT 服务（委托给 pkg/auth）
func NewJWTService() *JWTService {
	return auth.NewJWTService()
}
