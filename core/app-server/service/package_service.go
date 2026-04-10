package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type serviceTreePackageService struct {
	serviceTreeRepo  *repository.ServiceTreeRepository
	appRepo          *repository.AppRepository
	runtimeWorkspace *runtimeWorkspaceBridge
}

func newServiceTreePackageService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	runtimeWorkspace *runtimeWorkspaceBridge,
) *serviceTreePackageService {
	return &serviceTreePackageService{
		serviceTreeRepo:  serviceTreeRepo,
		appRepo:          appRepo,
		runtimeWorkspace: runtimeWorkspace,
	}
}

func (s *serviceTreePackageService) CreatePackage(ctx context.Context, req *dto.CreatePackageReq) (*dto.CreatePackageResp, error) {
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
		for admin := range parseAdminUserSet(req.Admins) {
			if admin != requestUser {
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

func (s *serviceTreePackageService) sendCreatePackageMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
	if err := s.runtimeWorkspace.createDirectoryScaffold(ctx, user, app, serviceTree); err != nil {
		return err
	}

	logger.Infof(ctx, "[PackageService] Directory scaffold created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)
	return nil
}
