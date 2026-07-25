package streamloop

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/llms"
)

func TestAppendToolCallArgsPreservesStringSpaceAcrossChunks(t *testing.T) {
	got := appendToolCallArgs(`{"deadline":"2026-04-25 `, `18:00:00"}`)
	want := `{"deadline":"2026-04-25 18:00:00"}`
	if got != want {
		t.Fatalf("appendToolCallArgs() = %q, want %q", got, want)
	}
}

func TestRunStreamLoopRetriesWithContextReductionOnContextWindowError(t *testing.T) {
	client := &retryContextWindowClient{}
	deps := &retryContextWindowDeps{client: client}

	if err := RunStreamLoop(context.Background(), deps); err != nil {
		t.Fatalf("RunStreamLoop returned error: %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("client calls = %d, want 2", client.calls)
	}
	if deps.reductions != 1 {
		t.Fatalf("context reductions = %d, want 1", deps.reductions)
	}
	if !deps.done {
		t.Fatal("OnDone should be called after retry succeeds")
	}
	wantActions := []string{"started", "discarded", "started", "committed"}
	if strings.Join(deps.attemptActions, ",") != strings.Join(wantActions, ",") {
		t.Fatalf("generation attempt actions = %#v, want %#v", deps.attemptActions, wantActions)
	}
}

func TestRunStreamLoopRetriesOnceWhenReasoningExhaustsOutputBudget(t *testing.T) {
	client := &retryOutputLimitClient{}
	deps := &retryOutputLimitDeps{client: client}

	if err := RunStreamLoop(context.Background(), deps); err != nil {
		t.Fatalf("RunStreamLoop returned error: %v", err)
	}
	if client.calls != 2 {
		t.Fatalf("client calls = %d, want 2", client.calls)
	}
	if deps.recoveries != 1 {
		t.Fatalf("output limit recoveries = %d, want 1", deps.recoveries)
	}
	if deps.errorEvents != 0 {
		t.Fatalf("error events = %d, want 0", deps.errorEvents)
	}
	if deps.savedContent != "完成" || deps.savedThinking != "精简思考" || !deps.done {
		t.Fatalf("saved content/thinking/done = %q/%q/%v, want 完成/精简思考/true", deps.savedContent, deps.savedThinking, deps.done)
	}
	if deps.doneUsage == nil || deps.doneUsage.CompletionTokens != 8206 {
		t.Fatalf("done usage = %#v, want both attempts accumulated", deps.doneUsage)
	}
}

func TestRunStreamLoopContinuesVisibleOutputAfterRepeatedLengthFinish(t *testing.T) {
	client := &continuationOutputClient{}
	deps := &retryOutputLimitDeps{client: client}

	if err := RunStreamLoop(context.Background(), deps); err != nil {
		t.Fatalf("RunStreamLoop returned error: %v", err)
	}
	if client.calls != 3 {
		t.Fatalf("client calls = %d, want 3", client.calls)
	}
	if deps.savedContent != "第一段第二段" {
		t.Fatalf("saved content = %q, want combined continuation", deps.savedContent)
	}
	if deps.continuations != 1 || !deps.recoveryCompleted {
		t.Fatalf("continuations/completed = %d/%v, want 1/true", deps.continuations, deps.recoveryCompleted)
	}
}

type retryOutputLimitDeps struct {
	client            llms.LLMClient
	recoveries        int
	continuations     int
	recoveryCompleted bool
	errorEvents       int
	savedContent      string
	savedThinking     string
	done              bool
	doneUsage         *llms.Usage
}

func (d *retryOutputLimitDeps) BuildMessages(context.Context) ([]llms.Message, []llms.ToolDef, error) {
	return []llms.Message{{Role: "user", Content: "完成任务"}}, nil, nil
}

func (d *retryOutputLimitDeps) PrepareLLM(context.Context, []llms.Message, []llms.ToolDef) (llms.LLMClient, *llms.ChatRequest, error) {
	return d.client, &llms.ChatRequest{Messages: []llms.Message{{Role: "user", Content: "完成任务"}}, MaxTokens: 8196}, nil
}

func (d *retryOutputLimitDeps) SendEvent(event string, _ interface{}) {
	if event == EventError {
		d.errorEvents++
	}
}

func (d *retryOutputLimitDeps) SaveAssistantMessage(_ context.Context, content string, thinking string, _ *llms.Usage) error {
	d.savedContent = content
	d.savedThinking = thinking
	return nil
}

func (d *retryOutputLimitDeps) SaveAssistantMessageWithToolCalls(context.Context, string, string, []llms.ToolCall, *llms.Usage) error {
	return nil
}

func (d *retryOutputLimitDeps) ExecuteToolCalls(context.Context, []llms.ToolCall, int, func(string, interface{})) ([]ToolCallSummary, error) {
	return nil, nil
}

func (d *retryOutputLimitDeps) OnDone(_ []ToolCallSummary, usage *llms.Usage) {
	d.done = true
	d.doneUsage = usage
}

func (d *retryOutputLimitDeps) RequestOutputLimitRecovery(context.Context, string) bool {
	d.recoveries++
	return d.recoveries == 1
}

func (d *retryOutputLimitDeps) RequestOutputContinuation(_ context.Context, partialContent string) bool {
	d.continuations++
	return strings.TrimSpace(partialContent) != "" && d.continuations <= 4
}

func (d *retryOutputLimitDeps) CompleteOutputRecovery() {
	d.recoveryCompleted = true
}

type retryOutputLimitClient struct {
	calls int
}

func (c *retryOutputLimitClient) Chat(context.Context, *llms.ChatRequest) (*llms.ChatResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *retryOutputLimitClient) ChatStream(context.Context, *llms.ChatRequest) (<-chan *llms.StreamChunk, error) {
	c.calls++
	stream := make(chan *llms.StreamChunk, 2)
	if c.calls == 1 {
		stream <- &llms.StreamChunk{ReasoningContent: "冗长思考", FinishReason: "length"}
		stream <- &llms.StreamChunk{Done: true, FinishReason: "length", Usage: &llms.Usage{CompletionTokens: 8196, TotalTokens: 8196}}
	} else {
		stream <- &llms.StreamChunk{ReasoningContent: "精简思考"}
		stream <- &llms.StreamChunk{Content: "完成", Done: true, Usage: &llms.Usage{CompletionTokens: 10, TotalTokens: 10}}
	}
	close(stream)
	return stream, nil
}

func (c *retryOutputLimitClient) GetModelName() string {
	return "test-reasoning-model"
}

type continuationOutputClient struct {
	calls int
}

func (c *continuationOutputClient) Chat(context.Context, *llms.ChatRequest) (*llms.ChatResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *continuationOutputClient) ChatStream(context.Context, *llms.ChatRequest) (<-chan *llms.StreamChunk, error) {
	c.calls++
	stream := make(chan *llms.StreamChunk, 2)
	switch c.calls {
	case 1:
		stream <- &llms.StreamChunk{ReasoningContent: "超长思考", FinishReason: "length"}
		stream <- &llms.StreamChunk{Done: true, FinishReason: "length"}
	case 2:
		stream <- &llms.StreamChunk{Content: "第一段", FinishReason: "length"}
		stream <- &llms.StreamChunk{Done: true, FinishReason: "length"}
	default:
		stream <- &llms.StreamChunk{Content: "第二段"}
		stream <- &llms.StreamChunk{Done: true}
	}
	close(stream)
	return stream, nil
}

func (c *continuationOutputClient) GetModelName() string {
	return "continuation-test-model"
}

type retryContextWindowDeps struct {
	client         *retryContextWindowClient
	reductions     int
	done           bool
	attemptActions []string
}

func (d *retryContextWindowDeps) BuildMessages(context.Context) ([]llms.Message, []llms.ToolDef, error) {
	return []llms.Message{{Role: "user", Content: "继续处理"}}, nil, nil
}

func (d *retryContextWindowDeps) PrepareLLM(context.Context, []llms.Message, []llms.ToolDef) (llms.LLMClient, *llms.ChatRequest, error) {
	return d.client, &llms.ChatRequest{Messages: []llms.Message{{Role: "user", Content: "继续处理"}}}, nil
}

func (d *retryContextWindowDeps) SendEvent(event string, data interface{}) {
	if event != EventGenerationAttempt {
		return
	}
	if payload, ok := data.(*dto.WorkspaceStreamGenerationAttempt); ok {
		d.attemptActions = append(d.attemptActions, payload.Action)
	}
}

func (d *retryContextWindowDeps) SaveAssistantMessage(context.Context, string, string, *llms.Usage) error {
	return nil
}

func (d *retryContextWindowDeps) SaveAssistantMessageWithToolCalls(context.Context, string, string, []llms.ToolCall, *llms.Usage) error {
	return nil
}

func (d *retryContextWindowDeps) ExecuteToolCalls(context.Context, []llms.ToolCall, int, func(string, interface{})) ([]ToolCallSummary, error) {
	return nil, nil
}

func (d *retryContextWindowDeps) OnDone([]ToolCallSummary, *llms.Usage) {
	d.done = true
}

func (d *retryContextWindowDeps) RequestContextReduction(context.Context, string) bool {
	d.reductions++
	return d.reductions <= 1
}

type retryContextWindowClient struct {
	calls int
}

func (c *retryContextWindowClient) Chat(context.Context, *llms.ChatRequest) (*llms.ChatResponse, error) {
	return nil, errors.New("not implemented")
}

func (c *retryContextWindowClient) ChatStream(context.Context, *llms.ChatRequest) (<-chan *llms.StreamChunk, error) {
	c.calls++
	if c.calls == 1 {
		stream := make(chan *llms.StreamChunk, 2)
		stream <- &llms.StreamChunk{Content: "半截内容"}
		stream <- &llms.StreamChunk{Error: "OpenAI API 错误: invalid params, context window exceeds limit (2013)"}
		close(stream)
		return stream, nil
	}
	stream := make(chan *llms.StreamChunk, 2)
	stream <- &llms.StreamChunk{Content: "完成"}
	stream <- &llms.StreamChunk{Done: true}
	close(stream)
	return stream, nil
}

func (c *retryContextWindowClient) GetModelName() string {
	return "test"
}

func TestAppendToolCallArgsIgnoresDeltaWhenCurrentIsValidJSON(t *testing.T) {
	got := appendToolCallArgs(`{"id":1}`, `,"extra":2`)
	want := `{"id":1}`
	if got != want {
		t.Fatalf("appendToolCallArgs() = %q, want %q", got, want)
	}
}

func TestMergeToolCallsUsesOpenAIStreamIndex(t *testing.T) {
	idx0 := 0
	idx1 := 1
	makeCall := func(index *int, id, name, args string) llms.ToolCallDelta {
		tc := llms.ToolCallDelta{ID: id, Type: "function", Index: index}
		tc.Function.Name = name
		tc.Function.Arguments = args
		return tc
	}

	var all []llms.ToolCall
	indexByID := map[string]int{}
	all, indexByID = mergeToolCallDeltas([]llms.ToolCallDelta{
		makeCall(&idx0, "call_a", "first_tool", `{"a":`),
		makeCall(&idx1, "call_b", "second_tool", `{"b":`),
	}, all, indexByID)
	all, indexByID = mergeToolCallDeltas([]llms.ToolCallDelta{
		makeCall(&idx0, "", "", `1}`),
		makeCall(&idx1, "", "", `2}`),
	}, all, indexByID)

	if len(all) != 2 {
		t.Fatalf("len(all) = %d, want 2", len(all))
	}
	if all[0].ID != "call_a" || all[0].Function.Name != "first_tool" || all[0].Function.Arguments != `{"a":1}` {
		t.Fatalf("first tool call not merged by index: %#v", all[0])
	}
	if all[1].ID != "call_b" || all[1].Function.Name != "second_tool" || all[1].Function.Arguments != `{"b":2}` {
		t.Fatalf("second tool call not merged by index: %#v", all[1])
	}
	if indexByID["call_a"] != 0 || indexByID["call_b"] != 1 {
		t.Fatalf("indexByID = %#v, want call_a=0 call_b=1", indexByID)
	}
}

func TestProcessStreamChunksEmitsStableToolCallIdentity(t *testing.T) {
	idx0 := 0
	makeCall := func(index *int, id, name, args string) llms.ToolCallDelta {
		tc := llms.ToolCallDelta{ID: id, Type: "function", Index: index}
		tc.Function.Name = name
		tc.Function.Arguments = args
		return tc
	}

	stream := make(chan *llms.StreamChunk, 2)
	stream <- &llms.StreamChunk{ToolCallDeltas: []llms.ToolCallDelta{
		makeCall(&idx0, "", "run_python", `{"code":"print(1)"}`),
	}}
	stream <- &llms.StreamChunk{ToolCallDeltas: []llms.ToolCallDelta{
		makeCall(&idx0, "call_1", "", ""),
	}}
	close(stream)

	var events []dto.WorkspaceStreamToolCallDeltaData
	_, _, calls, _, err := processStreamChunks(context.Background(), stream, func(event string, data interface{}) {
		if event != EventToolCallsStreamDelta {
			return
		}
		payload, ok := data.(*dto.WorkspaceStreamToolCallDeltaData)
		if !ok {
			t.Fatalf("delta payload type = %T, want *dto.WorkspaceStreamToolCallDeltaData", data)
		}
		events = append(events, *payload)
	}, 3)
	if err != nil {
		t.Fatalf("processStreamChunks returned error: %v", err)
	}
	if len(calls) != 1 || calls[0].ID != "call_1" {
		t.Fatalf("calls = %#v, want one call with id call_1", calls)
	}
	if len(events) != 1 || len(events[0].Updates) != 1 {
		t.Fatalf("events = %#v, want one delta update", events)
	}
	update := events[0].Updates[0]
	if update.ID != "call_1" || update.Index != 0 || update.Round != 3 || update.Name != "run_python" {
		t.Fatalf("delta update identity = %#v, want id/index/round/name", update)
	}
}

func TestProcessStreamChunksReturnsFinalUsage(t *testing.T) {
	stream := make(chan *llms.StreamChunk, 2)
	stream <- &llms.StreamChunk{Content: "ok"}
	stream <- &llms.StreamChunk{Usage: &llms.Usage{
		PromptTokens:         1200,
		CompletionTokens:     80,
		TotalTokens:          1280,
		CachedTokens:         1024,
		CachedTokensReported: true,
	}}
	close(stream)

	content, thinkingContent, calls, usage, err := processStreamChunks(context.Background(), stream, func(string, interface{}) {}, 0)
	if err != nil {
		t.Fatalf("processStreamChunks returned error: %v", err)
	}
	if content != "ok" || thinkingContent != "" || len(calls) != 0 {
		t.Fatalf("content/thinking/calls = %q/%q/%#v, want ok/empty/no calls", content, thinkingContent, calls)
	}
	if usage == nil || usage.CachedTokens != 1024 || usage.TotalTokens != 1280 || !usage.CachedTokensReported {
		t.Fatalf("usage = %#v, want cached 1024 total 1280", usage)
	}
}

func TestProcessStreamChunksFiltersThinkTagsAcrossChunks(t *testing.T) {
	stream := make(chan *llms.StreamChunk, 4)
	stream <- &llms.StreamChunk{Content: "<thi"}
	stream <- &llms.StreamChunk{Content: "nk>内部思考"}
	stream <- &llms.StreamChunk{Content: "</thi"}
	stream <- &llms.StreamChunk{Content: "nk>\n\n最终答案"}
	close(stream)

	var sent strings.Builder
	var thinking strings.Builder
	content, thinkingContent, calls, _, err := processStreamChunks(context.Background(), stream, func(event string, data interface{}) {
		if event == EventThinking {
			payload, ok := data.(*thinkingData)
			if !ok {
				t.Fatalf("thinking payload type = %T, want *thinkingData", data)
			}
			thinking.WriteString(payload.Content)
			return
		}
		if event != EventContent {
			return
		}
		payload, ok := data.(*contentData)
		if !ok {
			t.Fatalf("content payload type = %T, want *contentData", data)
		}
		sent.WriteString(payload.Content)
	}, 0)
	if err != nil {
		t.Fatalf("processStreamChunks returned error: %v", err)
	}
	if content != "最终答案" || thinkingContent != "内部思考" || len(calls) != 0 {
		t.Fatalf("content/thinking/calls = %q/%q/%#v, want final answer/internal thinking/no calls", content, thinkingContent, calls)
	}
	if strings.Contains(sent.String(), "内部思考") || sent.String() != "\n\n最终答案" {
		t.Fatalf("sent content = %q, want only final answer chunk", sent.String())
	}
	if thinking.String() != "内部思考" {
		t.Fatalf("thinking content = %q, want internal thinking", thinking.String())
	}
}

func TestProcessStreamChunksFiltersStrayThinkCloseTag(t *testing.T) {
	stream := make(chan *llms.StreamChunk, 1)
	stream <- &llms.StreamChunk{Content: "</think>\n最终答案"}
	close(stream)

	var sent strings.Builder
	content, _, _, _, err := processStreamChunks(context.Background(), stream, func(event string, data interface{}) {
		if event != EventContent {
			return
		}
		payload, ok := data.(*contentData)
		if !ok {
			t.Fatalf("content payload type = %T, want *contentData", data)
		}
		sent.WriteString(payload.Content)
	}, 0)
	if err != nil {
		t.Fatalf("processStreamChunks returned error: %v", err)
	}
	if content != "最终答案" || sent.String() != "\n最终答案" {
		t.Fatalf("content/sent = %q/%q, want final answer only", content, sent.String())
	}
}

func TestProcessStreamChunksReportsLengthInsideThink(t *testing.T) {
	stream := make(chan *llms.StreamChunk, 1)
	stream <- &llms.StreamChunk{Content: "<think>还在思考", FinishReason: "length"}
	close(stream)

	contentEvents := 0
	thinkingEvents := 0
	content, thinkingContent, calls, _, err := processStreamChunks(context.Background(), stream, func(event string, data interface{}) {
		if event == EventContent {
			contentEvents++
		}
		if event == EventThinking {
			thinkingEvents++
		}
	}, 0)
	if err == nil || !strings.Contains(err.Error(), "思考阶段") {
		t.Fatalf("err = %v, want thinking length error", err)
	}
	if content != "" || thinkingContent != "还在思考" || len(calls) != 0 || contentEvents != 0 || thinkingEvents != 1 {
		t.Fatalf("content/thinking/calls/content events/thinking events = %q/%q/%#v/%d/%d, want empty/thinking/no calls/no content events/one thinking event", content, thinkingContent, calls, contentEvents, thinkingEvents)
	}
}

func TestProcessStreamChunksReportsLengthInsideReasoningContent(t *testing.T) {
	stream := make(chan *llms.StreamChunk, 1)
	stream <- &llms.StreamChunk{ReasoningContent: "内部思考", FinishReason: "length"}
	close(stream)

	contentEvents := 0
	thinkingEvents := 0
	content, thinkingContent, calls, _, err := processStreamChunks(context.Background(), stream, func(event string, data interface{}) {
		if event == EventContent {
			contentEvents++
		}
		if event == EventThinking {
			thinkingEvents++
		}
	}, 0)
	if err == nil || !strings.Contains(err.Error(), "思考阶段") {
		t.Fatalf("err = %v, want thinking length error", err)
	}
	if content != "" || thinkingContent != "内部思考" || len(calls) != 0 || contentEvents != 0 || thinkingEvents != 1 {
		t.Fatalf("content/thinking/calls/content events/thinking events = %q/%q/%#v/%d/%d, want empty/thinking/no calls/no content events/one thinking event", content, thinkingContent, calls, contentEvents, thinkingEvents)
	}
}

func TestProcessStreamChunksReturnsLLMErrorWithoutEmittingErrorEvent(t *testing.T) {
	stream := make(chan *llms.StreamChunk, 1)
	stream <- &llms.StreamChunk{Error: "unexpected end of JSON input"}
	close(stream)

	errorEvents := 0
	_, _, _, _, err := processStreamChunks(context.Background(), stream, func(event string, data interface{}) {
		if event == EventError {
			errorEvents++
		}
	}, 0)
	if err == nil || err.Error() != "LLM 流式错误: unexpected end of JSON input" {
		t.Fatalf("err = %v, want LLM stream error", err)
	}
	if errorEvents != 0 {
		t.Fatalf("error events = %d, want 0 from processStreamChunks", errorEvents)
	}
}

func TestProcessStreamChunksRejectsMalformedToolCallArguments(t *testing.T) {
	idx0 := 0
	makeCall := func(index *int, id, name, args string) llms.ToolCallDelta {
		tc := llms.ToolCallDelta{ID: id, Type: "function", Index: index}
		tc.Function.Name = name
		tc.Function.Arguments = args
		return tc
	}

	stream := make(chan *llms.StreamChunk, 1)
	stream <- &llms.StreamChunk{ToolCallDeltas: []llms.ToolCallDelta{
		makeCall(&idx0, "call_bad", "read_file", `{"file_name":"purchase_inbound.go"}</invoke>`),
	}}
	close(stream)

	_, _, calls, _, err := processStreamChunks(context.Background(), stream, func(string, interface{}) {}, 0)
	if err == nil || !strings.Contains(err.Error(), "参数不是合法 JSON") {
		t.Fatalf("err = %v, want malformed JSON tool call error", err)
	}
	if len(calls) != 0 {
		t.Fatalf("calls = %#v, want no executable calls", calls)
	}
}

func TestMergeToolCallsPrefersKnownIDWhenStreamIndexIsWrong(t *testing.T) {
	idx0 := 0
	idx1 := 1
	makeCall := func(index *int, id, name, args string) llms.ToolCallDelta {
		tc := llms.ToolCallDelta{ID: id, Type: "function", Index: index}
		tc.Function.Name = name
		tc.Function.Arguments = args
		return tc
	}

	var all []llms.ToolCall
	indexByID := map[string]int{}
	all, indexByID = mergeToolCallDeltas([]llms.ToolCallDelta{
		makeCall(&idx0, "call_a", "first_tool", `{"a":`),
		makeCall(&idx1, "call_b", "second_tool", `{"b":`),
	}, all, indexByID)
	all, indexByID = mergeToolCallDeltas([]llms.ToolCallDelta{
		makeCall(&idx0, "call_b", "", `2}`),
	}, all, indexByID)

	if all[0].Function.Arguments != `{"a":` {
		t.Fatalf("first tool was corrupted by wrong index: %#v", all[0])
	}
	if all[1].Function.Arguments != `{"b":2}` {
		t.Fatalf("second tool did not merge by known id: %#v", all[1])
	}
	if indexByID["call_b"] != 1 {
		t.Fatalf("indexByID[call_b] = %d, want 1", indexByID["call_b"])
	}
}

func TestMergeToolCallsDoesNotAppendAnonymousArgsWhenAmbiguous(t *testing.T) {
	makeCall := func(id, name, args string) llms.ToolCallDelta {
		tc := llms.ToolCallDelta{ID: id, Type: "function"}
		tc.Function.Name = name
		tc.Function.Arguments = args
		return tc
	}

	var all []llms.ToolCall
	indexByID := map[string]int{}
	all, indexByID = mergeToolCallDeltas([]llms.ToolCallDelta{
		makeCall("call_a", "first_tool", `{"a":`),
		makeCall("call_b", "second_tool", `{"b":`),
	}, all, indexByID)
	all, _ = mergeToolCallDeltas([]llms.ToolCallDelta{
		makeCall("", "", `1}`),
	}, all, indexByID)

	if all[0].Function.Arguments != `{"a":` || all[1].Function.Arguments != `{"b":` {
		t.Fatalf("anonymous args were appended despite ambiguous target: %#v", all)
	}
}

func TestNormalizeToolCallsDropsBlankPlaceholdersAndAssignsMissingID(t *testing.T) {
	call := llms.ToolCall{Type: "function"}
	call.Function.Name = "write_file"
	call.Function.Arguments = `{}`

	got := normalizeToolCalls(context.Background(), []llms.ToolCall{
		{Type: "function"},
		call,
	})

	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1: %#v", len(got), got)
	}
	if got[0].ID != "call_local_1" {
		t.Fatalf("generated id = %q, want call_local_1", got[0].ID)
	}
}
