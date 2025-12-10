package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/utils"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"gorm.io/gorm"
)

// normalizeExtraConfig 规范化 extra_config 字段，确保是有效的 JSON 或 NULL
func normalizeExtraConfig(extraConfig string) (*string, error) {
	// 如果为空，返回 nil（允许 NULL）
	if extraConfig == "" {
		return nil, nil
	}

	// 验证是否为有效的 JSON
	var temp interface{}
	if err := json.Unmarshal([]byte(extraConfig), &temp); err != nil {
		return nil, fmt.Errorf("extra_config 不是有效的 JSON: %w", err)
	}

	// 重新序列化以确保格式正确
	normalized, err := json.Marshal(temp)
	if err != nil {
		return nil, fmt.Errorf("序列化 extra_config 失败: %w", err)
	}

	result := string(normalized)
	return &result, nil
}

// LLMService LLM 服务
type LLMService struct {
	repo *repository.LLMRepository
}

// NewLLMService 创建 LLM 服务
func NewLLMService(repo *repository.LLMRepository) *LLMService {
	return &LLMService{repo: repo}
}

// GetLLMConfig 获取 LLM 配置
func (s *LLMService) GetLLMConfig(ctx context.Context, id int64) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("LLM配置不存在")
		}
		return nil, fmt.Errorf("获取LLM配置失败: %w", err)
	}
	return cfg, nil
}

// GetDefaultLLMConfig 获取默认 LLM 配置
func (s *LLMService) GetDefaultLLMConfig(ctx context.Context) (*model.LLMConfig, error) {
	cfg, err := s.repo.GetDefault()
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("未设置默认LLM配置")
		}
		return nil, fmt.Errorf("获取默认LLM配置失败: %w", err)
	}
	return cfg, nil
}

// ListLLMConfigs 获取 LLM 配置列表
func (s *LLMService) ListLLMConfigs(ctx context.Context, scope string, page, pageSize int) ([]*model.LLMConfig, int64, error) {
	currentUser := contextx.GetRequestUser(ctx)
	offset := (page - 1) * pageSize
	if pageSize <= 0 {
		pageSize = 10
	}
	return s.repo.List(scope, currentUser, offset, pageSize)
}

// CreateLLMConfig 创建 LLM 配置
func (s *LLMService) CreateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	cfg.CreatedBy = user
	cfg.UpdatedBy = user

	// 验证必填字段
	if cfg.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}
	if cfg.Provider == "" {
		return fmt.Errorf("提供商不能为空")
	}
	if cfg.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}

	// 规范化 extra_config 字段
	normalizedExtraConfig, err := normalizeExtraConfig(func() string {
		if cfg.ExtraConfig != nil {
			return *cfg.ExtraConfig
		}
		return ""
	}())
	if err != nil {
		return err
	}
	cfg.ExtraConfig = normalizedExtraConfig

	// 设置默认管理员（如果为空，设置为创建用户）
	if cfg.Admin == "" {
		cfg.Admin = user
	}

	// 先创建配置
	if err := s.repo.Create(cfg); err != nil {
		return err
	}

	// 如果设置为默认，设置默认配置
	if cfg.IsDefault {
		if err := s.repo.SetDefault(cfg.ID); err != nil {
			return fmt.Errorf("设置默认配置失败: %w", err)
		}
	}

	return nil
}

// UpdateLLMConfig 更新 LLM 配置
func (s *LLMService) UpdateLLMConfig(ctx context.Context, cfg *model.LLMConfig) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	cfg.UpdatedBy = user

	// 检查权限：只有管理员可以修改资源
	existing, err := s.repo.GetByID(cfg.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("LLM配置不存在")
		}
		return fmt.Errorf("获取LLM配置失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以修改此资源")
	}

	// 验证必填字段
	if cfg.Name == "" {
		return fmt.Errorf("配置名称不能为空")
	}
	if cfg.Provider == "" {
		return fmt.Errorf("提供商不能为空")
	}
	if cfg.Model == "" {
		return fmt.Errorf("模型名称不能为空")
	}

	// 规范化 extra_config 字段
	normalizedExtraConfig, err := normalizeExtraConfig(func() string {
		if cfg.ExtraConfig != nil {
			return *cfg.ExtraConfig
		}
		return ""
	}())
	if err != nil {
		return err
	}
	cfg.ExtraConfig = normalizedExtraConfig

	// 如果设置为默认，先取消其他默认配置
	if cfg.IsDefault {
		if err := s.repo.SetDefault(cfg.ID); err != nil {
			return fmt.Errorf("设置默认配置失败: %w", err)
		}
	}

	return s.repo.Update(cfg)
}

// DeleteLLMConfig 删除 LLM 配置
func (s *LLMService) DeleteLLMConfig(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以删除资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("LLM配置不存在")
		}
		return fmt.Errorf("获取LLM配置失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以删除此资源")
	}

	return s.repo.Delete(id)
}

// SetDefaultLLMConfig 设置默认 LLM 配置
func (s *LLMService) SetDefaultLLMConfig(ctx context.Context, id int64) error {
	return s.repo.SetDefault(id)
}

