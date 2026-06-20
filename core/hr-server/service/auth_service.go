package service

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/pkg/auth"
	appconfig "github.com/kageos/kageos/pkg/config"
	"github.com/kageos/kageos/pkg/gormx/models"
	"github.com/kageos/kageos/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthService 认证服务
type AuthService struct {
	config          *appconfig.HRServerConfig
	jwtService      *auth.JWTService
	userRepo        *repository.UserRepository
	companyRepo     *repository.CompanyRepository
	userSessionRepo *repository.UserSessionRepository
	tokenPublisher  TokenPublisher // 可选：向 gateway 发布 token 命令
}

// NewAuthService 创建认证服务（依赖注入）
func NewAuthService(userRepo *repository.UserRepository, companyRepo *repository.CompanyRepository, userSessionRepo *repository.UserSessionRepository, tokenPublisher TokenPublisher) *AuthService {
	config := appconfig.GetHRServerConfig()
	jwtService := auth.NewJWTService()
	return &AuthService{
		config:          config,
		jwtService:      jwtService,
		userRepo:        userRepo,
		companyRepo:     companyRepo,
		userSessionRepo: userSessionRepo,
		tokenPublisher:  tokenPublisher,
	}
}

var companyCodePattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

const (
	defaultRefreshTokenExpireSeconds  = 24 * 3600
	rememberRefreshTokenExpireSeconds = 30 * 24 * 3600
)

// RegisterUser 注册用户
func (s *AuthService) RegisterUser(username, email, password, companyAction, companyCode, companyName, companyLogoURL string) (int64, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if err := ValidateUserCode(username); err != nil {
		return 0, err
	}

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

	companyCode, err = s.resolveRegisterCompany(companyAction, companyCode, companyName, companyLogoURL, username)
	if err != nil {
		return 0, err
	}

	// 创建用户（不再分配 HostID，Host 和 Nats 绑定在 App 上）
	// ⭐ 默认分配到未分配组织
	user := &model.User{
		Username:           username,
		Email:              email,
		CompanyCode:        companyCode,
		PasswordHash:       string(hashedPassword),
		RegisterType:       "email",
		Status:             "pending", // 待邮箱验证
		EmailVerified:      false,
		CreatedBy:          "system",
		DepartmentFullPath: "/org/unassigned", // ⭐ 默认分配到未分配组织
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

func (s *AuthService) resolveRegisterCompany(action, code, name, logoURL, createdBy string) (string, error) {
	action = strings.TrimSpace(action)
	code = strings.ToLower(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	logoURL = strings.TrimSpace(logoURL)
	if code == "" {
		return "", fmt.Errorf("企业代码不能为空")
	}
	if !companyCodePattern.MatchString(code) {
		return "", fmt.Errorf("企业代码只能包含字母、数字、下划线和中划线")
	}
	switch action {
	case "create":
		if name == "" {
			return "", fmt.Errorf("创建企业时企业名称不能为空")
		}
		if _, err := s.companyRepo.GetCompanyByCode(code); err == nil {
			return "", fmt.Errorf("企业代码已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("检查企业代码失败: %w", err)
		}
		if _, err := s.companyRepo.GetCompanyByName(name); err == nil {
			return "", fmt.Errorf("企业名称已存在")
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("检查企业名称失败: %w", err)
		}
		if err := s.companyRepo.CreateCompany(&model.Company{
			Code:      code,
			Name:      name,
			LogoURL:   logoURL,
			CreatedBy: createdBy,
		}); err != nil {
			return "", fmt.Errorf("创建企业失败: %w", err)
		}
		return code, nil
	case "join":
		if _, err := s.companyRepo.GetCompanyByCode(code); err != nil {
			return "", fmt.Errorf("企业不存在")
		}
		return code, nil
	default:
		return "", fmt.Errorf("企业操作类型必须是 create 或 join")
	}
}

func (s *AuthService) SearchCompaniesFuzzy(keyword string, limit int) ([]*model.Company, error) {
	return s.companyRepo.SearchCompaniesFuzzy(keyword, limit)
}

// CreateUserBySecretKey 超管一键创建用户（免邮箱验证，仅 system 用户可调用，用于创建测试用户）
func (s *AuthService) CreateUserBySecretKey(username, password string) (int64, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if err := ValidateUserCode(username); err != nil {
		return 0, err
	}

	existingUser, err := s.userRepo.GetUserByUsername(username)
	if err == nil && existingUser != nil {
		return 0, fmt.Errorf("用户名已存在")
	}

	// 占位邮箱，保证唯一且不与其他用户冲突
	placeholderEmail := username + "@test.local"
	existingEmail, err := s.userRepo.GetUserByEmail(placeholderEmail)
	if err == nil && existingEmail != nil {
		return 0, fmt.Errorf("邮箱占位冲突，请换用户名")
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to hash password: %v", err)
		return 0, fmt.Errorf("密码加密失败")
	}

	user := &model.User{
		Username:           username,
		Email:              placeholderEmail,
		CompanyCode:        defaultCompanyCode(),
		PasswordHash:       string(hashedPassword),
		RegisterType:       "system",
		Status:             "active",
		EmailVerified:      true,
		CreatedBy:          "admin_secret",
		DepartmentFullPath: "/org/unassigned",
	}

	err = s.userRepo.CreateUser(user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to create user by secret: %v", err)
		return 0, fmt.Errorf("用户创建失败")
	}

	logger.Infof(nil, "[AuthService] User created by secret key: %s", username)
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
		logger.Warnf(nil, "[AuthService] User not found: %s, error: %v", username, err)
		return nil, "", "", fmt.Errorf("用户名或密码错误")
	}

	// 检查用户状态
	if !user.IsActive() {
		logger.Warnf(nil, "[AuthService] User not active: %s, status: %s", username, user.Status)
		return nil, "", "", fmt.Errorf("账户未激活，请先验证邮箱")
	}

	// 检查是否支持密码登录
	if !user.IsPasswordLoginSupported() {
		logger.Warnf(nil, "[AuthService] User does not support password login: %s, register_type: %s, has_password: %v", username, user.RegisterType, user.PasswordHash != "")
		return nil, "", "", fmt.Errorf("该账户不支持密码登录")
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		logger.Warnf(nil, "[AuthService] Password mismatch for user: %s, error: %v", username, err)
		return nil, "", "", fmt.Errorf("用户名或密码错误")
	}

	token, refreshToken, err := s.IssueTokensForUser(user, remember)
	if err != nil {
		return nil, "", "", err
	}

	logger.Infof(nil, "[AuthService] User logged in successfully: %s (remember: %v)", username, remember)
	return user, token, refreshToken, nil
}

func (s *AuthService) IssueTokensForUser(user *model.User, remember bool) (string, string, error) {
	if user == nil {
		return "", "", fmt.Errorf("用户不存在")
	}
	if !user.IsActive() {
		logger.Warnf(nil, "[AuthService] User not active: %s, status: %s", user.Username, user.Status)
		return "", "", fmt.Errorf("账户未激活，请联系管理员")
	}

	jwtConfig := s.config.GetJWT()
	refreshTokenExpire := resolveRefreshTokenExpireSeconds(jwtConfig.RefreshTokenExpire, remember)

	tokenContext := s.buildUserTokenContext(user)
	refreshExpiresAt := time.Now().Add(time.Duration(refreshTokenExpire) * time.Second)

	// 生成 JWT Token（包含企业、组织架构信息）
	token, err := s.jwtService.GenerateAccessTokenWithContext(tokenContext)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to generate access token: %v", err)
		return "", "", fmt.Errorf("访问令牌生成失败")
	}

	refreshToken, err := s.jwtService.GenerateRefreshTokenWithContextExpiresAt(tokenContext, refreshExpiresAt)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to generate refresh token: %v", err)
		return "", "", fmt.Errorf("刷新令牌生成失败")
	}

	// 保存用户会话（使用自定义的Refresh Token有效期）
	err = s.saveUserSessionWithExpiresAt(user.ID, token, refreshToken, refreshExpiresAt)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to save user session: %v", err)
		// 不返回错误，继续执行
	}

	return token, refreshToken, nil
}

func resolveRefreshTokenExpireSeconds(configured int, remember bool) int {
	if configured <= 0 {
		configured = defaultRefreshTokenExpireSeconds
	}
	if remember && configured < rememberRefreshTokenExpireSeconds {
		return rememberRefreshTokenExpireSeconds
	}
	return configured
}

func (s *AuthService) buildUserTokenContext(user *model.User) auth.UserTokenContext {
	tokenContext := auth.UserTokenContext{
		UserID:             user.ID,
		Username:           user.Username,
		Email:              user.Email,
		CompanyCode:        user.CompanyCode,
		DepartmentFullPath: user.DepartmentFullPath,
		LeaderUsername:     user.LeaderUsername,
	}
	if user.CompanyCode == "" || s.companyRepo == nil {
		return tokenContext
	}
	company, err := s.companyRepo.GetCompanyByCode(user.CompanyCode)
	if err != nil {
		logger.Warnf(nil, "[AuthService] 查询企业信息失败 company_code=%s err=%v", user.CompanyCode, err)
		return tokenContext
	}
	tokenContext.CompanyName = company.Name
	tokenContext.CompanyLogoURL = company.LogoURL
	return tokenContext
}

// RefreshToken 刷新Token
func (s *AuthService) RefreshToken(refreshToken string) (string, string, error) {
	// 验证 RefreshToken
	session, err := s.userSessionRepo.GetUserSessionByRefreshToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("RefreshToken无效或已过期")
	}
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return "", "", fmt.Errorf("RefreshToken无效或已过期")
	}
	if len(claims.Subject) < len("refresh_") || claims.Subject[:len("refresh_")] != "refresh_" {
		return "", "", fmt.Errorf("无效的RefreshToken")
	}
	if claims.UserID != session.UserID {
		return "", "", fmt.Errorf("RefreshToken无效")
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(session.UserID)
	if err != nil {
		return "", "", fmt.Errorf("用户不存在")
	}

	tokenContext := s.buildUserTokenContext(user)
	refreshExpiresAt := time.Time(session.ExpiresAt)
	newAccessToken, err := s.jwtService.GenerateAccessTokenWithContext(tokenContext)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to refresh access token: %v", err)
		return "", "", fmt.Errorf("Token刷新失败")
	}
	newRefreshToken, err := s.jwtService.GenerateRefreshTokenWithContextExpiresAt(tokenContext, refreshExpiresAt)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to refresh token: %v", err)
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
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("未提供认证令牌")
	}

	if s.tokenPublisher != nil {
		if claims, err := s.jwtService.ValidateToken(token); err == nil {
			if err := s.tokenPublisher.InvalidateToken(nil, claims.UserID, claims.Username, token, "logout"); err != nil {
				logger.Warnf(nil, "[AuthService] 发送 logout token 失效通知失败: %v", err)
			}
		} else {
			logger.Warnf(nil, "[AuthService] logout token 解析失败，仅停用会话: %v", err)
		}
	}

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
	expiresAt := time.Now().Add(time.Duration(expireSeconds) * time.Second)

	return s.saveUserSessionWithExpiresAt(userID, token, refreshToken, expiresAt)
}

// saveUserSessionWithExpiresAt 保存用户会话（指定绝对过期时间）
func (s *AuthService) saveUserSessionWithExpiresAt(userID int64, token, refreshToken string, expiresAt time.Time) error {
	modelExpiresAt := models.Time(expiresAt)

	// 创建用户会话
	err := s.userSessionRepo.CreateUserSession(userID, token, refreshToken, modelExpiresAt, "", "")
	if err != nil {
		return fmt.Errorf("会话保存失败: %w", err)
	}

	return nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *AuthService) GetUserByEmail(email string) (*model.User, error) {
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GeneratePasswordResetToken 生成密码重置token
func (s *AuthService) GeneratePasswordResetToken(userID int64, username, email string) (string, error) {
	return s.jwtService.GeneratePasswordResetToken(userID, username, email)
}

// ResetPasswordByEmail 通过邮箱重置密码（简化版，用于测试阶段）
func (s *AuthService) ResetPasswordByEmail(email, newPassword string) error {
	// 获取用户信息
	user, err := s.userRepo.GetUserByEmail(email)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to get user by email %s: %v", email, err)
		return fmt.Errorf("用户不存在")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to hash password: %v", err)
		return fmt.Errorf("密码加密失败")
	}

	// 更新用户密码
	user.PasswordHash = string(hashedPassword)
	err = s.userRepo.UpdateUser(user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to update user password: %v", err)
		return fmt.Errorf("密码更新失败")
	}

	logger.Infof(nil, "[AuthService] Password reset successfully for user: %s", user.Username)
	return nil
}
