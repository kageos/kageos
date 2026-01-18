package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
	"github.com/nats-io/nats.go"
)

// FunctionGenService 函数生成服务
// 负责调用 plugin 处理输入（通过 Form API），以及发布函数生成结果到 app-server（通过 NATS）
type FunctionGenService struct {
	natsConn        *nats.Conn // NATS 连接，用于发布结果
	cfg             *config.AgentServerConfig
	functionGenRepo *repository.FunctionGenRepository
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(natsConn *nats.Conn, cfg *config.AgentServerConfig, functionGenRepo *repository.FunctionGenRepository) *FunctionGenService {
	return &FunctionGenService{
		natsConn:        natsConn,
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

	// 2. 验证插件是否已关联
	if agent.PluginID == nil || *agent.PluginID == 0 {
		return nil, fmt.Errorf("智能体未关联插件，请先关联插件")
	}

	// 3. 验证插件是否已预加载
	if agent.Plugin == nil {
		return nil, fmt.Errorf("插件信息未预加载，请确保 AgentRepository.GetByID 预加载了 Plugin")
	}

	plugin := agent.Plugin
	if !plugin.Enabled {
		return nil, fmt.Errorf("插件已禁用: PluginID=%d", plugin.ID)
	}

	// 4. 验证 FormPath
	if plugin.FormPath == "" {
		return nil, fmt.Errorf("插件 FormPath 为空: PluginID=%d", plugin.ID)
	}

	logger.Infof(ctx, "[FunctionGenService] 开始调用 Plugin - FormPath: %s, PluginID: %d, AgentID: %d, MessageLength: %d, FilesCount: %d, TraceID: %s",
		plugin.FormPath, plugin.ID, agent.ID, len(req.Message), len(req.Files), traceId)

	return s.callFormAPI(ctx, plugin.FormPath, req, traceId)
}

// callFormAPI 调用 Form API 的通用方法
func (s *FunctionGenService) callFormAPI(ctx context.Context, formPath string, req *dto.PluginRunReq, traceId string) (*dto.PluginRunResp, error) {
	// 1. 构建 Form 请求体（智能体插件场景使用固定格式）
	formReq := &dto.AgentPluginFormReq{
		Message: req.Message,
	}

	// 2. 转换文件列表
	if len(req.Files) > 0 {
		files := make([]*types.File, 0, len(req.Files))
		for _, f := range req.Files {
			files = append(files, &types.File{
				Url:         f.Url,
				Description: f.Remark, // 将 Remark 映射到 Description
			})
		}
		formReq.InputFiles = &types.Files{
			Files: files,
		}
	}

	// 3. 构建请求头
	header := &apicall.Header{
		TraceID:     traceId,
		RequestUser: contextx.GetRequestUser(ctx),
		Token:       contextx.GetToken(ctx),
	}

	// 4. 调用 Form API（智能体插件场景使用固定格式）
	startTime := time.Now()
	logger.Debugf(ctx, "[FunctionGenService] 发送 Form API 请求 - FormPath: %s, TraceID: %s", formPath, traceId)

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

// PublishResult 发布函数生成结果到 NATS
// result: 函数生成结果
// traceId: 追踪ID（用于设置 NATS header）
// requestUser: 请求用户（用于设置 NATS header）
func (s *FunctionGenService) PublishResult(ctx context.Context, result *dto.FunctionGenResult, traceId, requestUser string) error {
	// 1. 构建结果主题
	resultSubject := subjects.GetAgentServerFunctionGenSubject()
	logger.Infof(ctx, "[FunctionGenService] 开始发布结果到 NATS - RecordID: %d, AgentID: %d, TreeID: %d, Subject: %s, TraceID: %s, User: %s",
		result.RecordID, result.AgentID, result.TreeID, resultSubject, traceId, requestUser)

	// 2. 序列化结果
	resultJSON, err := json.Marshal(result)
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 序列化结果失败 - RecordID: %d, TraceID: %s, Error: %v",
			result.RecordID, traceId, err)
		return fmt.Errorf("序列化结果失败: %w", err)
	}
	logger.Debugf(ctx, "[FunctionGenService] 结果序列化成功 - RecordID: %d, JSONLength: %d, CodeLength: %d, TraceID: %s",
		result.RecordID, len(resultJSON), len(result.Code), traceId)

	// 3. 创建 NATS 消息，并在 header 中设置 trace_id 和 user 信息
	msg := nats.NewMsg(resultSubject)
	msg.Data = resultJSON

	// 设置 header，供下游（app-server）使用
	if traceId != "" {
		msg.Header.Set("X-Trace-Id", traceId)
	}
	if requestUser != "" {
		msg.Header.Set("X-Request-User", requestUser)
	}
	logger.Debugf(ctx, "[FunctionGenService] NATS 消息 Header 设置完成 - RecordID: %d, TraceID: %s, User: %s",
		result.RecordID, traceId, requestUser)

	// 4. 发布消息
	if err := s.natsConn.PublishMsg(msg); err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 发布NATS消息失败 - RecordID: %d, Subject: %s, TraceID: %s, Error: %v",
			result.RecordID, resultSubject, traceId, err)
		return fmt.Errorf("发布NATS消息失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] NATS消息发布成功 - RecordID: %d, Subject: %s, TraceID: %s, User: %s, CodeLength: %d",
		result.RecordID, resultSubject, traceId, requestUser, len(result.Code))

	return nil
}

// ProcessFunctionGenCallback 处理函数生成回调（来自 app-server）
func (s *FunctionGenService) ProcessFunctionGenCallback(ctx context.Context, callback *dto.FunctionGenCallback) error {
	if s.functionGenRepo == nil {
		return fmt.Errorf("FunctionGenRepository 未初始化")
	}

	traceId := contextx.GetTraceId(ctx)
	logger.Infof(ctx, "[FunctionGenService] 处理回调 - RecordID: %d, MessageID: %d, Success: %v, FullGroupCodes: %v, AppCode: %s, TraceID: %s",
		callback.RecordID, callback.MessageID, callback.Success, callback.FullGroupCodes, callback.AppCode, traceId)

	if callback.Success {
		// 更新记录状态为完成
		if err := s.functionGenRepo.UpdateStatus(callback.RecordID, model.FunctionGenStatusCompleted, ""); err != nil {
			logger.Errorf(ctx, "[FunctionGenService] 更新记录状态失败 - RecordID: %d, TraceID: %s, Error: %v", callback.RecordID, traceId, err)
			return fmt.Errorf("更新记录状态失败: %w", err)
		}
		logger.Infof(ctx, "[FunctionGenService] 记录状态已更新为完成 - RecordID: %d, FullGroupCodes: %v, TraceID: %s",
			callback.RecordID, callback.FullGroupCodes, traceId)
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
