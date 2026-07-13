package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/hr-server/model"
	"github.com/kageos/kageos/core/hr-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService 用户服务
type UserService struct {
	userRepo        *repository.UserRepository
	companyRepo     *repository.CompanyRepository
	tokenPublisher  TokenPublisher                    // 可选：向 gateway 发布 token 命令
	userSessionRepo *repository.UserSessionRepository // ⭐ 新增：用户会话仓库（用于查询活跃会话）
}

// NewUserService 创建用户服务（依赖注入）
func NewUserService(userRepo *repository.UserRepository, companyRepo *repository.CompanyRepository, tokenPublisher TokenPublisher, userSessionRepo *repository.UserSessionRepository) *UserService {
	return &UserService{
		userRepo:        userRepo,
		companyRepo:     companyRepo,
		tokenPublisher:  tokenPublisher,
		userSessionRepo: userSessionRepo,
	}
}

// GetUserByUsername 根据用户名获取用户信息
func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(context.Background(

	// SearchUsersFuzzy 模糊查询用户
	), username)
}

func (s *UserService) SearchUsersFuzzy(keyword string, limit int) ([]*model.User, error) {
	// 限制查询数量，防止大量数据查询
	if limit <= 0 {
		limit = 10 // 默认10条
	}
	if limit > 100 {
		limit = 100 // 最大100条
	}
	return s.userRepo.SearchUsersFuzzy(context.Background(), keyword, limit)
}

func (s *UserService) ListUsersForSystem(req dto.SystemListUsersReq) ([]*model.User, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	return s.userRepo.ListUsersForSystem(context.Background(), req.Keyword, strings.TrimSpace(req.CompanyCode), strings.TrimSpace(req.Status), strings.TrimSpace(req.RegisterType), req.Page, req.PageSize)
}

func (s *UserService) CreateUserFromSystem(ctx context.Context, req dto.SystemCreateUserReq, createdBy string) (*model.User, error) {
	username := strings.ToLower(strings.TrimSpace(req.Username))
	if err := ValidateUserCode(username); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, fmt.Errorf("密码不能为空")
	}
	if len(req.Password) < 6 {
		return nil, fmt.Errorf("密码至少 6 位")
	}
	if existingUser, err := s.userRepo.GetUserByUsername(ctx, username); err == nil && existingUser != nil {
		return nil, fmt.Errorf("用户名已存在")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	companyCode, err := s.ensureSystemUserCompany(req.CompanyCode, req.CompanyName, req.CompanyLogoURL, firstNonEmptyString(createdBy, SystemUsername))
	if err != nil {
		return nil, err
	}
	if email == "" {
		email = placeholderSystemUserEmail(username, companyCode)
	} else if existingEmail, err := s.userRepo.GetUserByEmail(ctx, email); err == nil && existingEmail != nil {
		return nil, fmt.Errorf("邮箱已被注册")
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}

	status, err := normalizeSystemUserStatus(req.Status, "active")
	if err != nil {
		return nil, err
	}
	departmentFullPath := strings.TrimSpace(req.DepartmentFullPath)
	if departmentFullPath == "" {
		departmentFullPath = "/org/unassigned"
	}
	leaderUsername := strings.ToLower(strings.TrimSpace(req.LeaderUsername))
	if err := s.validateLeaderForCompany(leaderUsername, companyCode); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		logger.Errorf(ctx, "[UserService] Failed to hash system-created user password: %v", err)
		return nil, fmt.Errorf("密码加密失败")
	}

	user := &model.User{
		Username:           username,
		Email:              email,
		CompanyCode:        companyCode,
		PasswordHash:       string(hashedPassword),
		RegisterType:       "system",
		Status:             status,
		EmailVerified:      status != "pending",
		CreatedBy:          firstNonEmptyString(createdBy, SystemUsername),
		Nickname:           strings.TrimSpace(req.Nickname),
		DepartmentFullPath: departmentFullPath,
		LeaderUsername:     leaderUsername,
		Type:               model.UserTypeNormal,
	}
	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		logger.Errorf(ctx, "[UserService] Failed to create user from system: %v", err)
		return nil, fmt.Errorf("用户创建失败")
	}
	logger.Infof(ctx, "[UserService] system created user: %s company=%s", username, companyCode)
	return user, nil
}

func (s *UserService) UpdateUserFromSystem(ctx context.Context, username string, req dto.SystemUpdateUserReq, updatedBy string) (*model.User, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}
	companyChanged := false
	profileChanged := false

	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email != "" && email != user.Email {
			if existingEmail, err := s.userRepo.GetUserByEmail(ctx, email); err == nil && existingEmail != nil && existingEmail.Username != user.Username {
				return nil, fmt.Errorf("邮箱已被注册")
			} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("检查邮箱失败: %w", err)
			}
		}
		user.Email = email
		profileChanged = true
	}
	if req.Nickname != nil {
		user.Nickname = strings.TrimSpace(*req.Nickname)
		profileChanged = true
	}
	if req.Signature != nil {
		user.Signature = strings.TrimSpace(*req.Signature)
		profileChanged = true
	}
	if req.Avatar != nil {
		user.Avatar = strings.TrimSpace(*req.Avatar)
		profileChanged = true
	}
	if req.Gender != nil {
		user.Gender = strings.TrimSpace(*req.Gender)
		profileChanged = true
	}

	targetCompanyCode := user.CompanyCode
	if req.CompanyCode != nil {
		if user.Username == SystemUsername {
			return nil, fmt.Errorf("不能修改 system 用户所属企业")
		}
		targetCompanyCode, err = s.ensureSystemUserCompany(*req.CompanyCode, pointerStringValue(req.CompanyName), pointerStringValue(req.CompanyLogoURL), firstNonEmptyString(updatedBy, SystemUsername))
		if err != nil {
			return nil, err
		}
		if targetCompanyCode != user.CompanyCode {
			user.CompanyCode = targetCompanyCode
			companyChanged = true
		}
	}

	if req.DepartmentFullPath != nil {
		if user.Username == SystemUsername {
			return nil, fmt.Errorf("不能修改 system 用户组织归属")
		}
		user.DepartmentFullPath = strings.TrimSpace(*req.DepartmentFullPath)
		profileChanged = true
	}
	if req.LeaderUsername != nil {
		if user.Username == SystemUsername {
			return nil, fmt.Errorf("不能修改 system 用户上级")
		}
		leaderUsername := strings.ToLower(strings.TrimSpace(*req.LeaderUsername))
		if err := s.validateLeaderForCompany(leaderUsername, targetCompanyCode); err != nil {
			return nil, err
		}
		user.LeaderUsername = leaderUsername
		profileChanged = true
	}

	if !profileChanged && !companyChanged {
		return user, nil
	}
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}
	if companyChanged || req.DepartmentFullPath != nil || req.LeaderUsername != nil {
		s.invalidateUserTokens(ctx, user, "system_user_profile_changed")
	}
	logger.Infof(ctx, "[UserService] system updated user: %s", user.Username)
	return user, nil
}

func (s *UserService) ResetUserPasswordFromSystem(ctx context.Context, username, password string) (*model.User, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}
	if len(password) < 6 {
		return nil, fmt.Errorf("密码至少 6 位")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败")
	}
	user.PasswordHash = string(hashedPassword)
	if user.RegisterType != "email" && user.RegisterType != "system" {
		user.RegisterType = "system"
	}
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("重置密码失败: %w", err)
	}
	s.invalidateUserTokens(ctx, user, "system_user_password_reset")
	logger.Infof(ctx, "[UserService] system reset password for user: %s", user.Username)
	return user, nil
}

func (s *UserService) UpdateUserStatusFromSystem(ctx context.Context, username, status string) (*model.User, error) {
	user, err := s.userRepo.GetUserByUsername(ctx, strings.ToLower(strings.TrimSpace(username)))
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}
	status, err = normalizeSystemUserStatus(status, "")
	if err != nil {
		return nil, err
	}
	if user.Username == SystemUsername && status != "active" {
		return nil, fmt.Errorf("不能停用 system 用户")
	}
	if user.Status == status {
		return user, nil
	}
	user.Status = status
	if status == "active" {
		user.EmailVerified = true
	}
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户状态失败: %w", err)
	}
	if status != "active" {
		s.invalidateUserTokens(ctx, user, "system_user_status_changed")
	}
	logger.Infof(ctx, "[UserService] system updated user status: %s status=%s", user.Username, status)
	return user, nil
}

func (s *UserService) SearchUsersFuzzyInRequesterCompany(requesterUsername, keyword string, limit int) ([]*model.User, error) {
	requester, err := s.userRepo.GetUserByUsername(context.Background(), requesterUsername)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return s.userRepo.SearchUsersFuzzyByCompany(context.Background(), requester.CompanyCode, keyword, limit)
}

// GetUsersByUsernames 批量获取用户信息
func (s *UserService) GetUsersByUsernames(usernames []string) ([]*model.User, error) {
	// 限制批量查询数量，防止大量数据查询
	if len(usernames) > 100 {
		logger.Warnf(nil, "[UserService] Too many usernames in batch query, limiting to 100")
		usernames = usernames[:100]
	}
	return s.userRepo.GetUsersByUsernames(context.Background(), usernames)
}

func (s *UserService) GetUsersByUsernamesInRequesterCompany(requesterUsername string, usernames []string) ([]*model.User, error) {
	if len(usernames) > 100 {
		logger.Warnf(nil, "[UserService] Too many usernames in batch query, limiting to 100")
		usernames = usernames[:100]
	}
	requester, err := s.userRepo.GetUserByUsername(context.Background(), requesterUsername)
	if err != nil {
		return nil, err
	}
	return s.userRepo.GetUsersByUsernamesAndCompany(context.Background(), usernames, requester.CompanyCode)
}

func (s *UserService) GetCompaniesByCodes(codes []string) ([]*model.Company, error) {
	return s.companyRepo.GetCompaniesByCodes(context.Background(

	// UpdateUser 更新用户信息（只更新提供的字段，空字符串会被忽略）
	), codes)
}

func (s *UserService) UpdateUser(username string, nickname, signature, avatar, gender *string) (*model.User, error) {
	// 获取用户
	user, err := s.userRepo.GetUserByUsername(context.Background(), username)
	if err != nil {
		return nil, err
	}

	// 更新字段（只更新非 nil 的字段）
	if nickname != nil {
		user.Nickname = *nickname
	}
	if signature != nil {
		user.Signature = *signature
	}
	if avatar != nil {
		user.Avatar = *avatar
	}
	if gender != nil {
		user.Gender = *gender
	}

	// 保存更新
	err = s.userRepo.UpdateUser(context.Background(), user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// AssignUserOrganization 分配用户组织架构
func (s *UserService) AssignUserOrganization(ctx context.Context, username string, departmentFullPath *string, leaderUsername *string) (*model.User, error) {
	// 获取用户
	user, err := s.userRepo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("用户不存在: %w", err)
	}

	// 更新部门和 Leader
	if departmentFullPath != nil {
		user.DepartmentFullPath = *departmentFullPath
	} else {
		user.DepartmentFullPath = "" // 清空部门
	}

	if leaderUsername != nil {
		// 验证 Leader 是否存在
		leader, err := s.userRepo.GetUserByUsername(ctx, *leaderUsername)
		if err != nil {
			return nil, fmt.Errorf("Leader 用户不存在: %w", err)
		}
		user.LeaderUsername = leader.Username
	} else {
		user.LeaderUsername = "" // 清空 Leader
	}

	// 保存到数据库
	if err := s.userRepo.UpdateUser(ctx, user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	// ⭐ 发送 NATS 失效通知（如果组织架构发生变化）
	if s.tokenPublisher != nil {
		if err := s.tokenPublisher.InvalidateUserToken(ctx, user.ID, user.Username, "organization_changed", s.userSessionRepo); err != nil {
			logger.Warnf(ctx, "[UserService] 发送 token 失效通知失败: %v", err)
			// 不返回错误，因为用户更新已成功
		}
	}

	logger.Infof(ctx, "[UserService] User organization assigned: %s, department: %s, leader: %s", username, user.DepartmentFullPath, user.LeaderUsername)
	return user, nil
}

// GetUsersByDepartmentFullPath 根据部门完整路径获取用户列表
func (s *UserService) GetUsersByDepartmentFullPath(ctx context.Context, departmentFullPath string) ([]*model.User, error) {
	return s.userRepo.GetUsersByDepartmentFullPath(ctx, departmentFullPath)
}

func (s *UserService) ensureSystemUserCompany(code, name, logoURL, createdBy string) (string, error) {
	if s.companyRepo == nil {
		return "", fmt.Errorf("企业仓库未初始化")
	}
	code = strings.ToLower(strings.TrimSpace(code))
	name = strings.TrimSpace(name)
	logoURL = strings.TrimSpace(logoURL)
	if code == "" {
		code = defaultCompanyCode()
	}
	if !companyCodePattern.MatchString(code) {
		return "", fmt.Errorf("企业代码只能包含字母、数字、下划线和中划线")
	}
	if _, err := s.companyRepo.GetCompanyByCode(context.Background(), code); err == nil {
		return code, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("检查企业失败: %w", err)
	}
	if name == "" {
		return "", fmt.Errorf("企业不存在，请填写企业名称")
	}
	if err := s.companyRepo.CreateCompany(context.Background(), &model.Company{
		Code:      code,
		Name:      name,
		LogoURL:   logoURL,
		CreatedBy: createdBy,
	}); err != nil {
		return "", fmt.Errorf("创建企业失败: %w", err)
	}
	return code, nil
}

func (s *UserService) validateLeaderForCompany(leaderUsername, companyCode string) error {
	if leaderUsername == "" {
		return nil
	}
	leader, err := s.userRepo.GetUserByUsername(context.Background(), leaderUsername)
	if err != nil {
		return fmt.Errorf("Leader 用户不存在: %w", err)
	}
	if strings.TrimSpace(companyCode) != "" && leader.CompanyCode != companyCode {
		return fmt.Errorf("Leader 必须属于同一企业")
	}
	return nil
}

func (s *UserService) invalidateUserTokens(ctx context.Context, user *model.User, reason string) {
	if s.tokenPublisher == nil || user == nil {
		return
	}
	if err := s.tokenPublisher.InvalidateUserToken(ctx, user.ID, user.Username, reason, s.userSessionRepo); err != nil {
		logger.Warnf(ctx, "[UserService] 发送 token 失效通知失败: %v", err)
	}
}

func normalizeSystemUserStatus(status, defaultStatus string) (string, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		status = defaultStatus
	}
	switch status {
	case "active", "pending", "disabled":
		return status, nil
	default:
		return "", fmt.Errorf("用户状态必须是 active、pending 或 disabled")
	}
}

func placeholderSystemUserEmail(username, companyCode string) string {
	domainCode := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(companyCode)), "_", "-")
	if domainCode == "" {
		domainCode = "default"
	}
	return username + "@" + domainCode + ".local"
}

func pointerStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
