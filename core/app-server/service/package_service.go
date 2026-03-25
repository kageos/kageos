package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/enterprise"
	"github.com/ai-agent-os/ai-agent-os/pkg/appcall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/license"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type PackageService struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
	appCall         *appcall.Client
}

func NewPackageService(serviceTreeRepo *repository.ServiceTreeRepository, appRepo *repository.AppRepository, appCall *appcall.Client) *PackageService {
	return &PackageService{
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
		appCall:         appCall,
	}
}

func (s *PackageService) CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
	app, err := s.appRepo.GetAppByUserName(req.User, req.App)
	if err != nil {
		return nil, fmt.Errorf("failed to get app: %w", err)
	}

	var parentTree *model.ServiceTree
	if req.ParentFullCodePath != "" {
		parentTree, err = s.serviceTreeRepo.GetServiceTreeByFullPath(req.ParentFullCodePath)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent node: %w", err)
		}
	}

	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
	}

	parentPath := ""
	if parentTree != nil {
		parentPath = parentTree.FullCodePath
	}

	exists, err := s.serviceTreeRepo.CheckNameExistsByPath(parentPath, req.Code, app.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check name exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("directory %s already exists", req.Code)
	}

	currentVersionNum := extractVersionNumForServiceTree(app.Version)
	requestUser := contextx.GetRequestUser(ctx)
	serviceTree := &model.ServiceTree{
		Name:             req.Name,
		Code:             req.Code,
		Type:             model.ServiceTreeTypePackage,
		Description:      req.Description,
		Tags:             req.Tags,
		Admins:           req.Admins,
		AppID:            app.ID,
		FullCodePath:     fullCodePath,
		AddVersionNum:    currentVersionNum,
		UpdateVersionNum: 0,
	}
	if requestUser != "" {
		serviceTree.CreatedBy = requestUser
	}

	if err := s.serviceTreeRepo.CreateServiceTreeWithParentPath(serviceTree, ""); err != nil {
		return nil, fmt.Errorf("failed to create service tree: %w", err)
	}

	logger.Infof(ctx, "[PackageService] Created package node: %s/%s/%s", req.User, req.App, req.Code)

	if requestUser != "" {
		if err := assignDirectoryAdminRoleToUser(ctx, req.User, req.App, requestUser, serviceTree.FullCodePath); err != nil {
			logger.Warnf(ctx, "[PackageService] 自动添加创建者管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
				req.User, req.App, requestUser, serviceTree.FullCodePath, err)
		}
	}

	if req.Admins != "" {
		admins := strings.Split(req.Admins, ",")
		for _, admin := range admins {
			admin = strings.TrimSpace(admin)
			if admin != "" && admin != requestUser {
				if err := assignDirectoryAdminRoleToUser(ctx, req.User, req.App, admin, serviceTree.FullCodePath); err != nil {
					logger.Warnf(ctx, "[PackageService] 自动添加管理员角色失败: user=%s, app=%s, username=%s, resource=%s, error=%v",
						req.User, req.App, admin, serviceTree.FullCodePath, err)
				}
			}
		}
	}

	if err := s.sendCreatePackageMessage(ctx, req.User, req.App, serviceTree); err != nil {
		logger.Warnf(ctx, "[PackageService] Failed to send NATS message: %v", err)
	}

	return &dto.CreatePackageResp{
		ID:           serviceTree.ID,
		Name:         serviceTree.Name,
		Code:         serviceTree.Code,
		Type:         serviceTree.Type,
		Description:  serviceTree.Description,
		Tags:         serviceTree.Tags,
		AppID:        serviceTree.AppID,
		FullCodePath: serviceTree.FullCodePath,
		Version:      serviceTree.Version,
		VersionNum:   serviceTree.VersionNum,
		Admins:       serviceTree.Admins,
	}, nil
}

func (s *PackageService) sendCreatePackageMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
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

	if _, err := s.appCall.CreateServiceTree(ctx, appModel.HostID, &req); err != nil {
		return fmt.Errorf("failed to create service tree via app-runtime: %w", err)
	}

	logger.Infof(ctx, "[PackageService] Service tree created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)
	return nil
}

func assignDirectoryAdminRoleToUser(ctx context.Context, user, app, username, resourcePath string) error {
	licenseMgr := license.GetManager()
	if !licenseMgr.HasFeature(enterprise.FeaturePermission) {
		return nil
	}

	permissionService := enterprise.GetPermissionService()
	if permissionService == nil {
		return fmt.Errorf("权限服务未初始化")
	}

	assignReq := &dto.AssignRoleToUserReq{
		User:         user,
		App:          app,
		Username:     username,
		RoleCode:     "admin",
		ResourceType: "directory",
		ResourcePath: resourcePath,
		StartTime:    nil,
		EndTime:      nil,
	}

	if _, err := permissionService.AssignRoleToUser(ctx, assignReq); err != nil {
		return fmt.Errorf("分配管理员角色失败: %w", err)
	}

	logger.Infof(ctx, "[ServiceTree] 分配管理员角色成功: user=%s, app=%s, username=%s, resource=%s",
		user, app, username, resourcePath)
	return nil
}
