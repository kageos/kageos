package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/logger"
)

type serviceTreeMutationService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
	docService       *DocService
}

func newServiceTreeMutationService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
	docService *DocService,
) *serviceTreeMutationService {
	return &serviceTreeMutationService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
		docService:       docService,
	}
}

func (m *serviceTreeMutationService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	if req == nil || req.ID <= 0 {
		return fmt.Errorf("服务目录ID不能为空")
	}

	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

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

	if err := m.serviceTreeRepo.UpdateServiceTree(ctx, serviceTree); err != nil {
		return fmt.Errorf("failed to update service tree: %w", err)
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

	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(ctx, req.ID)
	if err != nil {
		return fmt.Errorf("获取文档节点失败: %w", err)
	}
	if serviceTree.Type != model.ServiceTreeTypeDocs {
		return fmt.Errorf("节点类型不是 docs，当前类型: %s", serviceTree.Type)
	}

	return m.upsertDocContent(ctx, serviceTree, req)
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

func (m *serviceTreeMutationService) DeleteServiceTree(ctx context.Context, id int64) error {
	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	m.cleanupRuntimePackageScaffold(ctx, serviceTree)

	if err := m.serviceTreeRepo.DeleteServiceTree(ctx, id); err != nil {
		return fmt.Errorf("failed to delete service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Code=%s", id, serviceTree.Code)
	return nil
}

func (m *serviceTreeMutationService) deleteTypedServiceTree(ctx context.Context, id int64, expectedType string) error {
	serviceTree, err := m.serviceTreeRepo.GetServiceTreeByID(ctx, id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}
	if serviceTree.Type != expectedType {
		return fmt.Errorf("节点类型不是 %s，当前类型: %s", expectedType, serviceTree.Type)
	}
	return m.DeleteServiceTree(ctx, id)
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

	appModel, err := m.appRepo.GetAppByID(ctx, serviceTree.AppID)
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
