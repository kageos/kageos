package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// FunctionGenService 函数生成服务
// 负责调用 plugin 处理输入（通过 Form API），以及发布函数生成结果到 app-server（通过 HTTP）
type FunctionGenService struct {
	cfg             *config.AgentServerConfig
	functionGenRepo *repository.FunctionGenRepository
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(cfg *config.AgentServerConfig, functionGenRepo *repository.FunctionGenRepository) *FunctionGenService {
	return &FunctionGenService{
		cfg:             cfg,
		functionGenRepo: functionGenRepo,
	}
}

// RunPlugin 执行插件处理
// agent: 智能体信息（包含 Plugin 关联）
// req: 插件执行请求（包含用户消息和文件列表）
func (s *FunctionGenService) RunPlugin(ctx context.Context, agent *model.Agent, req *dto.PluginRunReq) (*dto.PluginRunResp, error) {
	traceId := contextx.GetTraceId(ctx)

	// 1. 验证插件类型
	if agent.AgentType != "plugin" {
		return nil, fmt.Errorf("智能体类型不是 plugin，无法调用插件")
	}

	// 2. 验证 PluginFunctionPath
	if agent.PluginFunctionPath == "" {
		return nil, fmt.Errorf("智能体未指定插件函数路径，请先配置 PluginFunctionPath")
	}

	filesCount := 0
	if req.Files != nil {
		filesCount = len(req.Files.Files)
	}
	logger.Infof(ctx, "[FunctionGenService] 开始调用 Form API - PluginFunctionPath: %s, AgentID: %d, ContentLength: %d, FilesCount: %d, TraceID: %s",
		agent.PluginFunctionPath, agent.ID, len(req.Content), filesCount, traceId)

	return s.callFormAPI(ctx, agent.PluginFunctionPath, req, traceId)
}

// callFormAPI 调用 Form API 的通用方法
func (s *FunctionGenService) callFormAPI(ctx context.Context, formPath string, req *dto.PluginRunReq, traceId string) (*dto.PluginRunResp, error) {
	// 1. 构建 Form 请求体（智能体插件场景使用固定格式）
	formReq := &dto.AgentPluginFormReq{
		Content: req.Content,
	}

	// 2. 直接使用 types.Files（无需转换）
	if req.Files != nil {
		formReq.InputFiles = req.Files
	}

	// 3. 构建请求头
	requestUser := contextx.GetRequestUser(ctx)
	token := contextx.GetToken(ctx)

	// ⭐ 确保用户信息不为空，否则权限检查会失败
	if requestUser == "" {
		logger.Warnf(ctx, "[FunctionGenService] RequestUser 为空，可能导致权限检查失败 - FormPath: %s, TraceID: %s", formPath, traceId)
	}

	header := &apicall.Header{
		TraceID:     traceId,
		RequestUser: requestUser,
		Token:       token,
	}

	// 4. 调用 Form API（智能体插件场景使用固定格式）
	startTime := time.Now()
	logger.Debugf(ctx, "[FunctionGenService] 发送 Form API 请求 - FormPath: %s, User: %s, TraceID: %s", formPath, requestUser, traceId)

	resp, err := apicall.CallFormAPI[dto.AgentPluginFormReq, dto.AgentPluginFormResp](header, formPath, *formReq)
	duration := time.Since(startTime)

	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 调用 Form API 失败 - FormPath: %s, Duration: %v, TraceID: %s, Error: %v",
			formPath, duration, traceId, err)
		return &dto.PluginRunResp{
			Error: err.Error(),
		}, nil
	}

	logger.Infof(ctx, "[FunctionGenService] Form API 调用成功 - FormPath: %s, DataLength: %d, Duration: %v, TraceID: %s",
		formPath, len(resp.Result), duration, traceId)

	return &dto.PluginRunResp{
		Data: resp.Result,
	}, nil
}

// SubmitGeneratedCodeTask 提交生成的函数代码任务到 app-server（通过 HTTP）
// req: 函数生成请求（包含生成的代码和目录信息）
func (s *FunctionGenService) SubmitGeneratedCodeTask(ctx context.Context, req *dto.AddFunctionsReq) error {
	logger.Infof(ctx, "[FunctionGenService] 开始提交生成的代码到 app-server (HTTP) - RecordID: %d, AgentID: %d, TreeID: %d",
		req.RecordID, req.AgentID, req.TreeID)

	// 1. 构建请求头
	token := contextx.GetToken(ctx)
	header := &apicall.Header{
		TraceID:     contextx.GetTraceId(ctx),
		RequestUser: contextx.GetRequestUser(ctx),
		Token:       token,
	}

	// 2. 设置 Async 为 true，使用异步模式（通过回调通知结果）
	req.Async = true

	// 3. 调用 HTTP API 提交代码
	startTime := time.Now()

	_, err := apicall.ServiceTreeAddFunctions(header, req)
	duration := time.Since(startTime)

	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 提交代码到 app-server 失败 - RecordID: %d, Duration: %v, TraceID: %s, Error: %v",
			req.RecordID, duration, contextx.GetTraceId(ctx), err)
		return fmt.Errorf("提交代码到 app-server 失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] 代码提交成功 - RecordID: %d, Duration: %v, TraceID: %s, User: %s, CodeLength: %d",
		req.RecordID, duration, contextx.GetTraceId(ctx), contextx.GetRequestUser(ctx), len(req.Code))

	return nil
}

// ProcessFunctionGenCallback 处理函数生成回调（来自 app-server）
func (s *FunctionGenService) ProcessFunctionGenCallback(ctx context.Context, callback *dto.FunctionGenCallback) error {
	if s.functionGenRepo == nil {
		return fmt.Errorf("FunctionGenRepository 未初始化")
	}

	traceId := contextx.GetTraceId(ctx)
	logger.Infof(ctx, "[FunctionGenService] 处理回调 - RecordID: %d, MessageID: %d, Success: %v, FullCodePaths: %v, AppCode: %s, TraceID: %s",
		callback.RecordID, callback.MessageID, callback.Success, callback.FullCodePaths, callback.AppCode, traceId)

	if callback.Success {
		// 更新记录状态为完成（包含 FullCodePaths）
		if err := s.functionGenRepo.UpdateStatusWithFullCodePaths(callback.RecordID, model.FunctionGenStatusCompleted, "", callback.FullCodePaths); err != nil {
			logger.Errorf(ctx, "[FunctionGenService] 更新记录状态失败 - RecordID: %d, TraceID: %s, Error: %v", callback.RecordID, traceId, err)
			return fmt.Errorf("更新记录状态失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenService] 记录状态已更新为完成 - RecordID: %d, FullCodePaths: %v, TraceID: %s",
			callback.RecordID, callback.FullCodePaths, traceId)
	} else {
		// 更新记录状态为失败
		errorMsg := callback.Error
		if errorMsg == "" {
			errorMsg = "处理失败"
		}
		if err := s.functionGenRepo.UpdateStatus(callback.RecordID, model.FunctionGenStatusFailed, errorMsg); err != nil {
			logger.Errorf(ctx, "[FunctionGenService] 更新记录状态失败 - RecordID: %d, TraceID: %s, Error: %v", callback.RecordID, traceId, err)
			return fmt.Errorf("更新记录状态失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenService] 记录状态已更新为失败 - RecordID: %d, Error: %s, TraceID: %s",
			callback.RecordID, errorMsg, traceId)
	}

	return nil
}
