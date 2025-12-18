package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/config"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/msgx"
	"github.com/ai-agent-os/ai-agent-os/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// FunctionGenService 函数生成服务
// 负责调用 plugin 处理输入，以及发布函数生成结果到 app-server（通过 HTTP）
type FunctionGenService struct {
	natsConn        *nats.Conn
	cfg             *config.AgentServerConfig
	functionGenRepo *repository.FunctionGenRepository
}

// NewFunctionGenService 创建函数生成服务
func NewFunctionGenService(natsConn *nats.Conn, cfg *config.AgentServerConfig) *FunctionGenService {
	return &FunctionGenService{
		natsConn: natsConn,
		cfg:      cfg,
	}
}

// SetFunctionGenRepository 设置函数生成仓库（延迟注入，避免循环依赖）
func (s *FunctionGenService) SetFunctionGenRepository(repo *repository.FunctionGenRepository) {
	s.functionGenRepo = repo
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

	// 2. 获取插件信息
	if agent.PluginID == nil || *agent.PluginID == 0 {
		// 向后兼容：如果没有关联插件，使用旧的逻辑
		pluginSubject := subjects.BuildAgentPluginRunSubject(agent.ChatType, agent.CreatedBy, agent.ID)
		logger.Warnf(ctx, "[FunctionGenService] 智能体未关联插件，使用旧的插件主题 - Subject: %s, AgentID: %d, TraceID: %s",
			pluginSubject, agent.ID, traceId)
		return s.callPlugin(ctx, pluginSubject, agent.ID, req, traceId)
	}

	// 3. 验证插件是否已预加载
	if agent.Plugin == nil {
		return nil, fmt.Errorf("插件信息未预加载，请确保 AgentRepository.GetByID 预加载了 Plugin")
	}

	plugin := agent.Plugin
	if !plugin.Enabled {
		return nil, fmt.Errorf("插件已禁用: PluginID=%d", plugin.ID)
	}

	// 4. 使用插件的主题
	pluginSubject := plugin.Subject
	if pluginSubject == "" {
		return nil, fmt.Errorf("插件主题为空: PluginID=%d", plugin.ID)
	}

	logger.Infof(ctx, "[FunctionGenService] 开始调用 Plugin - Subject: %s, PluginID: %d, AgentID: %d, MessageLength: %d, FilesCount: %d, TraceID: %s",
		pluginSubject, plugin.ID, agent.ID, len(req.Message), len(req.Files), traceId)

	return s.callPlugin(ctx, pluginSubject, agent.ID, req, traceId)
}

// callPlugin 调用插件的通用方法
func (s *FunctionGenService) callPlugin(ctx context.Context, pluginSubject string, agentID int64, req *dto.PluginRunReq, traceId string) (*dto.PluginRunResp, error) {

	// 调用插件（使用 NATS Request/Reply 模式）
	var pluginResp dto.PluginRunResp
	timeout := time.Duration(s.cfg.GetNatsTimeout()) * time.Second
	logger.Debugf(ctx, "[FunctionGenService] 发送 NATS 请求 - Subject: %s, Timeout: %v, TraceID: %s",
		pluginSubject, timeout, traceId)

	startTime := time.Now()
	_, err := msgx.RequestMsgWithTimeout(ctx, s.natsConn, pluginSubject, req, &pluginResp, timeout)
	duration := time.Since(startTime)

	if err != nil {
		logger.Errorf(ctx, "[FunctionGenService] 调用插件失败 - Subject: %s, AgentID: %d, Duration: %v, TraceID: %s, Error: %v",
			pluginSubject, agentID, duration, traceId, err)
		return nil, fmt.Errorf("调用 plugin 失败: %w", err)
	}

	logger.Infof(ctx, "[FunctionGenService] 插件执行成功 - Subject: %s, AgentID: %d, DataLength: %d, Duration: %v, TraceID: %s",
		pluginSubject, agentID, len(pluginResp.Data), duration, traceId)

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

// FunctionGenChat 方法已移至 agent_chat_service_function_gen.go
// 支持格式：
// 1. ```go\n代码\n```
// 2. ```\n代码\n```
// 3. 如果找不到代码块，返回原始内容
func extractCodeFromMarkdown(content string) string {
	// 查找 ```go 或 ``` 开头的代码块
	lines := strings.Split(content, "\n")

	var codeBlocks []string
	var inCodeBlock bool
	var codeBlockStart int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// 检查是否是代码块开始标记
		if strings.HasPrefix(trimmed, "```") {
			if inCodeBlock {
				// 代码块结束，提取内容
				if i > codeBlockStart {
					codeBlock := strings.Join(lines[codeBlockStart+1:i], "\n")
					codeBlocks = append(codeBlocks, codeBlock)
				}
				inCodeBlock = false
			} else {
				// 代码块开始
				inCodeBlock = true
				codeBlockStart = i
			}
			continue
		}
	}

	// 如果代码块没有正确关闭，也提取已收集的内容
	if inCodeBlock && codeBlockStart < len(lines)-1 {
		codeBlock := strings.Join(lines[codeBlockStart+1:], "\n")
		codeBlocks = append(codeBlocks, codeBlock)
	}

	// 如果有代码块，返回第一个（通常只有一个）
	if len(codeBlocks) > 0 {
		extracted := strings.TrimSpace(codeBlocks[0])
		// 如果提取的代码不为空，返回它
		if extracted != "" {
			return extracted
		}
	}

	// 如果没有找到代码块或代码块为空，返回原始内容（作为 fallback）
	return content
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
