package llms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

const defaultAnthropicModel = "claude-sonnet-4-5"

type AnthropicMessagesClient struct {
	apiKey   string
	model    string
	options  *ClientOptions
	client   *http.Client
	endpoint string
}

func NewAnthropicMessagesClientWithOptions(apiKey string, options *ClientOptions) *AnthropicMessagesClient {
	options = normalizeClientOptions(options)
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = defaultAnthropicModel
	}
	endpoint := buildEndpoint(options.BaseURL, options.EndpointPath, ProtocolAnthropicMessages)
	return &AnthropicMessagesClient{
		apiKey:   strings.TrimSpace(apiKey),
		model:    model,
		options:  options,
		client:   createHTTPClient(options, options.Timeout),
		endpoint: endpoint,
	}
}

func (c *AnthropicMessagesClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	stream, err := c.ChatStream(ctx, req)
	if err != nil {
		return nil, err
	}
	return collectStreamResponse(stream)
}

func (c *AnthropicMessagesClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	if err := validateRequest(ctx, c.apiKey, req); err != nil {
		return nil, err
	}
	body, err := c.buildRequestBody(req, true)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, resolveRequestTimeout(c.options, req))
	headers := applyAuthHeader(c.options.Headers, c.apiKey, c.options.AuthScheme, ProtocolAnthropicMessages)
	apiVersion := strings.TrimSpace(c.options.APIVersion)
	if apiVersion == "" {
		apiVersion = defaultAPIVersion(ProtocolAnthropicMessages)
	}
	headers["anthropic-version"] = apiVersion
	resp, err := postJSON(callCtx, c.client, c.endpoint, headers, body)
	if err != nil {
		cancel()
		return nil, err
	}
	ch := make(chan *StreamChunk)
	go func() {
		defer close(ch)
		defer cancel()
		defer resp.Body.Close()
		readAnthropicStream(callCtx, resp.Body, ch)
	}()
	return ch, nil
}

func (c *AnthropicMessagesClient) GetModelName() string {
	return c.model
}

func (c *AnthropicMessagesClient) buildRequestBody(req *ChatRequest, stream bool) (map[string]interface{}, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	system, messages, err := anthropicMessagesFromLLMMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"model":      model,
		"max_tokens": req.MaxTokens,
		"messages":   messages,
		"stream":     stream,
	}
	if req.MaxTokens <= 0 {
		body["max_tokens"] = 4096
	}
	if system != "" {
		body["system"] = system
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		body["tools"] = anthropicTools(req.Tools)
	}
	if req.ToolChoice != nil {
		body["tool_choice"] = anthropicToolChoice(req.ToolChoice)
	}
	return body, nil
}

func anthropicMessagesFromLLMMessages(messages []Message) (string, []map[string]interface{}, error) {
	var system []string
	out := make([]map[string]interface{}, 0, len(messages))
	for i, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "system", "developer":
			if strings.TrimSpace(msg.Content) != "" {
				system = append(system, msg.Content)
			}
		case "user":
			out = append(out, map[string]interface{}{"role": "user", "content": msg.Content})
		case "assistant":
			blocks := make([]map[string]interface{}, 0, 1+len(msg.ToolCalls))
			if strings.TrimSpace(msg.Content) != "" {
				blocks = append(blocks, map[string]interface{}{"type": "text", "text": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				blocks = append(blocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    tc.ID,
					"name":  tc.Function.Name,
					"input": anthropicToolInput(tc.Function.Arguments),
				})
			}
			if len(blocks) == 0 {
				return "", nil, fmt.Errorf("消息 %d 的 assistant content 或 tool_calls 不能同时为空", i)
			}
			out = append(out, map[string]interface{}{"role": "assistant", "content": blocks})
		case "tool":
			out = append(out, map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{"type": "tool_result", "tool_use_id": msg.ToolCallID, "content": msg.Content},
				},
			})
		default:
			return "", nil, fmt.Errorf("消息 %d 的 role %q 不支持，Anthropic Messages 仅支持 system/developer/user/assistant/tool", i, msg.Role)
		}
	}
	return strings.Join(system, "\n\n"), out, nil
}

func anthropicToolInput(arguments string) interface{} {
	arguments = strings.TrimSpace(arguments)
	if arguments == "" {
		return map[string]interface{}{}
	}
	var obj interface{}
	if err := json.Unmarshal([]byte(arguments), &obj); err != nil {
		return map[string]interface{}{"_raw": arguments}
	}
	return obj
}

func anthropicTools(tools []ToolDef) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		item := map[string]interface{}{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": tool.Function.Parameters,
		}
		if tool.Function.Strict != nil {
			item["strict"] = *tool.Function.Strict
		}
		out = append(out, item)
	}
	return out
}

func anthropicToolChoice(value interface{}) interface{} {
	switch v := value.(type) {
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "", "auto":
			return map[string]interface{}{"type": "auto"}
		case "none":
			return map[string]interface{}{"type": "none"}
		case "required":
			return map[string]interface{}{"type": "any"}
		default:
			return map[string]interface{}{"type": "tool", "name": v}
		}
	default:
		return value
	}
}

func readAnthropicStream(ctx context.Context, body interface {
	Read([]byte) (int, error)
}, chunkChan chan<- *StreamChunk) {
	state := newAnthropicStreamState()
	err := readSSE(ctx, body, func(_, data string) error {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return err
		}
		for _, chunk := range state.handle(payload) {
			chunkChan <- chunk
		}
		return nil
	})
	if err != nil {
		chunkChan <- &StreamChunk{Error: err.Error(), Done: true, Usage: state.usage}
		return
	}
	chunkChan <- &StreamChunk{Done: true, FinalToolCalls: state.finalToolCalls(), FinishReason: state.finishReason, Usage: state.usage}
}

type anthropicStreamState struct {
	calls        map[int]*ToolCall
	usage        *Usage
	inputTokens  int
	outputTokens int
	finishReason string
}

func newAnthropicStreamState() *anthropicStreamState {
	return &anthropicStreamState{calls: map[int]*ToolCall{}}
}

func (s *anthropicStreamState) handle(payload map[string]interface{}) []*StreamChunk {
	typ, _ := payload["type"].(string)
	switch typ {
	case "message_start":
		if msg, ok := payload["message"].(map[string]interface{}); ok {
			s.noteUsage(msg["usage"])
		}
	case "content_block_start":
		idx := intFromAny(payload["index"])
		if block, ok := payload["content_block"].(map[string]interface{}); ok {
			if blockType, _ := block["type"].(string); blockType == "tool_use" {
				call := s.ensureCall(idx)
				if id, _ := block["id"].(string); id != "" {
					call.ID = id
				}
				if name, _ := block["name"].(string); name != "" {
					call.Function.Name = name
				}
				if input, ok := block["input"].(map[string]interface{}); ok && len(input) > 0 {
					if b, err := json.Marshal(input); err == nil {
						call.Function.Arguments = string(b)
					}
				}
			}
		}
	case "content_block_delta":
		idx := intFromAny(payload["index"])
		delta, _ := payload["delta"].(map[string]interface{})
		deltaType, _ := delta["type"].(string)
		switch deltaType {
		case "text_delta":
			if text, _ := delta["text"].(string); text != "" {
				return []*StreamChunk{{Content: text}}
			}
		case "thinking_delta":
			if text, _ := delta["thinking"].(string); text != "" {
				return []*StreamChunk{{ReasoningContent: text}}
			}
		case "input_json_delta":
			partial, _ := delta["partial_json"].(string)
			call := s.ensureCall(idx)
			call.Function.Arguments += partial
			return []*StreamChunk{{ToolCallDeltas: []ToolCallDelta{newToolCallDelta(&idx, call.ID, call.Type, call.Function.Name, partial)}}}
		}
	case "content_block_stop":
		idx := intFromAny(payload["index"])
		if call, ok := s.calls[idx]; ok && call != nil && call.ID == "" {
			call.ID = "toolu_local_" + strconv.Itoa(idx+1)
		}
	case "message_delta":
		if delta, ok := payload["delta"].(map[string]interface{}); ok {
			if stopReason, _ := delta["stop_reason"].(string); stopReason != "" {
				s.finishReason = stopReason
			}
		}
		s.noteUsage(payload["usage"])
	case "error":
		if errObj, ok := payload["error"].(map[string]interface{}); ok {
			msg, _ := errObj["message"].(string)
			if msg != "" {
				return []*StreamChunk{{Error: msg, Done: true, Usage: s.usage}}
			}
		}
	}
	return nil
}

func (s *anthropicStreamState) ensureCall(idx int) *ToolCall {
	if idx < 0 {
		idx = len(s.calls)
	}
	if call, ok := s.calls[idx]; ok {
		return call
	}
	call := newToolCall("", "function", "", "")
	s.calls[idx] = &call
	return &call
}

func (s *anthropicStreamState) noteUsage(raw interface{}) {
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	if input := intFromAny(obj["input_tokens"]); input > 0 {
		s.inputTokens = input
	}
	if output := intFromAny(obj["output_tokens"]); output > 0 {
		s.outputTokens = output
	}
	cacheRead := intFromAny(obj["cache_read_input_tokens"])
	cacheCreate := intFromAny(obj["cache_creation_input_tokens"])
	total := s.inputTokens + s.outputTokens
	if total == 0 {
		return
	}
	s.usage = &Usage{
		PromptTokens:         s.inputTokens,
		CompletionTokens:     s.outputTokens,
		TotalTokens:          total,
		CachedTokens:         cacheRead + cacheCreate,
		CachedTokensReported: cacheRead > 0 || cacheCreate > 0,
	}
}

func (s *anthropicStreamState) finalToolCalls() []ToolCall {
	if len(s.calls) == 0 {
		return nil
	}
	out := make([]ToolCall, 0, len(s.calls))
	for i := 0; i < len(s.calls); i++ {
		call, ok := s.calls[i]
		if !ok || call == nil {
			continue
		}
		if call.ID == "" && call.Function.Name == "" && call.Function.Arguments == "" {
			continue
		}
		if call.ID == "" {
			call.ID = "toolu_local_" + strconv.Itoa(i+1)
		}
		if strings.TrimSpace(call.Function.Arguments) == "" {
			call.Function.Arguments = "{}"
		}
		out = append(out, *call)
	}
	return out
}

var _ LLMClient = (*AnthropicMessagesClient)(nil)
