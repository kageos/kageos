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

// InitDefaultActions 初始化默认权限点（支持增量添加）
// ⭐ 改为增量添加模式：检查每个权限点是否存在，不存在则创建
func (s *ActionService) InitDefaultActions(ctx context.Context) error {
	// ⭐ 预设权限点配置（格式：resource_type:action_type）
	defaultActions := []struct {
		name         string
		resourceType string
		actionType   string
		description  string
	}{
		// Directory 权限点
		{name: "目录查看", resourceType: permissionpkg.ResourceTypeDirectory, actionType: permissionpkg.ActionRead, description: "查看目录的权限"},
		{name: "目录写入", resourceType: permissionpkg.ResourceTypeDirectory, actionType: permissionpkg.ActionWrite, description: "创建子目录和函数的权限"},
		{name: "目录更新", resourceType: permissionpkg.ResourceTypeDirectory, actionType: permissionpkg.ActionUpdate, description: "修改目录信息的权限"},
		{name: "目录删除", resourceType: permissionpkg.ResourceTypeDirectory, actionType: permissionpkg.ActionDelete, description: "删除目录的权限"},
		{name: "目录管理员", resourceType: permissionpkg.ResourceTypeDirectory, actionType: permissionpkg.ActionAdmin, description: "目录管理员权限（拥有所有目录权限）"},
		// Table 权限点
		{name: "表格查看", resourceType: permissionpkg.ResourceTypeTable, actionType: permissionpkg.ActionRead, description: "查看表格的权限"},
		{name: "表格写入", resourceType: permissionpkg.ResourceTypeTable, actionType: permissionpkg.ActionWrite, description: "创建表格记录的权限"},
		{name: "表格更新", resourceType: permissionpkg.ResourceTypeTable, actionType: permissionpkg.ActionUpdate, description: "更新表格记录的权限"},
		{name: "表格删除", resourceType: permissionpkg.ResourceTypeTable, actionType: permissionpkg.ActionDelete, description: "删除表格记录的权限"},
		{name: "表格管理员", resourceType: permissionpkg.ResourceTypeTable, actionType: permissionpkg.ActionAdmin, description: "表格管理员权限（拥有所有表格权限）"},
		// Form 权限点
		{name: "表单查看", resourceType: permissionpkg.ResourceTypeForm, actionType: permissionpkg.ActionRead, description: "查看表单的权限"},
		{name: "表单写入", resourceType: permissionpkg.ResourceTypeForm, actionType: permissionpkg.ActionWrite, description: "提交表单的权限"},
		{name: "表单管理员", resourceType: permissionpkg.ResourceTypeForm, actionType: permissionpkg.ActionAdmin, description: "表单管理员权限（拥有所有表单权限）"},
		// Chart 权限点
		{name: "图表查看", resourceType: permissionpkg.ResourceTypeChart, actionType: permissionpkg.ActionRead, description: "查看图表的权限"},
		{name: "图表管理员", resourceType: permissionpkg.ResourceTypeChart, actionType: permissionpkg.ActionAdmin, description: "图表管理员权限（拥有所有图表权限）"},
		// App 权限点
		{name: "工作空间查看", resourceType: permissionpkg.ResourceTypeApp, actionType: permissionpkg.ActionRead, description: "查看工作空间的权限"},
		{name: "工作空间创建", resourceType: permissionpkg.ResourceTypeApp, actionType: permissionpkg.ActionWrite, description: "创建工作空间的权限"},
		{name: "工作空间更新", resourceType: permissionpkg.ResourceTypeApp, actionType: permissionpkg.ActionUpdate, description: "更新工作空间的权限"},
		{name: "工作空间删除", resourceType: permissionpkg.ResourceTypeApp, actionType: permissionpkg.ActionDelete, description: "删除工作空间的权限"},
		{name: "工作空间管理员", resourceType: permissionpkg.ResourceTypeApp, actionType: permissionpkg.ActionAdmin, description: "工作空间管理员权限（拥有所有工作空间权限）"},
		// Docs 权限点（文档）
		{name: "文档查看", resourceType: permissionpkg.ResourceTypeDocs, actionType: permissionpkg.ActionRead, description: "查看文档的权限"},
		{name: "文档编辑", resourceType: permissionpkg.ResourceTypeDocs, actionType: permissionpkg.ActionWrite, description: "编辑文档的权限"},
		{name: "文档删除", resourceType: permissionpkg.ResourceTypeDocs, actionType: permissionpkg.ActionDelete, description: "删除文档的权限"},
		{name: "文档管理员", resourceType: permissionpkg.ResourceTypeDocs, actionType: permissionpkg.ActionAdmin, description: "文档管理员权限（拥有所有文档权限）"},
		// Board 权限点（讨论区/板块）
		{name: "帖子查看", resourceType: permissionpkg.ResourceTypeBoard, actionType: permissionpkg.ActionRead, description: "查看讨论区帖子的权限"},
		{name: "发帖", resourceType: permissionpkg.ResourceTypeBoard, actionType: permissionpkg.ActionWrite, description: "在讨论区发帖的权限"},
		{name: "帖子更新", resourceType: permissionpkg.ResourceTypeBoard, actionType: permissionpkg.ActionUpdate, description: "更新帖子的权限"},
		{name: "帖子删除", resourceType: permissionpkg.ResourceTypeBoard, actionType: permissionpkg.ActionDelete, description: "删除帖子的权限"},
		{name: "板块管理员", resourceType: permissionpkg.ResourceTypeBoard, actionType: permissionpkg.ActionAdmin, description: "板块管理员权限（拥有所有讨论区权限）"},
	}

	// ⭐ 增量添加预设权限点（检查每个权限点是否存在）
	createdCount := 0
	skippedCount := 0

	for _, actionConfig := range defaultActions {
		actionCode := permissionpkg.BuildActionCode(actionConfig.resourceType, actionConfig.actionType)

		// 检查权限点是否已存在
		existingAction, err := s.actionRepo.GetActionByCode(ctx, actionCode)
		if err == nil && existingAction != nil {
			// 权限点已存在，跳过
			skippedCount++
			logger.Debugf(ctx, "[ActionService] 权限点已存在，跳过创建: code=%s", actionCode)
			continue
		}

		// 权限点不存在，创建新权限点
		action := &model.Action{
			Code:         actionCode,
			Name:         actionConfig.name,
			ResourceType: actionConfig.resourceType,
			ActionType:   actionConfig.actionType,
			Description:  actionConfig.description,
			IsSystem:     true, // 系统预设权限点
			CreatedBy:    "system",
		}

		if err := s.actionRepo.CreateAction(ctx, action); err != nil {
			return fmt.Errorf("创建预设权限点失败: code=%s, %w", actionCode, err)
		}

		createdCount++
		logger.Infof(ctx, "[ActionService] 创建预设权限点成功: code=%s, name=%s", action.Code, action.Name)
	}

	logger.Infof(ctx, "[ActionService] 预设权限点初始化完成，共 %d 个权限点，创建 %d 个，跳过 %d 个",
		len(defaultActions), createdCount, skippedCount)
	return nil
}
