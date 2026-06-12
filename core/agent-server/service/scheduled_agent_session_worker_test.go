package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/contextx"
	"github.com/kageos/kageos/pkg/scheduledsdk"
)

func TestScheduledAgentSessionWorkspaceRequestRequiresMessage(t *testing.T) {
	payload := map[string]interface{}{
		"directory":          "/system/test22/ticket_management",
		"cron":               "* * * * *",
		"max_duration_sec":   120,
		"llm_config_id":      7,
		"files":              "bucket/a.txt",
		"display_content":    "定时检查工单",
		"interaction_action": "continue_development",
	}
	raw, _ := json.Marshal(payload)
	_, decoded, err := scheduledAgentSessionWorkspaceRequest(scheduledsdk.ExecutionRequestedEvent{
		ExecutorPayload: raw,
		ResourceKey:     "/fallback",
	})
	if err == nil || !strings.Contains(err.Error(), "requires message") {
		t.Fatalf("expected missing message error, got %v", err)
	}
	if decoded.Message != "" {
		t.Fatalf("decoded message should stay empty, got %q", decoded.Message)
	}
	if decoded.MaxDurationSeconds != 120 {
		t.Fatalf("MaxDurationSeconds = %d, want 120", decoded.MaxDurationSeconds)
	}
}

func TestScheduledAgentSessionWorkspaceRequestUsesMessageDirectly(t *testing.T) {
	payload := map[string]interface{}{
		"full_code_path":  "/system/test22/hot_news",
		"message":         "搜索今天 AI 热点，整理成 Markdown 后发送企业微信群。",
		"display_content": "每日热点推送",
	}
	raw, _ := json.Marshal(payload)
	req, decoded, err := scheduledAgentSessionWorkspaceRequest(scheduledsdk.ExecutionRequestedEvent{
		ExecutorPayload: raw,
	})
	if err != nil {
		t.Fatalf("scheduledAgentSessionWorkspaceRequest() error = %v", err)
	}
	if decoded.Message != "搜索今天 AI 热点，整理成 Markdown 后发送企业微信群。" {
		t.Fatalf("decoded message = %q", decoded.Message)
	}
	if req.Message.DisplayContent != "搜索今天 AI 热点，整理成 Markdown 后发送企业微信群。" {
		t.Fatalf("DisplayContent = %q", req.Message.DisplayContent)
	}
	if strings.Contains(req.Message.Content, "每日热点推送") {
		t.Fatalf("title/display content should not be used as runtime message, got %q", req.Message.Content)
	}
	for _, want := range []string{"定时会话执行约束", "不是创建或管理定时任务", "app_operator", "不要向用户提问", "本次任务绑定工作台目录：/system/test22/hot_news", "搜索今天 AI 热点", "发送企业微信群"} {
		if !strings.Contains(req.Message.Content, want) {
			t.Fatalf("message content should contain %q, got %q", want, req.Message.Content)
		}
	}
}

func TestScheduledAgentSessionSinkBuildsExecutionResult(t *testing.T) {
	sink := &scheduledAgentSessionSink{}
	sink.Send(EventSession, dto.WorkspaceStreamSession{SessionID: "session-1"})
	sink.Send(EventContent, dto.WorkspaceStreamContent{Content: "执行完成。"})
	sink.Send(EventDone, dto.WorkspaceStreamDone{
		SessionID: "session-1",
		ToolCalls: []dto.WorkspaceChatToolCallSummary{
			{Name: "read_dir", Status: ToolCallStatusOK},
			{Name: "run_form_submit", Status: ToolCallStatusError},
		},
	})

	result := sink.ExecutionResult()
	if result.ExecutorRunID != "session-1" {
		t.Fatalf("ExecutorRunID = %q", result.ExecutorRunID)
	}
	if !strings.Contains(result.OutputSummary, "工具调用 2 次，失败 1 次") {
		t.Fatalf("unexpected summary: %q", result.OutputSummary)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(result.ResultPayload, &payload); err != nil {
		t.Fatalf("result payload should be JSON: %v", err)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("unexpected result payload: %#v", payload)
	}
}

func TestScheduledAgentSessionRunErrorExplainsMissingDirectory(t *testing.T) {
	err := scheduledAgentSessionRunError(
		"/system/test22/hot_news",
		fmt.Errorf("无效的 full_code_path，无法解析目录: 业务错误 [7]: 获取工作台环境信息失败: 获取目录详情失败: 服务目录不存在"),
	)
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	for _, want := range []string{
		"定时会话配置的工作台目录不存在",
		"/system/test22/hot_news",
		"编辑任务换成有效目录",
	} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error should contain %q, got %q", want, err.Error())
		}
	}
}

func TestAgentToolExecutionContextPreservesScheduledTaskSource(t *testing.T) {
	ctx := contextx.WithRequestInfo(context.Background(), contextx.RequestInfo{
		ClientSource: contextx.ClientSourceScheduledTask,
		SourceType:   contextx.SourceTypeScheduledTask,
		SourceRef:    "timer_task:1:execution:2",
	})
	got := withAgentToolExecutionContext(ctx, "session-1")

	if source := contextx.GetClientSource(got); source != contextx.ClientSourceScheduledTask {
		t.Fatalf("client source = %q, want scheduled_task", source)
	}
	if sourceType := contextx.GetSourceType(got); sourceType != contextx.SourceTypeScheduledTask {
		t.Fatalf("source type = %q, want scheduled_task", sourceType)
	}
	if sourceRef := contextx.GetSourceRef(got); sourceRef != "timer_task:1:execution:2" {
		t.Fatalf("source ref = %q", sourceRef)
	}
}
