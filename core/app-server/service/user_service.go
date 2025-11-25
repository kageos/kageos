package service

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// UserService 用户服务
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService 创建用户服务（依赖注入）
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
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

