package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/pkg/apperror"
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
func (s *AuthService) RegisterUser(ctx context.Context, username, email, password, companyAction, companyCode, companyName, companyLogoURL string) (int64, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if err := ValidateUserCode(username); err != nil {
		return 0, apperror.InvalidArgument(err.Error(), err)
	}

	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetUserByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return 0, apperror.Conflict("用户名已存在", nil)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, apperror.Internal(fmt.Errorf("检查用户名失败: %w", err))
	}

	// 检查邮箱是否已存在
	existingEmail, err := s.userRepo.GetUserByEmail(ctx, email)
	if err == nil && existingEmail != nil {
		return 0, apperror.Conflict("邮箱已被注册", nil)
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, apperror.Internal(fmt.Errorf("检查邮箱失败: %w", err))
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to hash password: %v", err)
		return 0, apperror.Internal(fmt.Errorf("密码加密失败: %w", err))
	}

	companyCode, err = s.resolveRegisterCompany(ctx, companyAction, companyCode, companyName, companyLogoURL, username)
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
	err = s.userRepo.CreateUser(ctx, user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to create user: %v", err)
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return 0, apperror.Conflict("用户名或邮箱已存在", err)
		}
		return 0, apperror.Internal(fmt.Errorf("用户创建失败: %w", err))
	}

	logger.Infof(nil, "[AuthService] User registered successfully: %s", username)
	return user.ID, nil
}

func (s *AuthService) resolveRegisterCompany(ctx context.Context, action, code, name, logoURL, createdBy string) (string, error) {
	action = strings.TrimSpace(action)
	code = strings.ToLower(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	logoURL = strings.TrimSpace(logoURL)
	if code == "" {
		return "", apperror.InvalidArgument("企业代码不能为空", nil)
	}
	if !companyCodePattern.MatchString(code) {
		return "", apperror.InvalidArgument("企业代码只能包含字母、数字、下划线和中划线", nil)
	}
	switch action {
	case "create":
		if name == "" {
			return "", apperror.InvalidArgument("创建企业时企业名称不能为空", nil)
		}
		if _, err := s.companyRepo.GetCompanyByCode(ctx, code); err == nil {
			return "", apperror.Conflict("企业代码已存在", nil)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperror.Internal(fmt.Errorf("检查企业代码失败: %w", err))
		}
		if _, err := s.companyRepo.GetCompanyByName(ctx, name); err == nil {
			return "", apperror.Conflict("企业名称已存在", nil)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", apperror.Internal(fmt.Errorf("检查企业名称失败: %w", err))
		}
		if err := s.companyRepo.CreateCompany(ctx, &model.Company{
			Code:      code,
			Name:      name,
			LogoURL:   logoURL,
			CreatedBy: createdBy,
		}); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return "", apperror.Conflict("企业代码或名称已存在", err)
			}
			return "", apperror.Internal(fmt.Errorf("创建企业失败: %w", err))
		}
		return code, nil
	case "join":
		if _, err := s.companyRepo.GetCompanyByCode(ctx, code); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return "", apperror.NotFound("企业不存在", err)
			}
			return "", apperror.Internal(fmt.Errorf("查询企业失败: %w", err))
		}
		return code, nil
	default:
		return "", apperror.InvalidArgument("企业操作类型必须是 create 或 join", nil)
	}
}

func (s *AuthService) SearchCompaniesFuzzy(ctx context.Context, keyword string, limit int) ([]*model.Company, error) {
	companies, err := s.companyRepo.SearchCompaniesFuzzy(ctx, keyword, limit)
	if err != nil {
		return nil, apperror.Internal(fmt.Errorf("搜索企业失败: %w", err))
	}
	return companies, nil
}

// ActivateUser 激活用户
func (s *AuthService) ActivateUser(ctx context.Context, userID int64) error {
	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, userID)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to get user %d: %v", userID, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("用户不存在", err)
		}
		return apperror.Internal(fmt.Errorf("查询用户失败: %w", err))
	}

	// 更新用户状态为active，并标记邮箱已验证
	user.Status = "active"
	user.EmailVerified = true
	err = s.userRepo.UpdateUser(ctx, user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to activate user %d: %v", userID, err)
		return apperror.Internal(fmt.Errorf("用户激活失败: %w", err))
	}

	logger.Infof(nil, "[AuthService] User activated successfully: %d", userID)
	return nil
}

// LoginUser 用户登录
func (s *AuthService) LoginUser(ctx context.Context, username, password string, remember bool) (*model.User, string, string, error) {
	// 获取用户信息
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		logger.Warnf(nil, "[AuthService] User not found: %s, error: %v", username, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, "", "", apperror.Unauthenticated("用户名或密码错误", err)
		}
		return nil, "", "", apperror.Internal(fmt.Errorf("查询登录用户失败: %w", err))
	}

	// 检查用户状态
	if !user.IsActive() {
		logger.Warnf(nil, "[AuthService] User not active: %s, status: %s", username, user.Status)
		return nil, "", "", apperror.PermissionDenied("账户未激活，请先验证邮箱", nil)
	}

	// 检查是否支持密码登录
	if !user.IsPasswordLoginSupported() {
		logger.Warnf(nil, "[AuthService] User does not support password login: %s, register_type: %s, has_password: %v", username, user.RegisterType, user.PasswordHash != "")
		return nil, "", "", apperror.PermissionDenied("该账户不支持密码登录", nil)
	}

	// 验证密码
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		logger.Warnf(nil, "[AuthService] Password mismatch for user: %s, error: %v", username, err)
		return nil, "", "", apperror.Unauthenticated("用户名或密码错误", err)
	}

	token, refreshToken, err := s.IssueTokensForUser(ctx, user, remember)
	if err != nil {
		return nil, "", "", err
	}

	logger.Infof(nil, "[AuthService] User logged in successfully: %s (remember: %v)", username, remember)
	return user, token, refreshToken, nil
}

func (s *AuthService) IssueTokensForUser(ctx context.Context, user *model.User, remember bool) (string, string, error) {
	if user == nil {
		return "", "", apperror.NotFound("用户不存在", nil)
	}
	if !user.IsActive() {
		logger.Warnf(nil, "[AuthService] User not active: %s, status: %s", user.Username, user.Status)
		return "", "", apperror.PermissionDenied("账户未激活，请联系管理员", nil)
	}

	jwtConfig := s.config.GetJWT()
	refreshTokenExpire := resolveRefreshTokenExpireSeconds(jwtConfig.RefreshTokenExpire, remember)

	tokenContext := s.buildUserTokenContext(ctx, user)
	refreshExpiresAt := time.Now().Add(time.Duration(refreshTokenExpire) * time.Second)

	// 生成 JWT Token（包含企业、组织架构信息）
	token, err := s.jwtService.GenerateAccessTokenWithContext(tokenContext)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to generate access token: %v", err)
		return "", "", apperror.Internal(fmt.Errorf("访问令牌生成失败: %w", err))
	}

	refreshToken, err := s.jwtService.GenerateRefreshTokenWithContextExpiresAt(tokenContext, refreshExpiresAt)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to generate refresh token: %v", err)
		return "", "", apperror.Internal(fmt.Errorf("刷新令牌生成失败: %w", err))
	}

	// 保存用户会话（使用自定义的Refresh Token有效期）
	err = s.saveUserSessionWithExpiresAt(ctx, user.ID, token, refreshToken, refreshExpiresAt)
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

func (s *AuthService) buildUserTokenContext(ctx context.Context, user *model.User) auth.UserTokenContext {
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
	company, err := s.companyRepo.GetCompanyByCode(ctx, user.CompanyCode)
	if err != nil {
		logger.Warnf(nil, "[AuthService] 查询企业信息失败 company_code=%s err=%v", user.CompanyCode, err)
		return tokenContext
	}
	tokenContext.CompanyName = company.Name
	tokenContext.CompanyLogoURL = company.LogoURL
	return tokenContext
}

// RefreshToken 刷新Token
func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	// 验证 RefreshToken
	session, err := s.userSessionRepo.GetUserSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", apperror.Unauthenticated("RefreshToken无效或已过期", err)
		}
		return "", "", apperror.Internal(fmt.Errorf("查询刷新会话失败: %w", err))
	}
	claims, err := s.jwtService.ValidateToken(refreshToken)
	if err != nil {
		return "", "", apperror.Unauthenticated("RefreshToken无效或已过期", err)
	}
	if len(claims.Subject) < len("refresh_") || claims.Subject[:len("refresh_")] != "refresh_" {
		return "", "", apperror.Unauthenticated("无效的RefreshToken", nil)
	}
	if claims.UserID != session.UserID {
		return "", "", apperror.Unauthenticated("RefreshToken无效", nil)
	}

	// 获取用户信息
	user, err := s.userRepo.GetUserByID(ctx, session.UserID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", apperror.Unauthenticated("刷新会话对应的用户不存在", err)
		}
		return "", "", apperror.Internal(fmt.Errorf("查询刷新用户失败: %w", err))
	}

	tokenContext := s.buildUserTokenContext(ctx, user)
	refreshExpiresAt := time.Time(session.ExpiresAt)
	newAccessToken, err := s.jwtService.GenerateAccessTokenWithContext(tokenContext)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to refresh access token: %v", err)
		return "", "", apperror.Internal(fmt.Errorf("访问令牌刷新失败: %w", err))
	}
	newRefreshToken, err := s.jwtService.GenerateRefreshTokenWithContextExpiresAt(tokenContext, refreshExpiresAt)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to refresh token: %v", err)
		return "", "", apperror.Internal(fmt.Errorf("刷新令牌生成失败: %w", err))
	}

	// 更新会话中的Token和RefreshToken
	err = s.userSessionRepo.UpdateUserSessionTokens(ctx, session.ID, newAccessToken, newRefreshToken)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to update tokens: %v", err)
		// 不返回错误，继续执行
	}

	logger.Infof(nil, "[AuthService] Tokens refreshed successfully for user: %s", user.Username)
	return newAccessToken, newRefreshToken, nil
}

// LogoutUser 用户登出
func (s *AuthService) LogoutUser(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return apperror.Unauthenticated("未提供认证令牌", nil)
	}

	if s.tokenPublisher != nil {
		if claims, err := s.jwtService.ValidateToken(token); err == nil {
			if err := s.tokenPublisher.InvalidateToken(ctx, claims.UserID, claims.Username, token, "logout"); err != nil {
				logger.Warnf(nil, "[AuthService] 发送 logout token 失效通知失败: %v", err)
			}
		} else {
			logger.Warnf(nil, "[AuthService] logout token 解析失败，仅停用会话: %v", err)
		}
	}

	// 停用用户会话
	err := s.userSessionRepo.DeactivateUserSession(ctx, token)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to deactivate user session: %v", err)
		return apperror.Internal(fmt.Errorf("登出失败: %w", err))
	}

	logger.Infof(nil, "[AuthService] User logged out successfully")
	return nil
}

// saveUserSession 保存用户会话
func (s *AuthService) saveUserSession(ctx context.Context, userID int64, token, refreshToken string) error {
	// 计算过期时间（24小时）
	expiresAt := models.Time(time.Now().Add(24 * time.Hour))

	// 创建用户会话
	err := s.userSessionRepo.CreateUserSession(ctx, userID, token, refreshToken, expiresAt, "", "")
	if err != nil {
		return fmt.Errorf("会话保存失败: %w", err)
	}

	return nil
}

// saveUserSessionWithExpire 保存用户会话（自定义过期时间）
func (s *AuthService) saveUserSessionWithExpire(ctx context.Context, userID int64, token, refreshToken string, expireSeconds int) error {
	// 计算过期时间
	expiresAt := time.Now().Add(time.Duration(expireSeconds) * time.Second)

	return s.saveUserSessionWithExpiresAt(ctx, userID, token, refreshToken, expiresAt)
}

// saveUserSessionWithExpiresAt 保存用户会话（指定绝对过期时间）
func (s *AuthService) saveUserSessionWithExpiresAt(ctx context.Context, userID int64, token, refreshToken string, expiresAt time.Time) error {
	modelExpiresAt := models.Time(expiresAt)

	// 创建用户会话
	err := s.userSessionRepo.CreateUserSession(ctx, userID, token, refreshToken, modelExpiresAt, "", "")
	if err != nil {
		return fmt.Errorf("会话保存失败: %w", err)
	}

	return nil
}

// GetUserByEmail 根据邮箱获取用户
func (s *AuthService) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
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
func (s *AuthService) ResetPasswordByEmail(ctx context.Context, email, newPassword string) error {
	// 获取用户信息
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to get user by email %s: %v", email, err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperror.NotFound("用户不存在", err)
		}
		return apperror.Internal(fmt.Errorf("查询用户失败: %w", err))
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to hash password: %v", err)
		return apperror.Internal(fmt.Errorf("密码加密失败: %w", err))
	}

	// 更新用户密码
	user.PasswordHash = string(hashedPassword)
	err = s.userRepo.UpdateUser(ctx, user)
	if err != nil {
		logger.Errorf(nil, "[AuthService] Failed to update user password: %v", err)
		return apperror.Internal(fmt.Errorf("密码更新失败: %w", err))
	}

	logger.Infof(nil, "[AuthService] Password reset successfully for user: %s", user.Username)
	return nil
}
