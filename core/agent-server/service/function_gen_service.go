package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// FunctionGenService 函数生成服务
// 负责调用 plugin 处理输入，以及发布函数生成结果到 NATS
type FunctionGenService struct {
	natsConn *nats.Conn
	cfg      *config.AgentServerConfig
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(natsConn *nats.Conn, cfg *config.AgentServerConfig) *FunctionGenService {
	return &FunctionGenService{
		natsConn: natsConn,
		cfg:      cfg,
	}
}

// RunPlugin 执行插件处理
// agent: 智能体信息（包含 ChatType、CreatedBy、ID 等信息）
// req: 插件执行请求（包含用户消息和文件列表）
func (s *FunctionGenService) RunPlugin(ctx context.Context, agent *model.Agent, req *dto.PluginRunReq) (*dto.PluginRunResp, error) {
	traceId := contextx.GetTraceId(ctx)
	
	// 1. 构建插件主题
	pluginSubject := subjects.BuildAgentPluginRunSubject(agent.ChatType, agent.CreatedBy, agent.ID)
	logger.Infof(ctx, "[FunctionGenService] 开始调用 Plugin - Subject: %s, AgentID: %d, MessageLength: %d, FilesCount: %d, TraceID: %s",
		pluginSubject, agent.ID, len(req.Message), len(req.Files), traceId)

	// 2. 调用插件（使用 NATS Request/Reply 模式）
	var pluginResp dto.PluginRunResp
	timeout := time.Duration(s.cfg.GetNatsTimeout()) * time.Second
	logger.Debugf(ctx, "[FunctionGenService] 发送 NATS 请求 - Subject: %s, Timeout: %v, TraceID: %s",
		pluginSubject, timeout, traceId)
	
	startTime := time.Now()
	_, err := msgx.RequestMsgWithTimeout(ctx, s.natsConn, pluginSubject, req, &pluginResp, timeout)
	duration := time.Since(startTime)
	
	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 调用插件失败 - Subject: %s, AgentID: %d, Duration: %v, TraceID: %s, Error: %v",
			pluginSubject, agent.ID, duration, traceId, err)
		return nil, fmt.Errorf("调用 plugin 失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] 插件执行成功 - Subject: %s, AgentID: %d, DataLength: %d, Duration: %v, TraceID: %s",
		pluginSubject, agent.ID, len(pluginResp.Data), duration, traceId)

	return &pluginResp, nil
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

