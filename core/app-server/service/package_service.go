package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/naming"
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
	if req == nil {
		return nil, fmt.Errorf("创建目录请求不能为空")
	}
	codeProvided := strings.TrimSpace(req.Code) != ""
	req.Code = naming.NormalizeGoPackageName(req.Code)
	if !codeProvided {
		req.Code = naming.DeriveGoPackageName(req.Name, "directory")
	}
	if err := naming.ValidateGoPackageName(req.Code, "目录英文标识"); err != nil {
		return nil, err
	}

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
		if parentTree.AppID != app.ID {
			return nil, fmt.Errorf("父目录不属于目标应用: %s", req.ParentFullCodePath)
		}
	}

	parentPath := ""
	if parentTree != nil {
		parentPath = parentTree.FullCodePath
	}

	if !codeProvided {
		req.Code, err = s.nextAvailablePackageCode(parentPath, app.ID, req.Code)
		if err != nil {
			return nil, err
		}
	}

	fullCodePath := fmt.Sprintf("/%s/%s/%s", app.User, app.Code, req.Code)
	if parentTree != nil {
		fullCodePath = parentTree.FullCodePath + "/" + req.Code
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

func (s *serviceTreePackageService) nextAvailablePackageCode(parentPath string, appID int64, baseCode string) (string, error) {
	for index := 1; index <= 1000; index++ {
		candidate := baseCode
		if index > 1 {
			candidate = naming.GoPackageNameWithNumericSuffix(baseCode, index)
		}
		exists, err := s.serviceTreeRepo.CheckNameExistsByPath(parentPath, candidate, appID)
		if err != nil {
			return "", fmt.Errorf("failed to check generated directory code exists: %w", err)
		}
		if !exists {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("自动生成目录英文标识失败，请手动填写")
}

func (s *serviceTreePackageService) sendCreatePackageMessage(ctx context.Context, user, app string, serviceTree *model.ServiceTree) error {
	if err := s.runtimeWorkspace.createDirectoryScaffold(ctx, user, app, serviceTree); err != nil {
		return err
	}

	logger.Infof(ctx, "[PackageService] Directory scaffold created successfully via app-runtime: %s/%s/%s",
		user, app, serviceTree.Code)
	return nil
}
