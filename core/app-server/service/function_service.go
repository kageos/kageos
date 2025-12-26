package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/app-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
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

// GetFunction 获取函数详情
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

	// 转换为响应格式
	resp := &dto.GetFunctionResp{
		ID:           function.ID,
		AppID:        function.AppID,
		TreeID:       function.TreeID,
		Method:       function.Method,
		Router:       function.Router,
		HasConfig:    function.HasConfig,
		CreateTables: function.CreateTables,
		Callbacks:    function.Callbacks,
		TemplateType: function.TemplateType,
		CreatedAt:    time.Time(function.CreatedAt).Format("2006-01-02T15:04:05Z"),
		UpdatedAt:    time.Time(function.UpdatedAt).Format("2006-01-02T15:04:05Z"),
		FullCodePath: function.Router, // Router 存储的就是 full-code-path
	}

	// 将json.RawMessage转换为interface{}以便返回JSON对象
	if len(function.Request) > 0 {
		var requestMap interface{}
		if err := json.Unmarshal(function.Request, &requestMap); err != nil {
			requestMap = map[string]interface{}{}
		}
		resp.Request = requestMap
	} else {
		resp.Request = map[string]interface{}{}
	}

	if len(function.Response) > 0 {
		var responseMap interface{}
		if err := json.Unmarshal(function.Response, &responseMap); err != nil {
			responseMap = map[string]interface{}{}
		}
		resp.Response = responseMap
	} else {
		resp.Response = map[string]interface{}{}
	}

	return resp, nil
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
			Callbacks:    function.Callbacks,
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
