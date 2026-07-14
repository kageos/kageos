package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/llms"
	"github.com/kageos/kageos/pkg/logger"
)

// executeToolCalls 执行工具调用并保存消息。若 tool 消息保存失败则返回 error，不再进入下一轮，避免 400 insufficient tool messages。
func (s *WorkspaceChatService) executeToolCalls(
	ctx context.Context,
	allToolCalls []llms.ToolCall,
	sessionID, fullCodePath string,
	user string,
	files string,
	round int,
	sendEvent func(string, interface{}),
) ([]dto.WorkspaceChatToolCallSummary, string, error) {
	ctx = contextx.WithInitiatorUser(ctx, user)
	sessionContext := s.workspaceSessionMessageContext(ctx, sessionID)
	ctx = withAgentToolExecutionContext(ctx, sessionID, sessionContext.title, sessionContext.role)
	ctx = withWorkspaceSessionSourceDisplay(ctx, sessionContext, fullCodePath)
	toolSummaries := make([]dto.WorkspaceChatToolCallSummary, 0, len(allToolCalls))
	logger.Infof(ctx, "[WorkspaceChatStream] 开始执行工具调用 - 工具数量: %d, SessionID: %s", len(allToolCalls), sessionID)
	loadedGuideDocs := s.loadedGuideDocsForSession(ctx, sessionID)
	activeRoleID := s.currentWorkspaceRoleForSession(ctx, sessionID)
	activeFullCodePath := strings.TrimSpace(fullCodePath)
	seenChangeRoleCalls := make(map[string]ToolResult)

	for i, tc := range allToolCalls {
		if err := ctx.Err(); err != nil {
			logger.Infof(ctx, "[WorkspaceChatStream] 工具调用队列已取消 - SessionID: %s, Round: %d", sessionID, round)
			return toolSummaries, activeFullCodePath, err
		}
		logger.Debugf(ctx, "[WorkspaceChatStream] [%d/%d] 执行工具调用 - ToolCallID: %s, ToolName: %s, Arguments: %q",
			i+1, len(allToolCalls), tc.ID, tc.Function.Name, tc.Function.Arguments)

		sendEvent(EventToolCall, StreamEventToolCall{
			ID:        tc.ID,
			Index:     i,
			Round:     round,
			Name:      tc.Function.Name,
			Status:    ToolCallStatusRunning,
			Arguments: tc.Function.Arguments,
		})

		args, parseErr := s.parseToolCallArgs(ctx, tc)
		var toolRes ToolResult
		var st string
		if parseErr != nil {
			toolRes = invalidToolArgumentsResult(tc, parseErr)
			st = ToolCallStatusError
		} else if tc.Function.Name == "change_role" {
			duplicateKey := duplicateChangeRoleToolKey(args)
			if duplicateKey != "" {
				if previous, ok := seenChangeRoleCalls[duplicateKey]; ok && !previous.IsError {
					toolRes = compactDuplicateChangeRoleResult(previous)
					st = ToolCallStatusOK
					logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 跳过重复 change_role 调用 - ToolCallID: %s", i+1, len(allToolCalls), tc.ID)
				}
			}
			if st == "" {
				if blockedRes, blocked := workspaceRoleToolGateResult(activeRoleID, tc.Function.Name); blocked {
					toolRes = blockedRes
					st = ToolCallStatusError
					logger.Warnf(ctx, "[WorkspaceChatStream] [%d/%d] 角色工具门禁阻断 - RoleID: %s, ToolName: %s, Error: %s", i+1, len(allToolCalls), activeRoleID, tc.Function.Name, toolRes.Content)
				} else {
					toolCtx := withWorkspaceToolSourceDisplay(contextx.WithToolCallInfo(ctx, tc.ID, tc.Function.Name), activeFullCodePath)
					toolRes, st = s.callOtherTool(toolCtx, tc.Function.Name, args, activeFullCodePath, files, i+1, len(allToolCalls))
				}
				if duplicateKey != "" {
					seenChangeRoleCalls[duplicateKey] = toolRes
				}
			}
		} else if blockedRes, blocked := workspaceRoleToolGateResult(activeRoleID, tc.Function.Name); blocked {
			toolRes = blockedRes
			st = ToolCallStatusError
			logger.Warnf(ctx, "[WorkspaceChatStream] [%d/%d] 角色工具门禁阻断 - RoleID: %s, ToolName: %s, Error: %s", i+1, len(allToolCalls), activeRoleID, tc.Function.Name, toolRes.Content)
		} else {
			toolCtx := withWorkspaceToolSourceDisplay(contextx.WithToolCallInfo(ctx, tc.ID, tc.Function.Name), activeFullCodePath)
			toolRes, st = s.callOtherTool(toolCtx, tc.Function.Name, args, activeFullCodePath, files, i+1, len(allToolCalls))
		}
		if err := ctx.Err(); err != nil {
			logger.Infof(ctx, "[WorkspaceChatStream] 工具调用已取消 - SessionID: %s, ToolName: %s", sessionID, tc.Function.Name)
			return toolSummaries, activeFullCodePath, err
		}
		roleBeforeCall := activeRoleID
		roleChanged := false
		if st == ToolCallStatusOK && tc.Function.Name == "change_role" {
			if nextRoleID := workspaceRoleFromToolResult(toolRes); nextRoleID != "" {
				activeRoleID = nextRoleID
				s.updateWorkspaceSessionRole(ctx, sessionID, nextRoleID, user)
				roleChanged = roleBeforeCall != "" && roleBeforeCall != nextRoleID
			}
			if nextFullCodePath := workspaceChangeRoleExecuteDirectory(toolRes); nextFullCodePath != "" {
				activeFullCodePath = nextFullCodePath
				s.updateWorkspaceSessionFullCodePath(ctx, sessionID, nextFullCodePath, user)
			}
		} else if st == ToolCallStatusOK && tc.Function.Name == "create_directory" {
			if targetPath := workspaceCreateDirectoryTargetPath(args, activeFullCodePath); targetPath != "" {
				activeFullCodePath = targetPath
				s.updateWorkspaceSessionFullCodePath(ctx, sessionID, targetPath, user)
			}
		}

		resultStr, errStr := "", ""
		var resultData interface{}
		if st == ToolCallStatusOK {
			resultStr = toolRes.Content
			resultData = toolRes.Data
		} else {
			errStr = toolRes.Content
		}
		toolSummaries = append(toolSummaries, dto.WorkspaceChatToolCallSummary{
			ID:         tc.ID,
			Index:      i,
			Round:      round,
			Name:       tc.Function.Name,
			Status:     st,
			Arguments:  tc.Function.Arguments,
			Result:     resultStr,
			ResultData: resultData,
			Metadata:   toolRes.Metadata,
			Error:      errStr,
		})
		sendEvent(EventToolCall, StreamEventToolCall{
			ID:         tc.ID,
			Index:      i,
			Round:      round,
			Name:       tc.Function.Name,
			Status:     st,
			Arguments:  tc.Function.Arguments,
			Result:     resultStr,
			ResultData: resultData,
			Metadata:   toolRes.Metadata,
			Error:      errStr,
		})
		if err := s.saveToolMessage(ctx, sessionID, tc.ID, tc.Function.Name, st, toolRes, user); err != nil {
			logger.Warnf(ctx, "[WorkspaceChatStream] 保存 tool 消息失败 ToolCallID=%s: %v（若为 Error 1366 请将表转为 utf8mb4）", tc.ID, err)
			return toolSummaries, activeFullCodePath, fmt.Errorf("保存 tool 消息失败: %w", err)
		}
		if st == ToolCallStatusOK && tc.Function.Name == "change_role" {
			s.updateWorkspaceModelContextAnchorAfterChangeRole(ctx, sessionID, toolRes, user, roleChanged)
		}
		updateLoadedGuideDocsAfterToolCall(loadedGuideDocs, tc.Function.Name, args, st)
	}

	successCount, errorCount := 0, 0
	for _, ts := range toolSummaries {
		if ts.Status == ToolCallStatusOK {
			successCount++
		} else {
			errorCount++
		}
	}
	logger.Infof(ctx, "[WorkspaceChatStream] 工具调用执行完成 - 总数量: %d, 成功: %d, 失败: %d, SessionID: %s",
		len(allToolCalls), successCount, errorCount, sessionID)
	return toolSummaries, activeFullCodePath, nil
}

func duplicateChangeRoleToolKey(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}
	b, err := json.Marshal(args)
	if err != nil {
		return ""
	}
	return string(b)
}

func compactDuplicateChangeRoleResult(previous ToolResult) ToolResult {
	roleID := workspaceRoleFromToolResult(previous)
	executeDirectory := workspaceChangeRoleExecuteDirectory(previous)
	displayName := workspaceRoleDisplayName(roleID)
	if displayName == "" {
		displayName = roleID
	}
	data := map[string]interface{}{
		"role_id":           roleID,
		"current_role":      roleID,
		"display_name":      displayName,
		"execute_directory": executeDirectory,
		"duplicate":         true,
	}
	content := "重复 change_role 调用已跳过，沿用本轮前一次角色切换结果。"
	if displayName != "" {
		content += "\n当前角色: " + displayName
	}
	if executeDirectory != "" {
		content += "\n执行目录: " + executeDirectory
	}
	return toolResultWithData(content, false, data)
}

type workspaceSessionMessageMetadata struct {
	title        string
	role         string
	sourcePath   string
	parentPath   string
	templateType string
}

func (s *WorkspaceChatService) workspaceSessionMessageContext(ctx context.Context, sessionID string) workspaceSessionMessageMetadata {
	if s == nil || s.sessionRepo == nil || strings.TrimSpace(sessionID) == "" {
		return workspaceSessionMessageMetadata{}
	}
	session, err := s.sessionRepo.GetBySessionID(ctx, sessionID)
	if err != nil || session == nil {
		return workspaceSessionMessageMetadata{}
	}
	executeDirectory := normalizeWorkspacePath(session.FullCodePath)
	resourcePath := normalizeWorkspacePath(session.ResourceFullCodePath)
	sourcePath := resourcePath
	if sourcePath == "" {
		sourcePath = executeDirectory
	}
	parentPath := ""
	if resourcePath != "" && executeDirectory != "" && resourcePath != executeDirectory {
		parentPath = executeDirectory
	}
	return workspaceSessionMessageMetadata{
		title:        strings.TrimSpace(session.Title),
		role:         workspaceSessionRoleID(session),
		sourcePath:   sourcePath,
		parentPath:   parentPath,
		templateType: scheduledFunctionTemplateType(resourcePath),
	}
}

func withAgentToolExecutionContext(ctx context.Context, sessionID, sessionTitle, role string) context.Context {
	ctx = contextx.WithWorkspaceSession(ctx, sessionID, sessionTitle, role)
	if contextx.ResolveClientSource(ctx) == contextx.ClientSourceScheduledTask {
		return ctx
	}
	ctx = withAgentToolClientSource(ctx)
	return contextx.WithSourceInfo(ctx, contextx.SourceTypeAgentTool, sessionID)
}

func withWorkspaceToolSourceDisplay(ctx context.Context, fullCodePath string) context.Context {
	if strings.TrimSpace(contextx.GetSourcePath(ctx)) != "" {
		return ctx
	}
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return ctx
	}
	return contextx.WithSourceDisplay(ctx, fullCodePath, "", "", "", "")
}

func withWorkspaceSessionSourceDisplay(ctx context.Context, session workspaceSessionMessageMetadata, fallbackFullCodePath string) context.Context {
	if strings.TrimSpace(contextx.GetSourcePath(ctx)) != "" {
		return ctx
	}
	sourcePath := strings.TrimSpace(session.sourcePath)
	if sourcePath == "" {
		return withWorkspaceToolSourceDisplay(ctx, fallbackFullCodePath)
	}
	return contextx.WithSourceDisplay(
		ctx,
		sourcePath,
		strings.TrimSpace(session.title),
		strings.TrimSpace(session.parentPath),
		"",
		strings.TrimSpace(session.templateType),
	)
}

// parseToolCallArgs 解析 tool_call 的 arguments JSON，解析失败时返回错误，由调用方保存一条 tool error，避免坏参数继续执行真实工具。
func (s *WorkspaceChatService) parseToolCallArgs(ctx context.Context, tc llms.ToolCall) (map[string]interface{}, error) {
	argumentsStr := strings.TrimSpace(tc.Function.Arguments)
	if argumentsStr == "" {
		logger.Warnf(ctx, "[WorkspaceChatStream] tool_call arguments 为空，使用空对象，ToolCallID: %s, ToolName: %s", tc.ID, tc.Function.Name)
		return map[string]interface{}{}, nil
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argumentsStr), &args); err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 解析 tool_call arguments 失败: %v, ToolCallID: %s, ToolName: %s", err, tc.ID, tc.Function.Name)
		return map[string]interface{}{}, err
	}
	if args == nil {
		args = map[string]interface{}{}
	}
	return args, nil
}

func invalidToolArgumentsResult(tc llms.ToolCall, err error) ToolResult {
	toolName := strings.TrimSpace(tc.Function.Name)
	if toolName == "" {
		toolName = "工具"
	}
	return toolResult(fmt.Sprintf("%s 参数不是合法 JSON，本次未执行工具。请重新调用该工具，并提供完整 JSON 参数。解析错误: %v", toolName, err), true)
}

// callOtherTool 调用 ToolRegistry 中的内置工作台工具。
func (s *WorkspaceChatService) callOtherTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files string, idx, total int) (res ToolResult, st string) {
	logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 调用工具 - ToolName: %s, FullCodePath: %s", idx, total, name, fullCodePath)
	var result ToolResult
	if name == workspaceArtifactReadToolName {
		result = s.readWorkspaceArtifactTool(ctx, args)
	} else {
		result = s.toolReg.CallTool(ctx, name, args, fullCodePath, files)
	}
	isErr := result.IsError
	st = ToolCallStatusOK
	if isErr {
		st = ToolCallStatusError
		logger.Warnf(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用失败 - ToolName: %s, Error: %s", idx, total, name, result.Content)
	} else if len(result.Content) > 200 {
		logger.Infof(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用成功 - ToolName: %s, ResultLength: %d", idx, total, name, len(result.Content))
	} else {
		logger.Debugf(ctx, "[WorkspaceChatStream] [%d/%d] 工具调用成功 - ToolName: %s, Result: %s", idx, total, name, result.Content)
	}
	return result, st
}

func (s *WorkspaceChatService) loadedGuideDocsForSession(ctx context.Context, sessionID string) map[string]struct{} {
	loaded := make(map[string]struct{})
	if s == nil || s.messageRepo == nil || strings.TrimSpace(sessionID) == "" {
		return loaded
	}
	messages, err := s.messageRepo.ListBySessionID(ctx, sessionID)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 查询会话已读 SOP 失败 SessionID=%s: %v", sessionID, err)
		return loaded
	}
	return loadedGuideDocsFromMessages(ctx, messages)
}

func loadedGuideDocsFromMessages(ctx context.Context, messages []*model.AgentChatMessage) map[string]struct{} {
	loaded := make(map[string]struct{})
	readDocCalls := make(map[string][]string)
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		switch msg.Role {
		case RoleAssistant:
			if msg.ToolCalls == nil || strings.TrimSpace(*msg.ToolCalls) == "" {
				continue
			}
			var toolCalls []llms.ToolCall
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err != nil {
				logger.Warnf(ctx, "[WorkspaceChatStream] 解析历史 tool_calls 失败 MessageID=%d: %v", msg.ID, err)
				continue
			}
			for _, tc := range toolCalls {
				if strings.TrimSpace(tc.ID) == "" {
					continue
				}
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					logger.Warnf(ctx, "[WorkspaceChatStream] 解析历史 %s 参数失败 ToolCallID=%s: %v", tc.Function.Name, tc.ID, err)
					continue
				}
				switch tc.Function.Name {
				case "read_doc":
					readDocCalls[tc.ID] = guideDocPathsFromReadDocArgs(args)
				}
			}
		case RoleTool:
			if msg.ToolStatus != ToolCallStatusOK {
				continue
			}
			for _, docPath := range readDocCalls[msg.ToolCallID] {
				loaded[docPath] = struct{}{}
			}
		}
	}
	return loaded
}

func guideDocPathsFromReadDocArgs(args map[string]interface{}) []string {
	dirArg, _ := args["directory"].(string)
	if strings.TrimSpace(dirArg) == "" {
		dirArg, _ = args["full_code_path"].(string)
	}
	rawPaths := splitDirectoryPaths(dirArg)
	paths := make([]string, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		path := normalizeGuideDocPath(rawPath)
		if path == "" {
			continue
		}
		paths = append(paths, path)
	}
	return paths
}

func updateLoadedGuideDocsAfterToolCall(loaded map[string]struct{}, toolName string, args map[string]interface{}, status string) {
	if status != ToolCallStatusOK || loaded == nil {
		return
	}
	switch strings.TrimSpace(toolName) {
	case "read_doc":
		for _, docPath := range guideDocPathsFromReadDocArgs(args) {
			loaded[docPath] = struct{}{}
		}
	}
}

func normalizeGuideDocPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}
