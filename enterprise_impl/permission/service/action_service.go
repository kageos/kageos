package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	permissionrepo "github.com/ai-agent-os/ai-agent-os/enterprise_impl/permission/repository"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	permissionpkg "github.com/ai-agent-os/ai-agent-os/pkg/permission"
)

// ActionService 权限点服务
type ActionService struct {
	actionRepo *permissionrepo.ActionRepository
}

// NewActionService 创建权限点服务
func NewActionService(actionRepo *permissionrepo.ActionRepository) *ActionService {
	return &ActionService{
		actionRepo: actionRepo,
	}
}

// InitDefaultActions 初始化默认权限点
func (s *ActionService) InitDefaultActions(ctx context.Context) error {
	// 检查是否已初始化
	count, err := s.actionRepo.CountActions(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		logger.Infof(ctx, "[ActionService] 预设权限点已存在，跳过初始化")
		return nil
	}

	// ⭐ 预设权限点配置（格式：resource_type:action_type）
	defaultActions := []struct {
		code         string
		name         string
		resourceType string
		actionType   string
		description  string
	}{
		// Directory 权限点
		{code: "directory:read", name: "目录查看", resourceType: permissionpkg.ResourceTypeDirectory, actionType: "read", description: "查看目录的权限"},
		{code: "directory:write", name: "目录写入", resourceType: permissionpkg.ResourceTypeDirectory, actionType: "write", description: "创建子目录和函数的权限"},
		{code: "directory:update", name: "目录更新", resourceType: permissionpkg.ResourceTypeDirectory, actionType: "update", description: "修改目录信息的权限"},
		{code: "directory:delete", name: "目录删除", resourceType: permissionpkg.ResourceTypeDirectory, actionType: "delete", description: "删除目录的权限"},
		{code: "directory:admin", name: "目录管理员", resourceType: permissionpkg.ResourceTypeDirectory, actionType: "admin", description: "目录管理员权限（拥有所有目录权限）"},
		// Table 权限点
		{code: "table:read", name: "表格查看", resourceType: permissionpkg.ResourceTypeTable, actionType: "read", description: "查看表格的权限"},
		{code: "table:write", name: "表格写入", resourceType: permissionpkg.ResourceTypeTable, actionType: "write", description: "创建表格记录的权限"},
		{code: "table:update", name: "表格更新", resourceType: permissionpkg.ResourceTypeTable, actionType: "update", description: "更新表格记录的权限"},
		{code: "table:delete", name: "表格删除", resourceType: permissionpkg.ResourceTypeTable, actionType: "delete", description: "删除表格记录的权限"},
		{code: "table:admin", name: "表格管理员", resourceType: permissionpkg.ResourceTypeTable, actionType: "admin", description: "表格管理员权限（拥有所有表格权限）"},
		// Form 权限点
		{code: "form:read", name: "表单查看", resourceType: permissionpkg.ResourceTypeForm, actionType: "read", description: "查看表单的权限"},
		{code: "form:write", name: "表单写入", resourceType: permissionpkg.ResourceTypeForm, actionType: "write", description: "提交表单的权限"},
		{code: "form:admin", name: "表单管理员", resourceType: permissionpkg.ResourceTypeForm, actionType: "admin", description: "表单管理员权限（拥有所有表单权限）"},
		// Chart 权限点
		{code: "chart:read", name: "图表查看", resourceType: permissionpkg.ResourceTypeChart, actionType: "read", description: "查看图表的权限"},
		{code: "chart:admin", name: "图表管理员", resourceType: permissionpkg.ResourceTypeChart, actionType: "admin", description: "图表管理员权限（拥有所有图表权限）"},
		// App 权限点
		{code: "app:read", name: "工作空间查看", resourceType: permissionpkg.ResourceTypeApp, actionType: "read", description: "查看工作空间的权限"},
		{code: "app:write", name: "工作空间创建", resourceType: permissionpkg.ResourceTypeApp, actionType: "write", description: "创建工作空间的权限"},
		{code: "app:update", name: "工作空间更新", resourceType: permissionpkg.ResourceTypeApp, actionType: "update", description: "更新工作空间的权限"},
		{code: "app:delete", name: "工作空间删除", resourceType: permissionpkg.ResourceTypeApp, actionType: "delete", description: "删除工作空间的权限"},
		{code: "app:admin", name: "工作空间管理员", resourceType: permissionpkg.ResourceTypeApp, actionType: "admin", description: "工作空间管理员权限（拥有所有工作空间权限）"},
	}

	// 创建预设权限点
	for _, actionConfig := range defaultActions {
		action := &model.Action{
			Code:         actionConfig.code,
			Name:         actionConfig.name,
			ResourceType: actionConfig.resourceType,
			ActionType:   actionConfig.actionType,
			Description:  actionConfig.description,
			IsSystem:     true, // 系统预设权限点
			CreatedBy:    "system",
		}

		if err := s.actionRepo.CreateAction(ctx, action); err != nil {
			return fmt.Errorf("创建预设权限点失败: code=%s, %w", actionConfig.code, err)
		}

		logger.Infof(ctx, "[ActionService] 创建预设权限点成功: code=%s, name=%s", action.Code, action.Name)
	}

	logger.Infof(ctx, "[ActionService] 预设权限点初始化完成，共 %d 个权限点", len(defaultActions))
	return nil
}
