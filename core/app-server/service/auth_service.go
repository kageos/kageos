package service

import (
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	appconfig "github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

// AuthService 认证服务
type AuthService struct {
	config          *appconfig.AppServerConfig
	jwtService      *JWTService
	userRepo        *repository.UserRepository
	hostRepo        *repository.HostRepository
	userSessionRepo *repository.UserSessionRepository
}

// NewAuthService 创建认证服务（依赖注入）
func NewAuthService(userRepo *repository.UserRepository, hostRepo *repository.HostRepository, userSessionRepo *repository.UserSessionRepository) *AuthService {
	config := appconfig.GetAppServerConfig()
	jwtService := NewJWTService()
	return &AuthService{
		config:          config,
		jwtService:      jwtService,
		userRepo:        userRepo,
		hostRepo:        hostRepo,
		userSessionRepo: userSessionRepo,
	}
}

// RegisterUser 注册用户
func (s *AuthService) RegisterUser(username, email, password string) (int64, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetUserByUsername(username)
	if err == nil && existingUser != nil {
		return 0, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingEmail, err := s.userRepo.GetUserByEmail(email)
	if err == nil && existingEmail != nil {
		return 0, fmt.Errorf("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to hash password: %v", err)
		return 0, fmt.Errorf("密码加密失败")
	}

	// 获取可用的host（选择app_count最小的host）
	hosts, err := s.hostRepo.GetHostList()
	if err != nil || len(hosts) == 0 {
		logger.Errorf(nil, "[AuthService] Failed to get available host: %v", err)
		return 0, fmt.Errorf("无法获取可用的主机")
	}
	host := hosts[0] // 选择第一个可用的host

	// 创建用户
	user := &model.User{
		Username:      username,
		Email:         email,
		PasswordHash:  string(hashedPassword),
		RegisterType:  "email",
		Status:        "pending", // 待邮箱验证
		EmailVerified: false,
		CreatedBy:     "system",
		HostID:        host.ID, // 分配host
	}

	// 保存到数据库
	err = s.userRepo.CreateUser(user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to create user: %v", err)
		return 0, fmt.Errorf("用户创建失败")
	}

	logger.Infof(nil, "[AuthService] User registered successfully: %s", username)
	return user.ID, nil
}

// ActivateUser 激活用户
func (s *AuthService) ActivateUser(userID int64) error {
	// 获取用户信息
	user, err := s.userRepo.GetUserByID(userID)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to get user %d: %v", userID, err)
		return fmt.Errorf("用户不存在")
	}

	// 更新用户状态为active，并标记邮箱已验证
	user.Status = "active"
	user.EmailVerified = true
	err = s.userRepo.UpdateUser(user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to activate user %d: %v", userID, err)
		return fmt.Errorf("用户激活失败")
	}

	logger.Infof(nil, "[AuthService] User activated successfully: %d", userID)
	return nil
}

// LoginUser 用户登录
func (s *AuthService) LoginUser(username, password string, remember bool) (*model.User, string, string, error) {
	// 获取用户信息
	user, err := s.userRepo.GetUserByUsername(username)
	if err != nil {
		return nil, "", "", fmt.Errorf("用户名或密码错误")
	}

	// 检查用户状态
	if user.Status != "active" {
		return nil, "", "", fmt.Errorf("账户未激活，请先验证邮箱")
	}

	// 检查是否支持密码登录
	if !user.IsPasswordLoginSupported() {
		return nil, "", "", fmt.Errorf("该账户不支持密码登录")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, "", "", fmt.Errorf("用户名或密码错误")
	}

	// 根据"记住我"设置不同的Refresh Token有效期
	refreshTokenExpire := s.config.JWT.RefreshTokenExpire
	if remember {
		// 记住我：延长到30天
		refreshTokenExpire = 30 * 24 * 3600 // 30天
	}

	// 生成 JWT Token
	token, err := s.jwtService.GenerateAccessToken(user.ID, user.Username, user.Email)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to generate access token: %v", err)
		return nil, "", "", fmt.Errorf("访问令牌生成失败")
	}

	refreshToken, err := s.jwtService.GenerateRefreshToken(user.ID, user.Username, user.Email)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to generate refresh token: %v", err)
		return nil, "", "", fmt.Errorf("刷新令牌生成失败")
	}

	// 保存用户会话（使用自定义的Refresh Token有效期）
	err = s.saveUserSessionWithExpire(user.ID, token, refreshToken, refreshTokenExpire)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to save user session: %v", err)
		// 不返回错误，继续执行
	}

	logger.Infof(nil, "[AuthService] User logged in successfully: %s (remember: %v)", username, remember)
	return user, token, refreshToken, nil
}

// RefreshToken 刷新Token
func (s *AuthService) RefreshToken(refreshToken string) (string, string, error) {
	// 验证 RefreshToken
	session, err := s.userSessionRepo.GetUserSessionByRefreshToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("RefreshToken无效或已过期")
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(session.UserID)
	if err != nil {
		return "", "", fmt.Errorf("用户不存在")
	}

	// 使用 JWT 服务刷新令牌（同时生成新的 Refresh Token）
	newAccessToken, newRefreshToken, err := s.jwtService.RefreshAccessToken(refreshToken)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to refresh tokens: %v", err)
		return "", "", fmt.Errorf("Token刷新失败")
	}

	// 更新会话中的Token和RefreshToken
	err = s.userSessionRepo.UpdateUserSessionTokens(session.ID, newAccessToken, newRefreshToken)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to update tokens: %v", err)
		// 不返回错误，继续执行
	}

	logger.Infof(nil, "[AuthService] Tokens refreshed successfully for user: %s", user.Username)
	return newAccessToken, newRefreshToken, nil
}

// LogoutUser 用户登出
func (s *AuthService) LogoutUser(token string) error {
	// 停用用户会话
	err := s.userSessionRepo.DeactivateUserSession(token)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to deactivate user session: %v", err)
		return fmt.Errorf("登出失败")
	}

	logger.Infof(nil, "[AuthService] User logged out successfully")
	return nil
}

// saveUserSession 保存用户会话
func (s *AuthService) saveUserSession(userID int64, token, refreshToken string) error {
	// 计算过期时间（24小时）
	expiresAt := models.Time(time.Now().Add(24 * time.Hour))

	// 创建用户会话
	err := s.userSessionRepo.CreateUserSession(userID, token, refreshToken, expiresAt, "", "")
	if err != nil {
		return fmt.Errorf("会话保存失败: %w", err)
	}

	return nil
}

// saveUserSessionWithExpire 保存用户会话（自定义过期时间）
func (s *AuthService) saveUserSessionWithExpire(userID int64, token, refreshToken string, expireSeconds int) error {
	// 计算过期时间
	expiresAt := models.Time(time.Now().Add(time.Duration(expireSeconds) * time.Second))

	// 创建用户会话
	err := s.userSessionRepo.CreateUserSession(userID, token, refreshToken, expiresAt, "", "")
	if err != nil {
		return fmt.Errorf("会话保存失败: %w", err)
	}

	return nil
}
