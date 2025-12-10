package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/utils"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"gorm.io/gorm"
)

// PluginService 插件服务
type PluginService struct {
	repo *repository.PluginRepository
}

// NewPluginService 创建插件服务
func NewPluginService(repo *repository.PluginRepository) *PluginService {
	return &PluginService{
		repo: repo,
	}
}

// CreatePlugin 创建插件
func (s *PluginService) CreatePlugin(ctx context.Context, plugin *model.Plugin) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	plugin.CreatedBy = user
	plugin.UpdatedBy = user
	plugin.User = user

	// 验证必填字段
	if plugin.Name == "" {
		return fmt.Errorf("插件名称不能为空")
	}
	if plugin.Code == "" {
		return fmt.Errorf("插件代码不能为空")
	}

	// 检查 Code 是否已存在
	existing, err := s.repo.GetByCode(plugin.Code)
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查插件代码失败: %w", err)
	}
	if existing != nil {
		return fmt.Errorf("插件代码已存在: %s", plugin.Code)
	}

	// 规范化 config 字段
	configStr := ""
	if plugin.Config != nil {
		configStr = *plugin.Config
	}
	normalizedConfig, err := normalizeMetadata(configStr)
	if err != nil {
		return err
	}
	plugin.Config = normalizedConfig

	// 设置默认管理员（如果为空，设置为创建用户）
	if plugin.Admin == "" {
		plugin.Admin = user
	}

	// 创建插件（AfterCreate 钩子会自动生成 NATS 主题）
	err = s.repo.Create(plugin)
	if err != nil {
		return err
	}

	// 如果创建成功，确保主题已生成（AfterCreate 钩子应该已经处理，这里作为兜底）
	if plugin.Subject == "" {
		plugin.Subject = subjects.BuildPluginSubject(plugin.CreatedBy, plugin.ID)
		return s.repo.Update(plugin)
	}

	return nil
}

// UpdatePlugin 更新插件
func (s *PluginService) UpdatePlugin(ctx context.Context, plugin *model.Plugin) error {
	// 获取用户信息
	user := contextx.GetRequestUser(ctx)
	plugin.UpdatedBy = user

	// 检查权限：只有管理员可以修改资源
	existing, err := s.repo.GetByID(plugin.ID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("插件不存在")
		}
		return fmt.Errorf("获取插件失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以修改此资源")
	}

	// 验证必填字段
	if plugin.Name == "" {
		return fmt.Errorf("插件名称不能为空")
	}
	if plugin.Code == "" {
		return fmt.Errorf("插件代码不能为空")
	}

	// 检查 Code 是否与其他插件冲突（排除自己）
	existing, err = s.repo.GetByCode(plugin.Code)
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查插件代码失败: %w", err)
	}
	if existing != nil && existing.ID != plugin.ID {
		return fmt.Errorf("插件代码已被其他插件使用: %s", plugin.Code)
	}

	// 规范化 config 字段
	configStr := ""
	if plugin.Config != nil {
		configStr = *plugin.Config
	}
	normalizedConfig, err := normalizeMetadata(configStr)
	if err != nil {
		return err
	}
	plugin.Config = normalizedConfig

	// 如果主题为空，自动生成（不应该发生，但作为兜底）
	if plugin.Subject == "" {
		plugin.Subject = subjects.BuildPluginSubject(plugin.CreatedBy, plugin.ID)
	}

	return s.repo.Update(plugin)
}

// GetPlugin 获取插件
func (s *PluginService) GetPlugin(ctx context.Context, id int64) (*model.Plugin, error) {
	return s.repo.GetByID(id)
}

// ListPlugins 获取插件列表
func (s *PluginService) ListPlugins(ctx context.Context, scope string, enabled *bool, page, pageSize int) ([]*model.Plugin, int64, error) {
	currentUser := contextx.GetRequestUser(ctx)
	offset := (page - 1) * pageSize
	return s.repo.List(scope, currentUser, enabled, offset, pageSize)
}

// DeletePlugin 删除插件
func (s *PluginService) DeletePlugin(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以删除资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("插件不存在")
		}
		return fmt.Errorf("获取插件失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以删除此资源")
	}

	return s.repo.Delete(id)
}

// EnablePlugin 启用插件
func (s *PluginService) EnablePlugin(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以启用资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("插件不存在")
		}
		return fmt.Errorf("获取插件失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以启用此资源")
	}

	return s.repo.Enable(id)
}

// DisablePlugin 禁用插件
func (s *PluginService) DisablePlugin(ctx context.Context, id int64) error {
	// 检查权限：只有管理员可以禁用资源
	user := contextx.GetRequestUser(ctx)
	existing, err := s.repo.GetByID(id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("插件不存在")
		}
		return fmt.Errorf("获取插件失败: %w", err)
	}

	if !utils.IsAdmin(existing.Admin, user) {
		return fmt.Errorf("无权限：只有管理员可以禁用此资源")
	}

	return s.repo.Disable(id)
}
