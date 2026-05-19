package service

import (
	"context"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type serviceTreeFunctionService struct {
	serviceTreeRepo *repository.ServiceTreeRepository
	appRepo         *repository.AppRepository
	appService      *AppService
}

func newServiceTreeFunctionService(
	serviceTreeRepo *repository.ServiceTreeRepository,
	appRepo *repository.AppRepository,
	appService *AppService,
) *serviceTreeFunctionService {
	return &serviceTreeFunctionService{
		serviceTreeRepo: serviceTreeRepo,
		appRepo:         appRepo,
		appService:      appService,
	}
}

func (s *serviceTreeFunctionService) CreateFunction(ctx context.Context, req *dto.CreateFunctionReq) (*dto.CreateFunctionResp, error) {
	parentTree, err := s.loadTargetTree(ctx, req.DirectoryPath)
	if err != nil {
		return nil, fmt.Errorf("获取父目录失败: %w", err)
	}

	addResp, err := s.AddFunctions(ctx, &dto.AddFunctionsReq{
		FullCodePath: req.DirectoryPath,
		FileName:     req.Code,
		SourceCode:   req.SourceCode,
	})
	if err != nil {
		return nil, fmt.Errorf("创建函数失败: %w", err)
	}
	if !addResp.Success {
		return nil, fmt.Errorf("创建函数失败: %s", addResp.Error)
	}

	expectedPath := req.DirectoryPath + "/" + req.Code
	functionTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(expectedPath)
	if err != nil {
		logger.Warnf(ctx, "[CreateFunction] 无法通过路径查找函数节点: %s, error: %v，返回基本信息", expectedPath, err)
		return &dto.CreateFunctionResp{
			ID:           0,
			Name:         req.Name,
			Code:         req.Code,
			Type:         model.ServiceTreeTypeFunction,
			TemplateType: req.TemplateType,
			Description:  req.Description,
			Tags:         req.Tags,
			AppID:        parentTree.AppID,
			FullCodePath: expectedPath,
			Version:      "v1",
			VersionNum:   1,
		}, nil
	}

	return toCreateFunctionResp(functionTree), nil
}

func (s *serviceTreeFunctionService) AddFunctions(ctx context.Context, req *dto.AddFunctionsReq) (*dto.AddFunctionsResp, error) {
	return addFunctionsImpl(s, ctx, req)
}

func (s *serviceTreeFunctionService) loadTargetTree(ctx context.Context, fullCodePath string) (*model.ServiceTree, error) {
	targetTree, err := s.serviceTreeRepo.GetServiceTreeByFullPath(fullCodePath)
	if err != nil {
		return nil, err
	}
	if targetTree.App == nil {
		app, err := s.appRepo.GetAppByID(targetTree.AppID)
		if err != nil {
			logger.Errorf(ctx, "[ServiceTreeService] 获取 App 失败: AppID=%d, error=%v", targetTree.AppID, err)
			return nil, err
		}
		targetTree.App = app
	}
	return targetTree, nil
}

func toCreateFunctionResp(functionTree *model.ServiceTree) *dto.CreateFunctionResp {
	return &dto.CreateFunctionResp{
		ID:           functionTree.ID,
		Name:         functionTree.Name,
		Code:         functionTree.Code,
		Type:         functionTree.Type,
		TemplateType: functionTree.TemplateType,
		Description:  functionTree.Description,
		Tags:         functionTree.Tags,
		AppID:        functionTree.AppID,
		RefID:        functionTree.RefID,
		FullCodePath: functionTree.FullCodePath,
		Version:      functionTree.Version,
		VersionNum:   functionTree.VersionNum,
	}
}
