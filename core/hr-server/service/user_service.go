package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/hr-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/hr-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// UserService 用户服务
type UserService struct {
	userRepo        *repository.UserRepository
	natsService     *NATSService                    // ⭐ 新增：NATS 服务（可选，可能为 nil）
	userSessionRepo *repository.UserSessionRepository // ⭐ 新增：用户会话仓库（用于查询活跃会话）
}

// NewUserService 创建用户服务（依赖注入）
func NewUserService(userRepo *repository.UserRepository, natsService *NATSService, userSessionRepo *repository.UserSessionRepository) *UserService {
	return &UserService{
		userRepo:        userRepo,
		natsService:     natsService,
		userSessionRepo: userSessionRepo,
	}
}

// GetUserByUsername 根据用户名获取用户信息
func (s *UserService) GetUserByUsername(username string) (*model.User, error) {
	return s.userRepo.GetUserByUsername(username)
}

// SearchUsersFuzzy 模糊查询用户
func (s *UserService) SearchUsersFuzzy(keyword string, limit int) ([]*model.User, error) {
	// 限制查询数量，防止大量数据查询
	if limit <= 0 {
		limit = 10 // 默认10条
	}
	if limit > 100 {
		limit = 100 // 最大100条
	}
	return s.userRepo.SearchUsersFuzzy(keyword, limit)
}

// GetUsersByUsernames 批量获取用户信息
func (s *UserService) GetUsersByUsernames(usernames []string) ([]*model.User, error) {
	// 限制批量查询数量，防止大量数据查询
	if len(usernames) > 100 {
		logger.Warnf(nil, "[UserService] Too many usernames in batch query, limiting to 100")
		usernames = usernames[:100]
	}
	return s.userRepo.GetUsersByUsernames(usernames)
}

// UpdateUser 更新用户信息（只更新提供的字段，空字符串会被忽略）
func (s *UserService) UpdateUser(username string, nickname, signature, avatar, gender *string) (*model.User, error) {
	// 获取用户
	user, err := s.userRepo.GetUserByUsername(username)
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
	err = s.userRepo.UpdateUser(user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// AssignUserOrganization 分配用户组织架构
func (s *UserService) AssignUserOrganization(ctx context.Context, username string, departmentFullPath *string, leaderUsername *string) (*model.User, error) {
	// 获取用户
	user, err := s.userRepo.GetUserByUsername(username)
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
		leader, err := s.userRepo.GetUserByUsername(*leaderUsername)
		if err != nil {
			return nil, fmt.Errorf("Leader 用户不存在: %w", err)
		}
		user.LeaderUsername = leader.Username
	} else {
		user.LeaderUsername = "" // 清空 Leader
	}

	// 保存到数据库
	if err := s.userRepo.UpdateUser(user); err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}

	// ⭐ 发送 NATS 失效通知（如果组织架构发生变化）
	if s.natsService != nil {
		if err := s.natsService.InvalidateUserToken(ctx, user.ID, user.Username, "organization_changed", s.userSessionRepo); err != nil {
			logger.Warnf(ctx, "[UserService] 发送 token 失效通知失败: %v", err)
			// 不返回错误，因为用户更新已成功
		}
	}

	logger.Infof(ctx, "[UserService] User organization assigned: %s, department: %s, leader: %s", username, user.DepartmentFullPath, user.LeaderUsername)
	return user, nil
}

// GetUsersByDepartmentFullPath 根据部门完整路径获取用户列表
func (s *UserService) GetUsersByDepartmentFullPath(ctx context.Context, departmentFullPath string) ([]*model.User, error) {
	return s.userRepo.GetUsersByDepartmentFullPath(departmentFullPath)
}
