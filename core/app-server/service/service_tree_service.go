package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// extractVersionNum 从版本号字符串中提取数字部分（如 "v1" -> 1, "v20" -> 20）
func extractVersionNumForServiceTree(version string) int {
	if version == "" {
		return 0
	}
	// 去掉 "v" 前缀
	version = strings.TrimPrefix(version, "v")
	version = strings.TrimPrefix(version, "V")
	// 提取数字部分
	num, err := strconv.Atoi(version)
	if err != nil {
		return 0
	}
	return num
}

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

	var parentTree *model.ServiceTree

	if req.ParentID != 0 {
		// 检查名称是否已存在
		parentTree, err = s.serviceTreeRepo.GetByID(req.ParentID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("failed to check name exists: %s", err)
		}
	}

	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
	}
	exists, err := s.serviceTreeRepo.CheckNameExists(req.ParentID, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("directory %s already exists", req.Code)
	}

	// 提取当前版本号数字
	currentVersionNum := extractVersionNumForServiceTree(app.Version)

	// 创建服务目录记录
	serviceTree := &model.ServiceTree{
		Name:            req.Name,
		Code:            req.Code,
		ParentID:        req.ParentID,
		Type:            model.ServiceTreeTypePackage,
		Description:     req.Description,
		Tags:            req.Tags,
		AppID:           app.ID,
		FullCodePath:    fullCodePath,
		AddVersionNum:   currentVersionNum, // 设置添加版本号
		UpdateVersionNum: 0,               // 新增节点，更新版本号为0
	}

	// 保存到数据库
	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create service tree: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Created service tree: %s/%s/%s", req.User, req.App, req.Code)

	// 发送NATS消息给app-runtime创建目录结构
	if err := s.sendCreateServiceTreeMessage(ctx, req.User, req.App, serviceTree); err != nil {
		logger.Warnf(ctx, "[ServiceTreeService] Failed to send NATS message: %v", err)
		// 不返回错误，因为数据库记录已创建成功
	}

	// 返回响应
	resp := &dto.CreateServiceTreeResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		ParentID:     serviceTree.ParentID,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullCodePath: serviceTree.FullCodePath,
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
	if req.Name != "" {
		serviceTree.Name = req.Name
	}
	if req.Code != "" {
		// 检查新名称是否已存在
		exists, err := s.serviceTreeRepo.CheckNameExists(serviceTree.ParentID, req.Code, serviceTree.AppID)
		if err != nil {
			return fmt.Errorf("failed to check name exists: %w", err)
		}
		if exists {
			return fmt.Errorf("service tree name '%s' already exists in this parent directory", req.Code)
		}
		serviceTree.Code = req.Code
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

	logger.Infof(ctx, "[ServiceTreeService] Deleted service tree: ID=%d, Code=%s", id, serviceTree.Code)
	return nil
}

// convertToGetServiceTreeResp 转换为响应格式
func (s *ServiceTreeService) convertToGetServiceTreeResp(tree *model.ServiceTree) *dto.GetServiceTreeResp {
	resp := &dto.GetServiceTreeResp{
		ID:           tree.ID,
		Name:         tree.Name,
		Code:          tree.Code,
		ParentID:      tree.ParentID,
		RefID:         tree.RefID,
		Type:          tree.Type,
		FullGroupCode: tree.FullGroupCode,
		GroupName:     tree.GroupName,
		Description:  tree.Description,
		Tags:         tree.Tags,
		AppID:        tree.AppID,
		FullCodePath: tree.FullCodePath,
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
			Name:         serviceTree.Name,
			Code:         serviceTree.Code,
			ParentID:     serviceTree.ParentID,
			Type:         serviceTree.Type,
			Description:  serviceTree.Description,
			Tags:         serviceTree.Tags,
			AppID:        serviceTree.AppID,
			FullCodePath: serviceTree.FullCodePath,
		},
	}

	// 调用 app-runtime 创建服务目录，使用应用所属的 HostID
	_, err = s.appRuntime.CreateServiceTree(ctx, appModel.HostID, &req)
	if err != nil {
		return fmt.Errorf("failed to create service tree via app-runtime: %w", err)
	}

	logger.Infof(ctx, "[ServiceTreeService] Service tree created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)

	return nil
}
