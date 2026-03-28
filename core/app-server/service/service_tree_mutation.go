package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// UpdateServiceTree 更新服务目录
func (s *ServiceTreeService) UpdateServiceTreeMetadata(ctx context.Context, req *dto.UpdateServiceTreeMetadataReq) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	if req.Name != nil {
		serviceTree.Name = *req.Name
	}

	if req.Code != nil {
		newCode := *req.Code
		if newCode != serviceTree.Code && newCode != "" {
			renameParentPath := serviceTree.GetParentFullPath()
			exists, err := s.serviceTreeRepo.CheckNameExistsByPath(renameParentPath, newCode, serviceTree.AppID)
			if err != nil {
				return fmt.Errorf("failed to check name exists: %w", err)
			}
			if exists {
				return fmt.Errorf("service tree name '%s' already exists in this parent directory", newCode)
			}
		}
		serviceTree.Code = newCode
	}

	if req.Description != nil {
		serviceTree.Description = *req.Description
	}

	if req.Tags != nil {
		serviceTree.Tags = *req.Tags
	}

	oldAdminsStr := serviceTree.Admins
	newAdminsStr := oldAdminsStr
	if req.Admins != nil {
		newAdminsStr = *req.Admins
	}

	if req.Admins != nil {
		oldAdmins := make(map[string]bool)
		if oldAdminsStr != "" {
			for _, admin := range strings.Split(oldAdminsStr, ",") {
				admin = strings.TrimSpace(admin)
				if admin != "" {
					oldAdmins[admin] = true
				}
			}
		}

		newAdmins := make(map[string]bool)
		if newAdminsStr != "" {
			for _, admin := range strings.Split(newAdminsStr, ",") {
				admin = strings.TrimSpace(admin)
				if admin != "" {
					newAdmins[admin] = true
				}
			}
		}

		serviceTree.Admins = newAdminsStr
	}

	if err := s.serviceTreeRepo.UpdateServiceTree(serviceTree); err != nil {
		return fmt.Errorf("failed to update service tree: %w", err)
	}

	if req.Admins != nil {
		oldAdmins := make(map[string]bool)
		if oldAdminsStr != "" {
			for _, admin := range strings.Split(oldAdminsStr, ",") {
				admin = strings.TrimSpace(admin)
				if admin != "" {
					oldAdmins[admin] = true
				}
			}
		}

		newAdmins := make(map[string]bool)
		if newAdminsStr != "" {
			for _, admin := range strings.Split(newAdminsStr, ",") {
				admin = strings.TrimSpace(admin)
				if admin != "" {
					newAdmins[admin] = true
				}
			}
		}

		parts := strings.Split(strings.Trim(serviceTree.FullCodePath, "/"), "/")
		if len(parts) >= 2 {
			user := parts[0]
			app := parts[1]

			for oldAdmin := range oldAdmins {
				if !newAdmins[oldAdmin] {
					if err := s.removeAdminRoleFromUserWithUserApp(ctx, user, app, oldAdmin, serviceTree.FullCodePath); err != nil {
						logger.Warnf(ctx, "[ServiceTreeService] 移除管理员角色失败: resource=%s, username=%s, error=%v",
							serviceTree.FullCodePath, oldAdmin, err)
					}
				}
			}

			for newAdmin := range newAdmins {
				if !oldAdmins[newAdmin] {
					if err := s.assignAdminRoleToUser(ctx, user, app, newAdmin, serviceTree.FullCodePath); err != nil {
						logger.Warnf(ctx, "[ServiceTreeService] 分配管理员角色失败: resource=%s, username=%s, error=%v",
							serviceTree.FullCodePath, newAdmin, err)
					}
				}
			}
		}
	}

	logger.Infof(ctx, "[ServiceTreeService] Updated service tree: ID=%d", req.ID)
	return nil
}

// UpdatePackage 更新 package 类型节点（专门的接口）
func (s *ServiceTreeService) UpdatePackage(ctx context.Context, req *dto.UpdatePackageReq) error {
	updateReq := &dto.UpdateServiceTreeMetadataReq{
		ID:          req.ID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Tags:        req.Tags,
		Admins:      req.Admins,
	}

	return s.UpdateServiceTreeMetadata(ctx, updateReq)
}

// UpdateFunction 更新 function 类型节点（专门的接口）
func (s *ServiceTreeService) UpdateFunction(ctx context.Context, req *dto.UpdateFunctionReq) error {
	updateReq := &dto.UpdateServiceTreeMetadataReq{
		ID:          req.ID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Tags:        req.Tags,
	}

	return s.UpdateServiceTreeMetadata(ctx, updateReq)
}

// UpdateDocs 更新 docs 类型节点（专门的接口）
func (s *ServiceTreeService) UpdateDocs(ctx context.Context, req *dto.UpdateDocsReq) error {
	updateReq := &dto.UpdateServiceTreeMetadataReq{
		ID:          req.ID,
		Name:        req.Name,
		Code:        req.Code,
		Description: req.Description,
		Tags:        req.Tags,
		Admins:      req.Admins,
	}

	err := s.UpdateServiceTreeMetadata(ctx, updateReq)
	if err != nil {
		return err
	}

	if req.Content != nil || req.Format != nil || req.Summary != nil {
		serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(req.ID)
		if err != nil {
			return fmt.Errorf("获取文档节点失败: %w", err)
		}

		if serviceTree.Type != model.ServiceTreeTypeDocs {
			return fmt.Errorf("节点类型不是 docs，当前类型: %s", serviceTree.Type)
		}

		_, err = s.docService.GetDoc(ctx, serviceTree.FullCodePath)
		if err != nil {
			if strings.Contains(err.Error(), "不存在") {
				docReq := &dto.CreateDocReq{
					FullCodePath: serviceTree.FullCodePath,
					Content:      "",
					Format:       "markdown",
					Summary:      "",
				}
				if req.Content != nil {
					docReq.Content = *req.Content
				}
				if req.Format != nil {
					docReq.Format = *req.Format
				}
				if req.Summary != nil {
					docReq.Summary = *req.Summary
				}

				_, err = s.docService.CreateDoc(ctx, docReq)
				if err != nil {
					return fmt.Errorf("创建文档内容失败: %w", err)
				}
			} else {
				return fmt.Errorf("获取文档记录失败: %w", err)
			}
		} else {
			updateDocReq := &dto.UpdateDocReq{
				FullCodePath: serviceTree.FullCodePath,
			}
			if req.Content != nil {
				updateDocReq.Content = *req.Content
			}
			if req.Format != nil {
				updateDocReq.Format = *req.Format
			}
			if req.Summary != nil {
				updateDocReq.Summary = *req.Summary
			}

			_, err = s.docService.UpdateDoc(ctx, updateDocReq)
			if err != nil {
				return fmt.Errorf("更新文档内容失败: %w", err)
			}
		}
	}

	return nil
}

// DeletePackage 删除 package 类型节点（专门的接口）
func (s *ServiceTreeService) DeletePackage(ctx context.Context, id int64) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}

	if serviceTree.Type != model.ServiceTreeTypePackage {
		return fmt.Errorf("节点类型不是 package，当前类型: %s", serviceTree.Type)
	}

	return s.DeleteServiceTree(ctx, id)
}

// DeleteFunction 删除 function 类型节点（专门的接口）
func (s *ServiceTreeService) DeleteFunction(ctx context.Context, id int64) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}

	if serviceTree.Type != model.ServiceTreeTypeFunction {
		return fmt.Errorf("节点类型不是 function，当前类型: %s", serviceTree.Type)
	}

	return s.DeleteServiceTree(ctx, id)
}

// DeleteDocs 删除 docs 类型节点（专门的接口）
func (s *ServiceTreeService) DeleteDocs(ctx context.Context, id int64) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}

	if serviceTree.Type != model.ServiceTreeTypeDocs {
		return fmt.Errorf("节点类型不是 docs，当前类型: %s", serviceTree.Type)
	}

	return s.DeleteServiceTree(ctx, id)
}

// UpdateBoard 更新版块节点（名称、描述、标签、管理员）
func (s *ServiceTreeService) UpdateBoard(ctx context.Context, req *dto.UpdateBoardReq) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(req.ID)
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
	return s.UpdateServiceTreeMetadata(ctx, updateReq)
}

// DeleteBoard 删除版块节点（先删该版块下全部帖子，再删节点）
func (s *ServiceTreeService) DeleteBoard(ctx context.Context, id int64) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("获取节点失败: %w", err)
	}
	if serviceTree.Type != model.ServiceTreeTypeBoard {
		return fmt.Errorf("节点类型不是 board，当前类型: %s", serviceTree.Type)
	}
	if err := s.boardPostRepo.DeleteByTreeID(id); err != nil {
		return fmt.Errorf("删除版块帖子失败: %w", err)
	}
	return s.DeleteServiceTree(ctx, id)
}

// DeleteServiceTree 删除服务目录（先调 app-runtime 删磁盘目录并从 main.go 移除 import，再删 DB）
func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, id int64) error {
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	if serviceTree.Type == model.ServiceTreeTypePackage && !serviceTree.IsRoot() && serviceTree.FullCodePath != "" {
		appModel, errApp := s.appRepo.GetAppByID(serviceTree.AppID)
		if errApp != nil {
			logger.Warnf(ctx, "[ServiceTreeService] GetAppByID failed, skip runtime delete: %v", errApp)
		} else if appModel.HostID > 0 {
			prefix := "/" + appModel.User + "/" + appModel.Code + "/"
			packagePath := strings.TrimPrefix(serviceTree.FullCodePath, prefix)
			packagePath = strings.Trim(packagePath, "/")
			if packagePath != "" {
				req := &dto.DeleteServiceTreeRuntimeReq{
					User:        appModel.User,
					App:         appModel.Code,
					PackagePath: packagePath,
				}
				resp, errRt := s.appCall.DeleteServiceTree(ctx, appModel.HostID, req)
				if errRt != nil {
					logger.Warnf(ctx, "[ServiceTreeService] DeleteServiceTree runtime failed: %v", errRt)
				} else if !resp.Success {
					logger.Warnf(ctx, "[ServiceTreeService] DeleteServiceTree runtime resp error: %s", resp.Error)
				}
			}
		}
	}

	if err := s.serviceTreeRepo.DeleteServiceTree(id); err != nil {
		return fmt.Errorf("failed to delete service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Code=%s", id, serviceTree.Code)
	return nil
}

// sendCreateServiceTreeMessage 发送创建服务目录的NATS消息
func (s *ServiceTreeService) sendCreateServiceTreeMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return fmt.Errorf("failed to get app info: %w", err)
	}

	req := dto.CreateServiceTreeRuntimeReq{
		User: user,
		App:  app,
		ServiceTree: &dto.ServiceTreeRuntimeData{
			ID:           serviceTree.ID,
			Name:         serviceTree.Name,
			Code:         serviceTree.Code,
			Type:         serviceTree.Type,
			Description:  serviceTree.Description,
			Tags:         serviceTree.Tags,
			AppID:        serviceTree.AppID,
			FullCodePath: serviceTree.FullCodePath,
		},
	}

	_, err = s.appCall.CreateServiceTree(ctx, appModel.HostID, &req)
	if err != nil {
		return fmt.Errorf("failed to create service tree via app-runtime: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)

	return nil
}
