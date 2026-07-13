package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kageos/kageos/core/app-server/model"
	"github.com/kageos/kageos/core/app-server/repository"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
	"gorm.io/gorm"
)

type FunctionService struct {
	functionRepo *repository.FunctionRepository
	appRepo      *repository.AppRepository
}

// NewFunctionService 创建函数服务
func NewFunctionService(
	functionRepo *repository.FunctionRepository,
	appRepo *repository.AppRepository,
) *FunctionService {
	return &FunctionService{
		functionRepo: functionRepo,
		appRepo:      appRepo,
	}
}

// GetFunctionByID 根据函数 ID 获取函数模型。
func (f *FunctionService) GetFunctionByID(ctx context.Context, functionID int64) (*model.Function, error) {
	function, err := f.functionRepo.GetFunctionByID(ctx, functionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}
	return function, nil
}

// GetFunctionByFullCodePath 根据 full-code-path 获取函数详情。
// 这里只返回函数基础信息，调用方负责组合页面所需的上下文。
func (f *FunctionService) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*dto.GetFunctionResp, error) {
	// 从数据库获取函数信息
	function, err := f.functionRepo.GetFunctionByFullCodePath(ctx, fullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}

	return f.convertFunctionToResp(ctx, function), nil
}

// GetFunction 获取函数详情（根据 ID，保留给内部旧调用方）。
// 这里只返回函数基础信息，调用方负责组合页面所需的上下文。
func (f *FunctionService) GetFunction(ctx context.Context, functionID int64) (*dto.GetFunctionResp, error) {
	// 从数据库获取函数信息
	function, err := f.functionRepo.GetFunctionByID(ctx, functionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}

	return f.convertFunctionToResp(ctx, function), nil
}

// convertFunctionToResp 将函数模型转换为响应格式
func (f *FunctionService) convertFunctionToResp(ctx context.Context, function *model.Function) *dto.GetFunctionResp {
	connectors := splitConnectorCodes(function.Connectors)
	connectorEndpoints := splitConnectorEndpoints(function.ConnectorEndpoints)

	resp := &dto.GetFunctionResp{
		ID:                 function.ID,
		AppID:              function.AppID,
		TreeID:             function.TreeID,
		Method:             function.Method,
		Router:             function.Router,
		HasConfig:          function.HasConfig,
		CreateTables:       function.CreateTables,
		Connectors:         connectors,
		ConnectorEndpoints: connectorEndpoints,
		ConnectorStatus:    functionConnectorStatuses(ctx, function.Router, connectors, connectorEndpoints),
		TemplateType:       function.TemplateType,
		CreatedAt:          time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:          time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		CreatedBy:          function.CreatedBy, // 创建者用户名
		FullCodePath:       function.Router,    // Router 存储的就是 full-code-path
	}

	schema, err := functionschema.Parse(function.Schema)
	if err == nil {
		resp.Schema = schema
	}

	return resp
}

// GetAppByUserAndCode 根据用户和应用代码获取应用信息。
func (f *FunctionService) GetAppByUserAndCode(ctx context.Context, user, app string) (*model.App, error) {
	return f.appRepo.GetAppByUserName(ctx, user, app)
}

// GetFunctionsByApp 获取应用下所有函数
func (f *FunctionService) GetFunctionsByApp(ctx context.Context, appID int64) (*dto.GetFunctionsByAppResp, error) {
	// 从数据库获取应用的所有函数
	functions, err := f.functionRepo.GetFunctionsByAppID(ctx, appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用函数列表失败: %w", err)
	}

	// 转换为响应格式
	functionInfos := make([]dto.FunctionInfo, len(functions))
	for i, function := range functions {
		functionInfos[i] = dto.FunctionInfo{
			ID:                 function.ID,
			AppID:              function.AppID,
			TreeID:             function.TreeID,
			Method:             function.Method,
			Router:             function.Router,
			HasConfig:          function.HasConfig,
			CreateTables:       function.CreateTables,
			Connectors:         splitConnectorCodes(function.Connectors),
			ConnectorEndpoints: splitConnectorEndpoints(function.ConnectorEndpoints),
			Callbacks:          function.GetCallbacks(),
			TemplateType:       function.TemplateType,
			CreatedAt:          time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:          time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		}
	}

	resp := &dto.GetFunctionsByAppResp{
		Functions: functionInfos,
	}

	logger.Infof(ctx, "[FunctionService] GetFunctionsByApp success: appID=%d, count=%d", appID, len(functions))
	return resp, nil
}
