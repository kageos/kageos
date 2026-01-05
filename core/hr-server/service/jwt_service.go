package service

import (
	"fmt"
	"time"

	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/golang-jwt/jwt/v5"
)

// JWTService JWT服务
type JWTService struct {
	config *appconfig.JWTConfig
}

// NewJWTService 创建JWT服务
func NewJWTService() *JWTService {
	// ⚠️ 使用全局配置获取 JWT 配置（与 app-server 共享）
	globalConfig := appconfig.GetGlobalSharedConfig()
	return &JWTService{
		config: &globalConfig.JWT,
	}
}

// JWTClaims JWT声明
type JWTClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	
	// ⭐ 新增：组织架构信息（只存储路径和用户名，不存储ID和名称）
	DepartmentFullPath *string `json:"department_full_path,omitempty"`
	LeaderUsername    *string `json:"leader_username,omitempty"`
	
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成访问令牌（兼容旧接口，不包含组织架构信息）
func (s *JWTService) GenerateAccessToken(userID int64, username, email string) (string, error) {
	return s.GenerateAccessTokenWithHR(userID, username, email, "", "")
}

// GenerateAccessTokenWithHR 生成访问令牌（包含组织架构信息，只存储路径，不存储名称）
// departmentFullPath: 部门完整路径（可选，如果为空字符串则不包含部门信息）
// leaderUsername: Leader 用户名（可选，如果为空字符串则不包含 Leader 信息）
func (s *JWTService) GenerateAccessTokenWithHR(userID int64, username, email string, departmentFullPath string, leaderUsername string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.config.AccessTokenExpire) * time.Second)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}
	
	// ⭐ 新增：组织架构信息（只存储路径，不存储名称）
	if departmentFullPath != "" {
		claims.DepartmentFullPath = &departmentFullPath
	}
	
	if leaderUsername != "" {
		claims.LeaderUsername = &leaderUsername
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		logger.Errorf(nil, "[JWTService] Failed to generate access token: %v", err)
		return "", fmt.Errorf("生成访问令牌失败: %w", err)
	}

	logger.Infof(nil, "[JWTService] Access token generated for user: %s", username)
	return tokenString, nil
}

// GenerateRefreshToken 生成刷新令牌
func (s *JWTService) GenerateRefreshToken(userID int64, username, email string) (string, error) {
	now := time.Now()
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   fmt.Sprintf("refresh_%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(s.config.RefreshTokenExpire) * time.Second)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		logger.Errorf(nil, "[JWTService] Failed to generate refresh token: %v", err)
		return "", fmt.Errorf("生成刷新令牌失败: %w", err)
	}

	logger.Infof(nil, "[JWTService] Refresh token generated for user: %s", username)
	return tokenString, nil
}

// ValidateToken 验证令牌
func (s *JWTService) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
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

// RefreshAccessToken 刷新访问令牌
func (s *JWTService) RefreshAccessToken(refreshTokenString string) (string, string, error) {
	// 验证刷新令牌
	claims, err := s.ValidateToken(refreshTokenString)
	if err != nil {
		return "", "", fmt.Errorf("刷新令牌验证失败: %w", err)
	}

	// 检查是否是刷新令牌
	if claims.Subject[:7] != "refresh_" {
		return "", "", fmt.Errorf("无效的刷新令牌")
	}

	// 生成新的访问令牌
	newAccessToken, err := s.GenerateAccessToken(claims.UserID, claims.Username, claims.Email)
	if err != nil {
		return "", "", fmt.Errorf("生成新访问令牌失败: %w", err)
	}

	// 生成新的刷新令牌（延长有效期）
	newRefreshToken, err := s.GenerateRefreshToken(claims.UserID, claims.Username, claims.Email)
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

// GeneratePasswordResetToken 生成密码重置令牌
func (s *JWTService) GeneratePasswordResetToken(userID int64, username, email string) (string, error) {
	now := time.Now()
	// 重置密码token有效期为1小时
	expiresAt := now.Add(1 * time.Hour)
	
	claims := JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.config.Issuer,
			Subject:   fmt.Sprintf("reset_password_%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.config.Secret))
	if err != nil {
		logger.Errorf(nil, "[JWTService] Failed to generate password reset token: %v", err)
		return "", fmt.Errorf("生成密码重置令牌失败: %w", err)
	}

	logger.Infof(nil, "[JWTService] Password reset token generated for user: %s", username)
	return tokenString, nil
}

// ValidatePasswordResetToken 验证密码重置令牌
func (s *JWTService) ValidatePasswordResetToken(tokenString string) (*JWTClaims, error) {
	claims, err := s.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// 检查是否是密码重置令牌
	if len(claims.Subject) < 16 || claims.Subject[:16] != "reset_password_" {
		return nil, fmt.Errorf("无效的密码重置令牌")
	}

	return claims, nil
}
