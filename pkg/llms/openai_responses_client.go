package llms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const defaultOpenAIResponsesModel = "gpt-4o-mini"

type OpenAIResponsesClient struct {
	apiKey    string
	model     string
	options   *ClientOptions
	client    *http.Client
	endpoints []string
}

func NewOpenAIResponsesClientWithOptions(apiKey string, options *ClientOptions) *OpenAIResponsesClient {
	options = normalizeClientOptions(options)
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = defaultOpenAIResponsesModel
	}
	endpoints := openAIResponsesEndpointCandidates(options.BaseURL, options.EndpointPath)
	return &OpenAIResponsesClient{
		apiKey:    strings.TrimSpace(apiKey),
		model:     model,
		options:   options,
		client:    createHTTPClient(options, options.Timeout),
		endpoints: endpoints,
	}
}

func (c *OpenAIResponsesClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	stream, err := c.ChatStream(ctx, req)
	if err != nil {
		return nil, err
	}
	return collectStreamResponse(stream)
}

func (c *OpenAIResponsesClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	if err := validateRequest(ctx, c.apiKey, req); err != nil {
		return nil, err
	}
	body, err := c.buildRequestBody(req, true)
	if err != nil {
		return nil, err
	}
	callCtx, cancel := context.WithTimeout(ctx, resolveRequestTimeout(c.options, req))
	headers := applyAuthHeader(c.options.Headers, c.apiKey, c.options.AuthScheme, ProtocolOpenAIResponses)
	resp, err := c.postResponsesJSON(callCtx, headers, body)
	if err != nil {
		cancel()
		return nil, err
	}
	ch := make(chan *StreamChunk)
	go func() {
		defer close(ch)
		defer cancel()
		defer resp.Body.Close()
		readOpenAIResponsesStream(callCtx, resp.Body, ch)
	}()
	return ch, nil
}

func (c *OpenAIResponsesClient) postResponsesJSON(ctx context.Context, headers map[string]string, body interface{}) (*http.Response, error) {
	endpoints := c.endpoints
	if len(endpoints) == 0 {
		endpoints = openAIResponsesEndpointCandidates(c.options.BaseURL, c.options.EndpointPath)
	}
	var lastErr error
	for i, endpoint := range endpoints {
		resp, err := postJSON(ctx, c.client, endpoint, headers, body)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !shouldRetryOpenAIResponsesEndpoint(err) || i == len(endpoints)-1 {
			return nil, err
		}
	}
	return nil, lastErr
}

func openAIResponsesEndpointCandidates(baseURL, endpointPath string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, 3)
	add := func(endpoint string) {
		endpoint = strings.TrimSpace(endpoint)
		if endpoint == "" {
			return
		}
		if _, ok := seen[endpoint]; ok {
			return
		}
		seen[endpoint] = struct{}{}
		out = append(out, endpoint)
	}

	add(buildEndpoint(baseURL, endpointPath, ProtocolOpenAIResponses))

	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	endpointPath = strings.TrimSpace(endpointPath)
	if endpointPath == "" {
		endpointPath = defaultEndpointPath(ProtocolOpenAIResponses)
	}
	if baseURL == "" || strings.HasPrefix(endpointPath, "http://") || strings.HasPrefix(endpointPath, "https://") {
		return out
	}

	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return out
	}
	basePath := strings.TrimRight(parsed.Path, "/")
	basePathLower := strings.ToLower(basePath)
	if basePathLower != "/api/v1" && !strings.HasSuffix(basePathLower, "/api/v1") {
		return out
	}

	root := *parsed
	root.Path = strings.TrimRight(strings.TrimSuffix(basePath, "/api/v1"), "/")
	root.RawQuery = ""
	root.Fragment = ""
	if root.Path == "" {
		root.Path = ""
	}
	rootBase := strings.TrimRight(root.String(), "/")
	add(buildEndpoint(rootBase, endpointPath, ProtocolOpenAIResponses))
	if strings.EqualFold(endpointPath, "/responses") {
		add(buildEndpoint(rootBase, "/v1/responses", ProtocolOpenAIResponses))
	}
	return out
}

func shouldRetryOpenAIResponsesEndpoint(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "HTTP 404")
}

func (c *OpenAIResponsesClient) GetModelName() string {
	return c.model
}

func (c *OpenAIResponsesClient) buildRequestBody(req *ChatRequest, stream bool) (map[string]interface{}, error) {
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	instructions, input, err := responsesInputFromMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{
		"model":  model,
		"input":  input,
		"stream": stream,
		"store":  false,
	}
	if instructions != "" {
		body["instructions"] = instructions
	}
	if req.MaxTokens > 0 {
		body["max_output_tokens"] = req.MaxTokens
	}
	if reasoningEffort := normalizeReasoningEffort(req.ReasoningEffort); reasoningEffort != "" {
		body["reasoning"] = map[string]interface{}{"effort": reasoningEffort}
	}
	if verbosity := normalizeVerbosity(req.Verbosity); verbosity != "" {
		body["text"] = map[string]interface{}{"verbosity": verbosity}
	}
	if req.Temperature > 0 {
		body["temperature"] = req.Temperature
	}
	if len(req.Tools) > 0 {
		body["tools"] = responsesTools(req.Tools)
	}
	if req.ToolChoice != nil {
		body["tool_choice"] = req.ToolChoice
	}
	return body, nil
}

func responsesInputFromMessages(messages []Message) (string, []interface{}, error) {
	var instructions []string
	input := make([]interface{}, 0, len(messages))
	for i, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		switch role {
		case "system", "developer":
			if strings.TrimSpace(msg.Content) != "" {
				instructions = append(instructions, msg.Content)
			}
		case "user":
			input = append(input, map[string]interface{}{"role": "user", "content": msg.Content})
		case "assistant":
			if strings.TrimSpace(msg.Content) != "" {
				input = append(input, map[string]interface{}{"role": "assistant", "content": msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				input = append(input, map[string]interface{}{
					"type":      "function_call",
					"call_id":   tc.ID,
					"name":      tc.Function.Name,
					"arguments": sanitizeToolCallArguments(tc.Function.Arguments),
				})
			}
		case "tool":
			input = append(input, map[string]interface{}{
				"type":    "function_call_output",
				"call_id": msg.ToolCallID,
				"output":  msg.Content,
			})
		default:
			return "", nil, fmt.Errorf("消息 %d 的 role %q 不支持，OpenAI Responses 仅支持 system/developer/user/assistant/tool", i, msg.Role)
		}
	}
	return strings.Join(instructions, "\n\n"), input, nil
}

func responsesTools(tools []ToolDef) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		item := map[string]interface{}{
			"type":        "function",
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"parameters":  tool.Function.Parameters,
		}
		if tool.Function.Strict != nil {
			item["strict"] = *tool.Function.Strict
		}
		out = append(out, item)
	}
	return out
}

func readOpenAIResponsesStream(ctx context.Context, body interface {
	Read([]byte) (int, error)
}, chunkChan chan<- *StreamChunk) {
	state := newResponsesStreamState()
	err := readSSE(ctx, body, func(_, data string) error {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return err
		}
		for _, chunk := range state.handle(payload) {
			if !sendStreamChunk(ctx, chunkChan, chunk) {
				return ctx.Err()
			}
		}
		return nil
	})
	if err != nil {
		sendStreamChunk(ctx, chunkChan, &StreamChunk{Error: err.Error(), Done: true, Usage: state.usage})
		return
	}
	sendStreamChunk(ctx, chunkChan, &StreamChunk{Done: true, FinalToolCalls: state.finalToolCalls(), FinishReason: state.finishReason, Usage: state.usage})
}

type responsesStreamState struct {
	calls        map[int]*ToolCall
	itemToIndex  map[string]int
	usage        *Usage
	finishReason string
}

func newResponsesStreamState() *responsesStreamState {
	return &responsesStreamState{
		calls:       map[int]*ToolCall{},
		itemToIndex: map[string]int{},
	}
}

func (s *responsesStreamState) handle(payload map[string]interface{}) []*StreamChunk {
	typ, _ := payload["type"].(string)
	switch typ {
	case "response.output_text.delta":
		if delta, _ := payload["delta"].(string); delta != "" {
			return []*StreamChunk{{Content: delta}}
		}
	case "response.reasoning_summary_text.delta", "response.reasoning_text.delta":
		if delta, _ := payload["delta"].(string); delta != "" {
			return []*StreamChunk{{ReasoningContent: delta}}
		}
	case "response.output_item.added", "response.output_item.done":
		if item, ok := payload["item"].(map[string]interface{}); ok {
			s.noteResponseItem(payload, item)
		}
	case "response.function_call_arguments.delta":
		idx := intFromAny(payload["output_index"])
		itemID, _ := payload["item_id"].(string)
		if itemID != "" {
			if existing, ok := s.itemToIndex[itemID]; ok {
				idx = existing
			} else {
				s.itemToIndex[itemID] = idx
			}
		}
		delta, _ := payload["delta"].(string)
		call := s.ensureCall(idx)
		call.Function.Arguments += delta
		return []*StreamChunk{{ToolCallDeltas: []ToolCallDelta{newToolCallDelta(&idx, call.ID, call.Type, call.Function.Name, delta)}}}
	case "response.function_call_arguments.done":
		idx := intFromAny(payload["output_index"])
		itemID, _ := payload["item_id"].(string)
		if itemID != "" {
			if existing, ok := s.itemToIndex[itemID]; ok {
				idx = existing
			} else {
				s.itemToIndex[itemID] = idx
			}
		}
		call := s.ensureCall(idx)
		if name, _ := payload["name"].(string); name != "" {
			call.Function.Name = name
		}
		if args, _ := payload["arguments"].(string); args != "" {
			call.Function.Arguments = args
		}
	case "response.completed", "response.incomplete", "response.failed":
		if response, ok := payload["response"].(map[string]interface{}); ok {
			s.noteUsage(response["usage"])
			s.finishReason = openAIResponsesFinishReason(response)
			if output, ok := response["output"].([]interface{}); ok {
				for i, raw := range output {
					if item, ok := raw.(map[string]interface{}); ok {
						s.noteResponseItem(map[string]interface{}{"output_index": i}, item)
					}
				}
			}
		}
	}
	return nil
}

func openAIResponsesFinishReason(response map[string]interface{}) string {
	status, _ := response["status"].(string)
	if status != "incomplete" {
		return status
	}
	details, _ := response["incomplete_details"].(map[string]interface{})
	reason, _ := details["reason"].(string)
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "max_output_tokens", "max_tokens", "length":
		return "max_output_tokens"
	default:
		return status
	}
}

func (s *responsesStreamState) noteResponseItem(event map[string]interface{}, item map[string]interface{}) {
	if itemType, _ := item["type"].(string); itemType != "function_call" {
		return
	}
	idx := intFromAny(event["output_index"])
	if id, _ := item["id"].(string); id != "" {
		if existing, ok := s.itemToIndex[id]; ok {
			idx = existing
		} else {
			s.itemToIndex[id] = idx
		}
	}
	call := s.ensureCall(idx)
	if callID, _ := item["call_id"].(string); callID != "" {
		call.ID = callID
	} else if id, _ := item["id"].(string); id != "" && call.ID == "" {
		call.ID = id
	}
	if name, _ := item["name"].(string); name != "" {
		call.Function.Name = name
	}
	switch args := item["arguments"].(type) {
	case string:
		if args != "" {
			call.Function.Arguments = args
		}
	case map[string]interface{}:
		if b, err := json.Marshal(args); err == nil {
			call.Function.Arguments = string(b)
		}
	}
}

func (s *responsesStreamState) ensureCall(idx int) *ToolCall {
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

func (s *responsesStreamState) noteUsage(raw interface{}) {
	obj, ok := raw.(map[string]interface{})
	if !ok {
		return
	}
	input := intFromAny(obj["input_tokens"])
	output := intFromAny(obj["output_tokens"])
	total := intFromAny(obj["total_tokens"])
	if total == 0 {
		total = input + output
	}
	if input == 0 && output == 0 && total == 0 {
		return
	}
	cached := 0
	cachedReported := false
	if details, ok := obj["input_tokens_details"].(map[string]interface{}); ok {
		cached = intFromAny(details["cached_tokens"])
		_, cachedReported = details["cached_tokens"]
	}
	s.usage = &Usage{
		PromptTokens:         input,
		CompletionTokens:     output,
		TotalTokens:          total,
		CachedTokens:         cached,
		CachedTokensReported: cachedReported,
	}
}

func (s *responsesStreamState) finalToolCalls() []ToolCall {
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
			call.ID = "call_response_" + strconv.Itoa(i+1)
		}
		out = append(out, *call)
	}
	return out
}

func intFromAny(value interface{}) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		n, _ := v.Int64()
		return int(n)
	default:
		return 0
	}
}

func collectStreamResponse(stream <-chan *StreamChunk) (*ChatResponse, error) {
	var content strings.Builder
	var toolCalls []ToolCall
	var usage *Usage
	for chunk := range stream {
		if chunk.Error != "" {
			return nil, errors.New(chunk.Error)
		}
		content.WriteString(chunk.Content)
		if len(chunk.FinalToolCalls) > 0 {
			toolCalls = chunk.FinalToolCalls
		}
		if chunk.Usage != nil {
			usage = chunk.Usage
		}
	}
	return &ChatResponse{Content: content.String(), ToolCalls: toolCalls, Usage: usage}, nil
}

var _ LLMClient = (*OpenAIResponsesClient)(nil)
