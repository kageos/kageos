package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type ServiceTreeService struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
	appRuntime      *AppRuntime
}

// NewServiceTreeService 创建服务目录服务
func NewServiceTreeService(serviceTreeRepo *repository.ServiceTreeRepository, appRepo *repository.AppRepository, appRuntime *AppRuntime) *ServiceTreeService {
	return &ServiceTreeService{
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
		appRuntime:      appRuntime,
	}
}

// CreateServiceTree 创建服务目录
func (s *ServiceTreeService) CreateServiceTree(ctx context.Context, req *dto.CreateServiceTreeReq) (*dto.CreateServiceTreeResp, error) {
	// 获取应用信息
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// 检查名称是否已存在
	exists, err := s.serviceTreeRepo.CheckNameExists(app.ID, req.ParentID, req.Name, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("service tree name '%s' already exists in this parent directory", req.Name)
	}

	// 创建服务目录记录
	serviceTree := &model.ServiceTree{
		Title:       req.Title,
		Name:        req.Name,
		ParentID:    req.ParentID,
		Type:        model.ServiceTreeTypePackage,
		Description: req.Description,
		Tags:        req.Tags,
		AppID:       app.ID,
	}

	// 保存到数据库（repository会自动计算路径）
	if err := s.serviceTreeRepo.CreateServiceTree(serviceTree); err != nil {
		return nil, fmt.Errorf("failed to create service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Created service tree: %s/%s/%s", req.User, req.App, req.Name)

	// 发送NATS消息给app-runtime创建目录结构
	if err := s.sendCreateServiceTreeMessage(ctx, req.User, req.App, serviceTree); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to send NATS message: %v", err)
		// 不返回错误，因为数据库记录已创建成功
	}

	// 返回响应
	resp := &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Title:        serviceTree.Title,
		Name:         serviceTree.Name,
		ParentID:     serviceTree.ParentID,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullIDPath:   serviceTree.FullIDPath,
		FullNamePath: serviceTree.FullNamePath,
		Status:       "created",
	}

	return resp, nil
}

// GetServiceTree 获取服务目录
func (s *ServiceTreeService) GetServiceTree(ctx context.Context, user, app string) ([]*dto.GetServiceTreeResp, error) {
	// 获取应用信息
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	// 构建树形结构
	trees, err := s.serviceTreeRepo.BuildServiceTree(appModel.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to build service tree: %w", err)
	}

	// 转换为响应格式
	var resp []*dto.GetServiceTreeResp
	for _, tree := range trees {
		resp = append(resp, s.convertToGetServiceTreeResp(tree))
	}

	return resp, nil
}

// UpdateServiceTree 更新服务目录
func (s *ServiceTreeService) UpdateServiceTree(ctx context.Context, req *dto.UpdateServiceTreeReq) error {
	// 获取服务目录
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(req.ID)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	// 更新字段
	if req.Title != "" {
		serviceTree.Title = req.Title
	}
	if req.Name != "" {
		// 检查新名称是否已存在
		exists, err := s.serviceTreeRepo.CheckNameExists(serviceTree.AppID, serviceTree.ParentID, req.Name, req.ID)
		if err != nil {
			return fmt.Errorf("failed to check name exists: %w", err)
		}
		if exists {
			return fmt.Errorf("service tree name '%s' already exists in this parent directory", req.Name)
		}
		serviceTree.Name = req.Name
	}
	if req.Description != "" {
		serviceTree.Description = req.Description
	}
	if req.Tags != "" {
		serviceTree.Tags = req.Tags
	}

	// 保存更新
	if err := s.serviceTreeRepo.UpdateServiceTree(serviceTree); err != nil {
		return fmt.Errorf("failed to update service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Updated service tree: ID=%d", req.ID)
	return nil
}

// DeleteServiceTree 删除服务目录
func (s *ServiceTreeService) DeleteServiceTree(ctx context.Context, id int64) error {
	// 获取服务目录信息（用于日志）
	serviceTree, err := s.serviceTreeRepo.GetServiceTreeByID(id)
	if err != nil {
		return fmt.Errorf("failed to get service tree: %w", err)
	}

	// 删除服务目录（级联删除子目录）
	if err := s.serviceTreeRepo.DeleteServiceTree(id); err != nil {
		return fmt.Errorf("failed to delete service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Name=%s", id, serviceTree.Name)
	return nil
}

// convertToGetServiceTreeResp 转换为响应格式
func (s *ServiceTreeService) convertToGetServiceTreeResp(tree *model.ServiceTree) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:           tree.ID,
		Title:        tree.Title,
		Name:         tree.Name,
		ParentID:     tree.ParentID,
		Type:         tree.Type,
		Description:  tree.Description,
		Tags:         tree.Tags,
		AppID:        tree.AppID,
		FullIDPath:   tree.FullIDPath,
		FullNamePath: tree.FullNamePath,
	}

	// 递归处理子节点
	if len(tree.Children) > 0 {
		for _, child := range tree.Children {
			resp.Children = append(resp.Children, s.convertToGetServiceTreeResp(child))
		}
	}

	return resp
}

// sendCreateServiceTreeMessage 发送创建服务目录的NATS消息
func (s *ServiceTreeService) sendCreateServiceTreeMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
	// 获取应用信息以获取HostID
	appModel, err := s.appRepo.GetAppByUserName(user, app)
	if err != nil {
		return fmt.Errorf("failed to get app info: %w", err)
	}

	// 构建消息
	req := dto.CreateServiceTreeRuntimeReq{
		User: user,
		App:  app,
		ServiceTree: &dto.ServiceTreeRuntimeData{
			ID:           serviceTree.ID,
			Title:        serviceTree.Title,
			Name:         serviceTree.Name,
			ParentID:     serviceTree.ParentID,
			Type:         serviceTree.Type,
			Description:  serviceTree.Description,
			Tags:         serviceTree.Tags,
			AppID:        serviceTree.AppID,
			FullIDPath:   serviceTree.FullIDPath,
			FullNamePath: serviceTree.FullNamePath,
		},
	}

	// 调用 app-runtime 创建服务目录，使用应用所属的 HostID
	resp, err := s.appRuntime.CreateServiceTree(ctx, appModel.HostID, &req)
	if err != nil {
		return fmt.Errorf("failed to create service tree via app-runtime: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully via app-runtime: %s/%s/%s, status: %s",
		user, app, serviceTree.Name, resp.Status)

	return nil
}
