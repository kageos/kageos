package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type serviceTreeMutationService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	docService       *DocService
	boardPostRepo    *repository.BoardPostRepository
}

func newServiceTreeMutationService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	docService *DocService,
	boardPostRepo *repository.BoardPostRepository,
) *serviceTreeMutationService {
	return &serviceTreeMutationService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
		docService:       docService,
		boardPostRepo:    boardPostRepo,
	}
}

func (m *serviceTreeMutationService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	if req == nil || req.ID <= 0 {
		return fmt.Errorf("服务目录ID不能为空")
	}

	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	oldAdmins := serviceTree.Admins

	if req.Name != nil {
		serviceTree.Name = *req.Name
	}
	if req.Code != nil {
		newCode := strings.TrimSpace(*req.Code)
		if newCode != serviceTree.Code {
			return fmt.Errorf("节点 code 暂不支持修改，请新建节点后迁移内容")
		}
	}
	if req.Description != nil {
		serviceTree.Description = *req.Description
	}
	if req.Tags != nil {
		serviceTree.Tags = *req.Tags
	}
	if req.Admins != nil {
		serviceTree.Admins = *req.Admins
	}

	if err := m.serviceTreeRepo.UpdateServiceTree(serviceTree); err != nil {
		return fmt.Errorf("failed to update service tree: %w", err)
	}

	if req.Admins != nil {
		m.syncDirectoryAdminRoles(ctx, serviceTree.FullCodePath, oldAdmins, serviceTree.Admins)
	}

	logger.Infof(ctx, "[ServiceTreeService] Updated service tree: ID=%d", req.ID)
	return nil
}

func (m *serviceTreeMutationService) UpdatePackage(ctx context.Context, req *dto.UpdatePackageReq) error {
	return m.UpdateServiceTreeMetadata(ctx, &dto.UpdateServiceTreeMetadataReq{
		ID:          req.ID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Tags:        req.Tags,
		Admins:      req.Admins,
	})
}

func (m *serviceTreeMutationService) UpdateFunction(ctx context.Context, req *dto.UpdateFunctionReq) error {
	return m.UpdateServiceTreeMetadata(ctx, &dto.UpdateServiceTreeMetadataReq{
		ID:          req.ID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Tags:        req.Tags,
	})
}

func (m *serviceTreeMutationService) UpdateDocs(ctx context.Context, req *dto.UpdateDocsReq) error {
	if err := m.UpdateServiceTreeMetadata(ctx, &dto.UpdateServiceTreeMetadataReq{
		ID:          req.ID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Tags:        req.Tags,
		Admins:      req.Admins,
	}); err != nil {
		return err
	}

	if req.Content == nil && req.Format == nil && req.Summary == nil {
		return nil
	}

	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("获取文档节点失败: %w", err)
	}
	if serviceTree.Type != model.ServiceTreeTypeDocs {
		return fmt.Errorf("节点类型不是 docs，当前类型: %s", serviceTree.Type)
	}

	return m.upsertDocContent(ctx, serviceTree, req)
}

func (m *serviceTreeMutationService) UpdateBoard(ctx context.Context, req *dto.UpdateBoardReq) error {
	if req == nil || req.ID <= 0 {
		return fmt.Errorf("版块ID不能为空")
	}

	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}
	if serviceTree.Type != model.ServiceTreeTypeBoard {
		return fmt.Errorf("节点类型不是 board，当前类型: %s", serviceTree.Type)
	}

	updateReq := &dto.UpdateServiceTreeMetadataReq{ID: req.ID}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}
	if req.Tags != "" {
		updateReq.Tags = &req.Tags
	}
	if req.Admins != "" {
		updateReq.Admins = &req.Admins
	}
	return m.UpdateServiceTreeMetadata(ctx, updateReq)
}

func (m *serviceTreeMutationService) UpdateWorkflowNode(ctx context.Context, req *dto.UpdateWorkflowNodeReq) error {
	if req == nil || req.ID <= 0 {
		return fmt.Errorf("工作流节点ID不能为空")
	}

	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}
	if serviceTree.Type != model.ServiceTreeTypeWorkflow {
		return fmt.Errorf("节点类型不是 workflow，当前类型: %s", serviceTree.Type)
	}

	updateReq := &dto.UpdateServiceTreeMetadataReq{ID: req.ID}
	if req.Name != "" {
		updateReq.Name = &req.Name
	}
	if req.Description != "" {
		updateReq.Description = &req.Description
	}
	if req.Tags != "" {
		updateReq.Tags = &req.Tags
	}
	if req.Admins != "" {
		updateReq.Admins = &req.Admins
	}
	return m.UpdateServiceTreeMetadata(ctx, updateReq)
}

func (m *serviceTreeMutationService) DeletePackage(ctx context.Context, id int64) error {
	return m.deleteTypedServiceTree(ctx, id, model.ServiceTreeTypePackage)
}

func (m *serviceTreeMutationService) DeleteFunction(ctx context.Context, id int64) error {
	return m.deleteTypedServiceTree(ctx, id, model.ServiceTreeTypeFunction)
}

func (m *serviceTreeMutationService) DeleteDocs(ctx context.Context, id int64) error {
	return m.deleteTypedServiceTree(ctx, id, model.ServiceTreeTypeDocs)
}

func (m *serviceTreeMutationService) DeleteBoard(ctx context.Context, id int64) error {
	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}
	if serviceTree.Type != model.ServiceTreeTypeBoard {
		return fmt.Errorf("节点类型不是 board，当前类型: %s", serviceTree.Type)
	}
	if err := m.boardPostRepo.DeleteByTreeID(id); err != nil {
		return fmt.Errorf("删除版块帖子失败: %w", err)
	}
	return m.DeleteServiceTree(ctx, id)
}

func (m *serviceTreeMutationService) DeleteWorkflowNode(ctx context.Context, id int64) error {
	return m.deleteTypedServiceTree(ctx, id, model.ServiceTreeTypeWorkflow)
}

func (m *serviceTreeMutationService) DeleteServiceTree(ctx context.Context, id int64) error {
	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	m.cleanupRuntimePackageScaffold(ctx, serviceTree)

	if err := m.serviceTreeRepo.DeleteServiceTree(id); err != nil {
		return fmt.Errorf("failed to delete service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Code=%s", id, serviceTree.Code)
	return nil
}

func (m *serviceTreeMutationService) deleteTypedServiceTree(ctx context.Context, id int64, expectedType string) error {
	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}
	if serviceTree.Type != expectedType {
		return fmt.Errorf("节点类型不是 %s，当前类型: %s", expectedType, serviceTree.Type)
	}
	return m.DeleteServiceTree(ctx, id)
}

func (m *serviceTreeMutationService) syncDirectoryAdminRoles(ctx context.Context, resourcePath, oldAdmins, newAdmins string) {
	user, app, err := parseUserAppFromResourcePath(resourcePath)
	if err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] 解析管理员资源路径失败: resource=%s, error=%v", resourcePath, err)
		return
	}

	oldAdminSet := parseAdminUserSet(oldAdmins)
	newAdminSet := parseAdminUserSet(newAdmins)

	for username := range oldAdminSet {
		if newAdminSet[username] {
			continue
		}
		if err := removeDirectoryAdminRoleFromUserWithUserApp(ctx, user, app, username, resourcePath); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 移除管理员角色失败: resource=%s, username=%s, error=%v",
				resourcePath, username, err)
		}
	}

	for username := range newAdminSet {
		if oldAdminSet[username] {
			continue
		}
		if err := assignDirectoryAdminRoleToUser(ctx, user, app, username, resourcePath); err != nil {
			logger.Warnf(ctx, "[ServiceTreeService] 分配管理员角色失败: resource=%s, username=%s, error=%v",
				resourcePath, username, err)
		}
	}
}

func (m *serviceTreeMutationService) upsertDocContent(ctx context.Context, serviceTree *model.ServiceTree, req *dto.UpdateDocsReq) error {
	_, err := m.docService.GetDoc(ctx, serviceTree.FullCodePath)
	if err != nil {
		if !strings.Contains(err.Error(), "不存在") {
			return fmt.Errorf("获取文档记录失败: %w", err)
		}

		createReq := &dto.CreateDocReq{
			FullCodePath: serviceTree.FullCodePath,
			Content:      "",
			Format:       "markdown",
			Summary:      "",
		}
		if req.Content != nil {
			createReq.Content = *req.Content
		}
		if req.Format != nil {
			createReq.Format = *req.Format
		}
		if req.Summary != nil {
			createReq.Summary = *req.Summary
		}

		if _, err := m.docService.CreateDoc(ctx, createReq); err != nil {
			return fmt.Errorf("创建文档内容失败: %w", err)
		}
		return nil
	}

	updateReq := &dto.UpdateDocReq{FullCodePath: serviceTree.FullCodePath}
	if req.Content != nil {
		updateReq.Content = *req.Content
	}
	if req.Format != nil {
		updateReq.Format = *req.Format
	}
	if req.Summary != nil {
		updateReq.Summary = *req.Summary
	}

	if _, err := m.docService.UpdateDoc(ctx, updateReq); err != nil {
		return fmt.Errorf("更新文档内容失败: %w", err)
	}
	return nil
}

func (m *serviceTreeMutationService) cleanupRuntimePackageScaffold(ctx context.Context, serviceTree *model.ServiceTree) {
	if serviceTree.Type != model.ServiceTreeTypePackage || serviceTree.IsRoot() || serviceTree.FullCodePath == "" {
		return
	}

	appModel, err := m.appRepo.GetAppByID(serviceTree.AppID)
	if err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] GetAppByID failed, skip runtime delete: %v", err)
		return
	}

	prefix := "/" + appModel.User + "/" + appModel.Code + "/"
	packagePath := strings.TrimPrefix(serviceTree.FullCodePath, prefix)
	packagePath = strings.Trim(packagePath, "/")
	if packagePath == "" {
		return
	}

	_, resp, err := m.runtimeWorkspace.deleteDirectoryScaffold(ctx, serviceTree.AppID, packagePath)
	if err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] DeleteServiceTree runtime failed: %v", err)
		return
	}
	if !resp.Success {
		logger.Warnf(ctx, "[ServiceTreeService] DeleteServiceTree runtime resp error: %s", resp.Error)
	}
}
