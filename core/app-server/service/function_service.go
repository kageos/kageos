package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/functionschema"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
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

// GetFunctionByID 根据函数ID获取函数模型（用于权限检查等）
func (f *FunctionService) GetFunctionByID(ctx context.Context, functionID int64) (*model.Function, error) {
	function, err := f.functionRepo.GetFunctionByID(functionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}
	return function, nil
}

// GetFunctionByFullCodePath 根据 full-code-path 获取函数详情
// ⭐ 注意：权限信息在 API Handler 中查询并添加到响应中，这里只返回基础信息
func (f *FunctionService) GetFunctionByFullCodePath(ctx context.Context, fullCodePath string) (*dto.GetFunctionResp, error) {
	// 从数据库获取函数信息
	function, err := f.functionRepo.GetFunctionByFullCodePath(fullCodePath)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}

	return f.convertFunctionToResp(function), nil
}

// GetFunction 获取函数详情（根据ID，保留用于向后兼容）
// ⭐ 注意：权限信息在 API Handler 中查询并添加到响应中，这里只返回基础信息
func (f *FunctionService) GetFunction(ctx context.Context, functionID int64) (*dto.GetFunctionResp, error) {
	// 从数据库获取函数信息
	function, err := f.functionRepo.GetFunctionByID(functionID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("函数不存在")
		}
		return nil, fmt.Errorf("获取函数失败: %w", err)
	}

	return f.convertFunctionToResp(function), nil
}

// convertFunctionToResp 将函数模型转换为响应格式
func (f *FunctionService) convertFunctionToResp(function *model.Function) *dto.GetFunctionResp {

	resp := &dto.GetFunctionResp{
		ID:           function.ID,
		AppID:        function.AppID,
		TreeID:       function.TreeID,
		Method:       function.Method,
		Router:       function.Router,
		HasConfig:    function.HasConfig,
		CreateTables: function.CreateTables,
		TemplateType: function.TemplateType,
		CreatedAt:    time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		CreatedBy:    function.CreatedBy, // 创建者用户名
		FullCodePath: function.Router,    // Router 存储的就是 full-code-path
	}

	schema, err := functionschema.Parse(function.Schema)
	if err == nil {
		resp.Schema = schema
	}

	return resp
}

// GetAppByUserAndCode 根据用户和应用代码获取应用信息（用于权限检查）
func (f *FunctionService) GetAppByUserAndCode(ctx context.Context, user, app string) (*model.App, error) {
	return f.appRepo.GetAppByUserName(user, app)
}

// GetFunctionsByApp 获取应用下所有函数
func (f *FunctionService) GetFunctionsByApp(ctx context.Context, appID int64) (*dto.GetFunctionsByAppResp, error) {
	// 从数据库获取应用的所有函数
	functions, err := f.functionRepo.GetFunctionsByAppID(appID)
	if err != nil {
		return nil, fmt.Errorf("获取应用函数列表失败: %w", err)
	}

	// 转换为响应格式
	functionInfos := make([]dto.FunctionInfo, len(functions))
	for i, function := range functions {
		functionInfos[i] = dto.FunctionInfo{
			ID:           function.ID,
			AppID:        function.AppID,
			TreeID:       function.TreeID,
			Method:       function.Method,
			Router:       function.Router,
			HasConfig:    function.HasConfig,
			CreateTables: function.CreateTables,
			Callbacks:    function.GetCallbacks(),
			TemplateType: function.TemplateType,
			CreatedAt:    time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		}
	}

	resp := &dto.GetFunctionsByAppResp{
		Functions: functionInfos,
	}

	logger.Infof(ctx, "[FunctionService] GetFunctionsByApp success: appID=%d, count=%d", appID, len(functions))
	return resp, nil
}
