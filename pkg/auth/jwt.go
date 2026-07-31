package auth

import (
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/logger"
)

// JWTService JWT 服务（公共基础设施，所有服务共享）
type JWTService struct {
	config *appconfig.JWTConfig
}

// NewJWTService 创建 JWT 服务（使用全局配置）
func NewJWTService() *JWTService {
	globalConfig := appconfig.GetGlobalSharedConfig()
	return &JWTService{
		config: &globalConfig.JWT,
	}
}

// JWTClaims JWT 声明
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`

	CompanyCode    string `json:"company_code,omitempty"`
	CompanyName    string `json:"company_name,omitempty"`
	CompanyLogoURL string `json:"company_logo_url,omitempty"`

	DepartmentFullPath *string `json:"department_full_path,omitempty"`
	LeaderUsername     *string `json:"leader_username,omitempty"`

	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问令牌（不含组织架构信息）
func (s *JWTService) GenerateAccessToken(userID int64, username, email string) (string, error) {
	return s.GenerateAccessTokenWithContext(UserTokenContext{
		UserID:   userID,
		Username: username,
		Email:    email,
	})
}

// GenerateAccessTokenWithHR 生成访问令牌（含组织架构信息）
func (s *JWTService) GenerateAccessTokenWithHR(userID int64, username, email string, departmentFullPath string, leaderUsername string) (string, error) {
	return s.GenerateAccessTokenWithContext(UserTokenContext{
		UserID:             userID,
		Username:           username,
		Email:              email,
		DepartmentFullPath: departmentFullPath,
		LeaderUsername:     leaderUsername,
	})
}

type UserTokenContext struct {
	UserID             int64
	Username           string
	Email              string
	CompanyCode        string
	CompanyName        string
	CompanyLogoURL     string
	DepartmentFullPath string
	LeaderUsername     string
}

// AccessTokenPrincipal 是 HR 在确认会话有效后返回给网关的当前用户上下文。
type AccessTokenPrincipal struct {
	UserID             int64  `json:"user_id"`
	Username           string `json:"username"`
	Email              string `json:"email"`
	CompanyCode        string `json:"company_code"`
	CompanyName        string `json:"company_name"`
	CompanyLogoURL     string `json:"company_logo_url"`
	DepartmentFullPath string `json:"department_full_path"`
}

// GenerateAccessTokenWithContext 生成访问令牌，携带用户、企业和组织架构上下文。
func (s *JWTService) GenerateAccessTokenWithContext(userContext UserTokenContext) (string, error) {
	return s.generateTokenWithContext(userContext, "access", time.Now().Add(time.Duration(s.config.AccessTokenExpire)*time.Second))
}

// GenerateOpenAPITokenWithContext 生成用于 OpenAPI 调用的长期 JWT。
// expiresAt 为零值时不写 exp，表示不过期；吊销由 OpenAPI Token 记录控制。
func (s *JWTService) GenerateOpenAPITokenWithContext(userContext UserTokenContext, expiresAt *time.Time) (string, error) {
	username := strings.TrimSpace(userContext.Username)
	if username == "" {
		return "", fmt.Errorf("OpenAPI Token 用户名不能为空")
	}
	exp := time.Time{}
	if expiresAt != nil {
		exp = *expiresAt
	}
	return s.generateTokenWithSubject(userContext, "openapi:"+username, exp)
}

func (s *JWTService) generateTokenWithContext(userContext UserTokenContext, subjectPrefix string, expiresAt time.Time) (string, error) {
	return s.generateTokenWithSubject(userContext, fmt.Sprintf("%s_%d", subjectPrefix, userContext.UserID), expiresAt)
}

func (s *JWTService) generateTokenWithSubject(userContext UserTokenContext, subject string, expiresAt time.Time) (string, error) {
	now := time.Now()
	registeredClaims := jwt.RegisteredClaims{
		Issuer:    s.config.Issuer,
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
	}
	if !expiresAt.IsZero() {
		registeredClaims.ExpiresAt = jwt.NewNumericDate(expiresAt)
	}
	return s.generateTokenWithClaims(userContext, registeredClaims)
}

func (s *JWTService) generateTokenWithClaims(userContext UserTokenContext, registeredClaims jwt.RegisteredClaims) (string, error) {
	claims := JWTClaims{
		UserID:           userContext.UserID,
		Username:         userContext.Username,
		Email:            userContext.Email,
		CompanyCode:      userContext.CompanyCode,
		CompanyName:      userContext.CompanyName,
		CompanyLogoURL:   userContext.CompanyLogoURL,
		RegisteredClaims: registeredClaims,
	}

	if userContext.DepartmentFullPath != "" {
		claims.DepartmentFullPath = &userContext.DepartmentFullPath
	}
	if userContext.LeaderUsername != "" {
		claims.LeaderUsername = &userContext.LeaderUsername
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		logger.Errorf(nil, "[JWTService] Failed to generate access token: %v", err)
		return "", fmt.Errorf("生成访问令牌失败: %w", err)
	}

	logger.Infof(nil, "[JWTService] Access token generated for user: %s", userContext.Username)
	return tokenString, nil
}

// GenerateRefreshToken 生成刷新令牌
func (s *JWTService) GenerateRefreshToken(userID int64, username, email string) (string, error) {
	return s.GenerateRefreshTokenWithContext(UserTokenContext{
		UserID:   userID,
		Username: username,
		Email:    email,
	})
}

// GenerateRefreshTokenWithContext 生成刷新令牌，携带企业上下文，便于刷新时保持租户信息。
func (s *JWTService) GenerateRefreshTokenWithContext(userContext UserTokenContext) (string, error) {
	return s.GenerateRefreshTokenWithContextExpiresAt(
		userContext,
		time.Now().Add(time.Duration(s.config.RefreshTokenExpire)*time.Second),
	)
}

// GenerateRefreshTokenWithContextExpiresAt 生成在指定时间失效的刷新令牌。
func (s *JWTService) GenerateRefreshTokenWithContextExpiresAt(userContext UserTokenContext, expiresAt time.Time) (string, error) {
	now := time.Now()
	if expiresAt.IsZero() {
		expiresAt = now.Add(time.Duration(s.config.RefreshTokenExpire) * time.Second)
	}
	claims := JWTClaims{
		UserID:         userContext.UserID,
		Username:       userContext.Username,
		Email:          userContext.Email,
		CompanyCode:    userContext.CompanyCode,
		CompanyName:    userContext.CompanyName,
		CompanyLogoURL: userContext.CompanyLogoURL,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   fmt.Sprintf("refresh_%d", userContext.UserID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	if userContext.DepartmentFullPath != "" {
		claims.DepartmentFullPath = &userContext.DepartmentFullPath
	}
	if userContext.LeaderUsername != "" {
		claims.LeaderUsername = &userContext.LeaderUsername
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		logger.Errorf(nil, "[JWTService] Failed to generate refresh token: %v", err)
		return "", fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	logger.Infof(nil, "[JWTService] Refresh token generated for user: %s", userContext.Username)
	return tokenString, nil
}

// ValidateToken 验证令牌
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return []byte(s.config.Secret), nil
	})

	if err != nil {
		logger.Errorf(nil, "[JWTService] Failed to parse token: %v", err)
		return nil, fmt.Errorf("令牌解析失败: %w", err)
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("无效的令牌")
}

// ValidateAccessToken 验证浏览器/API 访问令牌的签名、时效和用途。
// 这里只验证 JWT 本身；会话是否仍有效由 HR 的 user_session 记录判定。
func (s *JWTService) ValidateAccessToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.UserID <= 0 || strings.TrimSpace(claims.Username) == "" {
		return nil, fmt.Errorf("访问令牌缺少用户身份")
	}
	if claims.Subject != fmt.Sprintf("access_%d", claims.UserID) {
		return nil, fmt.Errorf("令牌不是访问令牌")
	}
	return claims, nil
}

// ValidateRefreshToken 验证刷新令牌的签名、时效和用途。
func (s *JWTService) ValidateRefreshToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.UserID <= 0 || strings.TrimSpace(claims.Username) == "" {
		return nil, fmt.Errorf("刷新令牌缺少用户身份")
	}
	if claims.Subject != fmt.Sprintf("refresh_%d", claims.UserID) {
		return nil, fmt.Errorf("令牌不是刷新令牌")
	}
	return claims, nil
}

// ValidateOpenAPIToken 验证 OpenAPI Token 的签名、时效和用途。
// OpenAPI Token 只接受 openapi:<username> subject。
func (s *JWTService) ValidateOpenAPIToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}
	username := strings.TrimSpace(claims.Username)
	if username == "" {
		return nil, fmt.Errorf("OpenAPI Token 缺少用户名")
	}

	if claims.Subject != "openapi:"+username {
		return nil, fmt.Errorf("令牌不是 OpenAPI Token")
	}
	return claims, nil
}

// RefreshAccessToken 刷新访问令牌
func (s *JWTService) RefreshAccessToken(refreshTokenString string) (string, string, error) {
	claims, err := s.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("刷新令牌验证失败: %w", err)
	}

	tokenContext := UserTokenContext{
		UserID:         claims.UserID,
		Username:       claims.Username,
		Email:          claims.Email,
		CompanyCode:    claims.CompanyCode,
		CompanyName:    claims.CompanyName,
		CompanyLogoURL: claims.CompanyLogoURL,
	}
	if claims.DepartmentFullPath != nil {
		tokenContext.DepartmentFullPath = *claims.DepartmentFullPath
	}
	if claims.LeaderUsername != nil {
		tokenContext.LeaderUsername = *claims.LeaderUsername
	}

	newAccessToken, err := s.GenerateAccessTokenWithContext(tokenContext)
	if err != nil {
		return "", "", fmt.Errorf("生成新访问令牌失败: %w", err)
	}

	newRefreshToken, err := s.GenerateRefreshTokenWithContext(tokenContext)
	if err != nil {
		return "", "", fmt.Errorf("生成新刷新令牌失败: %w", err)
	}

	logger.Infof(nil, "[JWTService] Tokens refreshed for user: %s", claims.Username)
	return newAccessToken, newRefreshToken, nil
}

// ExtractUserID 从令牌中提取用户ID
func (s *JWTService) ExtractUserID(tokenString string) (int64, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return 0, err
	}
	return claims.UserID, nil
}

// ExtractUsername 从令牌中提取用户名
func (s *JWTService) ExtractUsername(tokenString string) (string, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return "", err
	}
	return claims.Username, nil
}
