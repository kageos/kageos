package llms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/shared"
)

const defaultOpenAIModel = "gpt-4o-mini"

// OpenAIClient is the only concrete LLM client. BaseURL is only an OpenAI SDK
// endpoint override for proxies or controlled deployments, not a provider switch.
type OpenAIClient struct {
	apiKey  string
	model   string
	options *ClientOptions
	client  openai.Client
}

func NewOpenAIClient(apiKey string) *OpenAIClient {
	return NewOpenAIClientWithOptions(apiKey, DefaultClientOptions())
}

func NewOpenAIClientWithOptions(apiKey string, options *ClientOptions) *OpenAIClient {
	options = normalizeClientOptions(options)
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = defaultOpenAIModel
	}
	return &OpenAIClient{
		apiKey:  strings.TrimSpace(apiKey),
		model:   model,
		options: options,
		client:  newOpenAISDKClient(strings.TrimSpace(apiKey), options),
	}
}

func NewOpenAIClientFromEnv() *OpenAIClient {
	return NewOpenAIClient(os.Getenv("OPENAI_API_KEY"))
}

func newOpenAISDKClient(apiKey string, options *ClientOptions) openai.Client {
	options = normalizeClientOptions(options)
	requestOptions := []option.RequestOption{
		option.WithAPIKey(apiKey),
		option.WithHTTPClient(createHTTPClient(options, options.Timeout)),
		option.WithMaxRetries(options.MaxRetries),
	}
	if baseURL := strings.TrimSpace(options.BaseURL); baseURL != "" {
		requestOptions = append(requestOptions, option.WithBaseURL(baseURL))
	}
	if userAgent := strings.TrimSpace(options.UserAgent); userAgent != "" {
		requestOptions = append(requestOptions, option.WithHeader("User-Agent", userAgent))
	}
	for key, value := range cleanHeaderMap(options.Headers) {
		requestOptions = append(requestOptions, option.WithHeader(key, value))
	}
	return openai.NewClient(requestOptions...)
}

func (c *OpenAIClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if err := validateRequest(ctx, c.apiKey, req); err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, resolveRequestTimeout(c.options, req))
	defer cancel()

	params, err := c.buildChatParams(req)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Chat.Completions.New(callCtx, params)
	if err != nil {
		return nil, formatOpenAIError(err)
	}
	if resp == nil || len(resp.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI 响应为空")
	}

	msg := resp.Choices[0].Message
	return &ChatResponse{
		Content:   msg.Content,
		ToolCalls: convertOpenAIToolCalls(msg.ToolCalls),
		Usage:     convertOpenAIUsage(resp.Usage),
	}, nil
}

func (c *OpenAIClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	if err := validateRequest(ctx, c.apiKey, req); err != nil {
		return nil, err
	}

	callCtx, cancel := context.WithTimeout(ctx, resolveRequestTimeout(c.options, req))
	params, err := c.buildChatParams(req)
	if err != nil {
		cancel()
		return nil, err
	}
	params.StreamOptions = openai.ChatCompletionStreamOptionsParam{
		IncludeUsage: openai.Bool(true),
	}

	stream := c.client.Chat.Completions.NewStreaming(callCtx, params)
	if err := stream.Err(); err != nil {
		cancel()
		return nil, formatOpenAIError(err)
	}

	chunkChan := make(chan *StreamChunk)
	go func() {
		defer close(chunkChan)
		defer cancel()
		defer stream.Close()
		readOpenAIStream(callCtx, stream, chunkChan)
	}()

	return chunkChan, nil
}

func readOpenAIStream(ctx context.Context, stream interface {
	Next() bool
	Current() openai.ChatCompletionChunk
	Err() error
}, chunkChan chan<- *StreamChunk) {
	state := newOpenAIStreamState()
	lastFinishReason := ""

	for stream.Next() {
		chunk := stream.Current()
		state.noteUsage(chunk.Usage)
		for _, choice := range chunk.Choices {
			if choice.FinishReason != "" {
				lastFinishReason = choice.FinishReason
			}
			state.noteToolCallDeltas(choice.Delta.ToolCalls)
			out := &StreamChunk{
				Content:          choice.Delta.Content,
				ReasoningContent: extractOpenAIReasoningContent(choice.Delta),
				ToolCallDeltas:   convertOpenAIDeltaToolCalls(choice.Delta.ToolCalls),
				FinishReason:     choice.FinishReason,
			}
			if out.Content != "" || out.ReasoningContent != "" || len(out.ToolCallDeltas) > 0 {
				if !sendStreamChunk(ctx, chunkChan, out) {
					return
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		if ctx.Err() != nil {
			sendStreamChunk(ctx, chunkChan, &StreamChunk{Error: ctx.Err().Error(), Done: true, Usage: state.finalUsage(), FinishReason: lastFinishReason})
			return
		}
		sendStreamChunk(ctx, chunkChan, &StreamChunk{Error: formatOpenAIError(err).Error(), Done: true, Usage: state.finalUsage(), FinishReason: lastFinishReason})
		return
	}

	sendStreamChunk(ctx, chunkChan, &StreamChunk{
		Done:           true,
		FinalToolCalls: state.finalToolCalls(),
		FinishReason:   lastFinishReason,
		Usage:          state.finalUsage(),
	})
}

type openAIStreamState struct {
	toolCalls            map[int]*ToolCall
	promptTokens         int
	completionTokens     int
	totalTokens          int
	cachedTokens         int
	cachedTokensReported bool
}

func newOpenAIStreamState() *openAIStreamState {
	return &openAIStreamState{toolCalls: map[int]*ToolCall{}}
}

func (s *openAIStreamState) noteUsage(usage openai.CompletionUsage) {
	s.promptTokens += int(usage.PromptTokens)
	s.completionTokens += int(usage.CompletionTokens)
	s.totalTokens += int(usage.TotalTokens)
	cachedTokens, cachedTokensReported := openAIUsageCachedTokens(usage)
	if cachedTokensReported {
		s.cachedTokensReported = true
	}
	s.cachedTokens += cachedTokens
}

func (s *openAIStreamState) noteToolCallDeltas(toolCalls []openai.ChatCompletionChunkChoiceDeltaToolCall) {
	for _, delta := range toolCalls {
		idx := int(delta.Index)
		if idx < 0 {
			idx = 0
		}
		call := s.ensureToolCall(idx)
		if delta.ID != "" {
			call.ID = delta.ID
		}
		if delta.Type != "" {
			call.Type = delta.Type
		}
		call.Function.Name += delta.Function.Name
		call.Function.Arguments += delta.Function.Arguments
	}
}

func (s *openAIStreamState) ensureToolCall(idx int) *ToolCall {
	if call, ok := s.toolCalls[idx]; ok {
		return call
	}
	call := newToolCall("", "function", "", "")
	s.toolCalls[idx] = &call
	return &call
}

func (s *openAIStreamState) finalToolCalls() []ToolCall {
	if len(s.toolCalls) == 0 {
		return nil
	}
	out := make([]ToolCall, 0, len(s.toolCalls))
	for i := 0; i < len(s.toolCalls); i++ {
		call, ok := s.toolCalls[i]
		if !ok || call == nil {
			continue
		}
		if call.ID == "" && call.Function.Name == "" && call.Function.Arguments == "" {
			continue
		}
		out = append(out, *call)
	}
	return out
}

func (s *openAIStreamState) finalUsage() *Usage {
	if s.promptTokens == 0 && s.completionTokens == 0 && s.totalTokens == 0 {
		return nil
	}
	total := s.totalTokens
	if total == 0 {
		total = s.promptTokens + s.completionTokens
	}
	return &Usage{
		PromptTokens:         s.promptTokens,
		CompletionTokens:     s.completionTokens,
		TotalTokens:          total,
		CachedTokens:         s.cachedTokens,
		CachedTokensReported: s.cachedTokensReported,
	}
}

func extractOpenAIReasoningContent(delta openai.ChatCompletionChunkChoiceDelta) string {
	raw := strings.TrimSpace(delta.RawJSON())
	if raw == "" {
		return ""
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	for _, key := range []string{"reasoning_content", "reasoning"} {
		field, ok := payload[key]
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(field, &value); err == nil {
			return value
		}
	}
	return ""
}

func (c *OpenAIClient) GetModelName() string {
	return c.model
}

func (c *OpenAIClient) buildChatParams(req *ChatRequest) (openai.ChatCompletionNewParams, error) {
	messages, err := convertMessages(req.Messages)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	params := openai.ChatCompletionNewParams{
		Messages: messages,
		Model:    openai.ChatModel(model),
	}
	if req.MaxTokens > 0 {
		if usesMaxCompletionTokens(model, req.ReasoningEffort) {
			params.MaxCompletionTokens = openai.Int(int64(req.MaxTokens))
		} else {
			params.MaxTokens = openai.Int(int64(req.MaxTokens))
		}
	}
	if req.Temperature > 0 {
		params.Temperature = openai.Float(req.Temperature)
	}
	if len(req.Tools) > 0 {
		params.Tools = convertTools(req.Tools)
	}
	toolChoice, err := convertToolChoice(req.ToolChoice)
	if err != nil {
		return openai.ChatCompletionNewParams{}, err
	}
	params.ToolChoice = toolChoice
	if reasoningEffort := normalizeReasoningEffort(req.ReasoningEffort); reasoningEffort != "" {
		params.ReasoningEffort = shared.ReasoningEffort(reasoningEffort)
	}
	if verbosity := normalizeVerbosity(req.Verbosity); verbosity != "" {
		params.Verbosity = openai.ChatCompletionNewParamsVerbosity(verbosity)
	}
	if promptCacheKey := strings.TrimSpace(req.PromptCacheKey); promptCacheKey != "" {
		params.PromptCacheKey = openai.String(promptCacheKey)
	}
	if promptCacheRetention := normalizePromptCacheRetention(req.PromptCacheRetention); promptCacheRetention != "" {
		params.PromptCacheRetention = openai.ChatCompletionNewParamsPromptCacheRetention(promptCacheRetention)
	}
	return params, nil
}

func usesMaxCompletionTokens(model, reasoningEffort string) bool {
	if normalizeReasoningEffort(reasoningEffort) != "" {
		return true
	}
	model = strings.ToLower(strings.TrimSpace(model))
	for _, prefix := range []string{"gpt-5", "o1", "o3", "o4"} {
		if strings.HasPrefix(model, prefix) {
			return true
		}
	}
	return false
}

func normalizeReasoningEffort(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "none", "minimal", "low", "medium", "high", "xhigh":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeVerbosity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low", "medium", "high":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizePromptCacheRetention(value string) string {
	switch strings.TrimSpace(value) {
	case string(openai.ChatCompletionNewParamsPromptCacheRetentionInMemory):
		return string(openai.ChatCompletionNewParamsPromptCacheRetentionInMemory)
	case string(openai.ChatCompletionNewParamsPromptCacheRetention24h):
		return string(openai.ChatCompletionNewParamsPromptCacheRetention24h)
	default:
		return ""
	}
}

func convertMessages(messages []Message) ([]openai.ChatCompletionMessageParamUnion, error) {
	out := make([]openai.ChatCompletionMessageParamUnion, 0, len(messages))
	for i, msg := range messages {
		switch strings.ToLower(strings.TrimSpace(msg.Role)) {
		case "system":
			out = append(out, openai.SystemMessage(msg.Content))
		case "developer":
			out = append(out, openai.DeveloperMessage(msg.Content))
		case "user":
			out = append(out, openai.UserMessage(msg.Content))
		case "assistant":
			assistant := openai.ChatCompletionAssistantMessageParam{}
			if msg.Content != "" {
				assistant.Content.OfString = openai.String(msg.Content)
			}
			if len(msg.ToolCalls) > 0 {
				assistant.ToolCalls = convertToolCallParams(msg.ToolCalls)
			}
			out = append(out, openai.ChatCompletionMessageParamUnion{OfAssistant: &assistant})
		case "tool":
			out = append(out, openai.ToolMessage(msg.Content, msg.ToolCallID))
		default:
			return nil, fmt.Errorf("消息 %d 的 role %q 不支持，OpenAI Chat Completions 仅支持 system/developer/user/assistant/tool", i, msg.Role)
		}
	}
	return out, nil
}

func convertTools(tools []ToolDef) []openai.ChatCompletionToolUnionParam {
	out := make([]openai.ChatCompletionToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		fn := openai.FunctionDefinitionParam{
			Name:        tool.Function.Name,
			Description: openai.String(tool.Function.Description),
			Parameters:  openai.FunctionParameters(tool.Function.Parameters),
		}
		if tool.Function.Strict != nil {
			fn.Strict = openai.Bool(*tool.Function.Strict)
		}
		out = append(out, openai.ChatCompletionFunctionTool(fn))
	}
	return out
}

func convertToolChoice(value interface{}) (openai.ChatCompletionToolChoiceOptionUnionParam, error) {
	switch v := value.(type) {
	case nil:
		return openai.ChatCompletionToolChoiceOptionUnionParam{}, nil
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "":
			return openai.ChatCompletionToolChoiceOptionUnionParam{}, nil
		case "none", "auto", "required":
			return openai.ChatCompletionToolChoiceOptionUnionParam{OfAuto: openai.String(strings.ToLower(strings.TrimSpace(v)))}, nil
		default:
			return openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{Name: v}), nil
		}
	case map[string]interface{}:
		return convertMapToolChoice(v)
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return openai.ChatCompletionToolChoiceOptionUnionParam{}, fmt.Errorf("tool_choice 序列化失败: %w", err)
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return openai.ChatCompletionToolChoiceOptionUnionParam{}, fmt.Errorf("tool_choice 必须是字符串或对象: %w", err)
		}
		return convertMapToolChoice(obj)
	}
}

func convertMapToolChoice(value map[string]interface{}) (openai.ChatCompletionToolChoiceOptionUnionParam, error) {
	typ, _ := value["type"].(string)
	if strings.ToLower(strings.TrimSpace(typ)) != "function" {
		return openai.ChatCompletionToolChoiceOptionUnionParam{}, fmt.Errorf("tool_choice 仅支持 function 类型")
	}
	fn, _ := value["function"].(map[string]interface{})
	name, _ := fn["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return openai.ChatCompletionToolChoiceOptionUnionParam{}, fmt.Errorf("tool_choice.function.name 不能为空")
	}
	return openai.ToolChoiceOptionFunctionToolChoice(openai.ChatCompletionNamedToolChoiceFunctionParam{Name: name}), nil
}

func convertToolCallParams(toolCalls []ToolCall) []openai.ChatCompletionMessageToolCallUnionParam {
	out := make([]openai.ChatCompletionMessageToolCallUnionParam, 0, len(toolCalls))
	for _, tc := range toolCalls {
		out = append(out, openai.ChatCompletionMessageToolCallUnionParam{
			OfFunction: &openai.ChatCompletionMessageFunctionToolCallParam{
				ID: tc.ID,
				Function: openai.ChatCompletionMessageFunctionToolCallFunctionParam{
					Name:      tc.Function.Name,
					Arguments: sanitizeToolCallArguments(tc.Function.Arguments),
				},
			},
		})
	}
	return out
}

func convertOpenAIToolCalls(toolCalls []openai.ChatCompletionMessageToolCallUnion) []ToolCall {
	out := make([]ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		id := tc.ID
		typ := string(tc.Type)
		name := tc.Function.Name
		arguments := tc.Function.Arguments
		if id == "" && name == "" && arguments == "" {
			fn := tc.AsFunction()
			id = fn.ID
			typ = string(fn.Type)
			name = fn.Function.Name
			arguments = fn.Function.Arguments
		}
		if id == "" && name == "" && arguments == "" {
			continue
		}
		out = append(out, newToolCall(id, typ, name, arguments))
	}
	return out
}

func convertOpenAIDeltaToolCalls(toolCalls []openai.ChatCompletionChunkChoiceDeltaToolCall) []ToolCallDelta {
	out := make([]ToolCallDelta, 0, len(toolCalls))
	for _, tc := range toolCalls {
		idx := int(tc.Index)
		out = append(out, newToolCallDelta(&idx, tc.ID, tc.Type, tc.Function.Name, tc.Function.Arguments))
	}
	return out
}

func accumulatedOpenAIToolCalls(acc openai.ChatCompletionAccumulator) []ToolCall {
	if len(acc.Choices) == 0 {
		return nil
	}
	return convertOpenAIToolCalls(acc.Choices[0].Message.ToolCalls)
}

func newToolCall(id, typ, name, arguments string) ToolCall {
	if typ == "" {
		typ = "function"
	}
	tc := ToolCall{ID: id, Type: typ}
	tc.Function.Name = name
	tc.Function.Arguments = arguments
	return tc
}

func newToolCallDelta(index *int, id, typ, name, arguments string) ToolCallDelta {
	if typ == "" {
		typ = "function"
	}
	tc := ToolCallDelta{Index: index, ID: id, Type: typ}
	tc.Function.Name = name
	tc.Function.Arguments = arguments
	return tc
}

func sanitizeToolCallArguments(arguments string) string {
	trimmed := strings.TrimSpace(arguments)
	if trimmed == "" {
		return "{}"
	}
	if !json.Valid([]byte(trimmed)) {
		return "{}"
	}
	return arguments
}

func convertOpenAIUsage(usage openai.CompletionUsage) *Usage {
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	cachedTokens, cachedTokensReported := openAIUsageCachedTokens(usage)
	return &Usage{
		PromptTokens:         int(usage.PromptTokens),
		CompletionTokens:     int(usage.CompletionTokens),
		TotalTokens:          int(usage.TotalTokens),
		CachedTokens:         cachedTokens,
		CachedTokensReported: cachedTokensReported,
	}
}

func openAIUsageCachedTokens(usage openai.CompletionUsage) (int, bool) {
	if usage.PromptTokensDetails.JSON.CachedTokens.Valid() {
		return int(usage.PromptTokensDetails.CachedTokens), true
	}
	raw := strings.TrimSpace(usage.RawJSON())
	if raw == "" {
		return int(usage.PromptTokensDetails.CachedTokens), false
	}
	var obj map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return int(usage.PromptTokensDetails.CachedTokens), false
	}
	if cached, ok := cachedTokensFromUsageObject(obj); ok {
		return cached, true
	}
	return int(usage.PromptTokensDetails.CachedTokens), false
}

func cachedTokensFromUsageObject(obj map[string]json.RawMessage) (int, bool) {
	if rawDetails, ok := obj["prompt_tokens_details"]; ok && len(rawDetails) > 0 && string(rawDetails) != "null" {
		var details map[string]json.RawMessage
		if err := json.Unmarshal(rawDetails, &details); err == nil {
			if cached, ok := intFromRawJSON(details["cached_tokens"]); ok {
				return cached, true
			}
		}
	}
	if cached, ok := intFromRawJSON(obj["cached_tokens"]); ok {
		return cached, true
	}
	if cached, ok := intFromRawJSON(obj["cache_read_input_tokens"]); ok {
		return cached, true
	}
	if _, ok := intFromRawJSON(obj["cache_creation_input_tokens"]); ok {
		return 0, true
	}
	return 0, false
}

func intFromRawJSON(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return 0, false
	}
	var number json.Number
	if err := json.Unmarshal(raw, &number); err != nil {
		return 0, false
	}
	i, err := number.Int64()
	if err == nil {
		return int(i), true
	}
	f, err := number.Float64()
	if err != nil {
		return 0, false
	}
	return int(f), true
}

func convertAccumulatedOpenAIUsage(usage openai.CompletionUsage, cachedTokensReported bool) *Usage {
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return &Usage{
		PromptTokens:         int(usage.PromptTokens),
		CompletionTokens:     int(usage.CompletionTokens),
		TotalTokens:          int(usage.TotalTokens),
		CachedTokens:         int(usage.PromptTokensDetails.CachedTokens),
		CachedTokensReported: cachedTokensReported,
	}
}

func formatOpenAIError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		if apiErr.Message != "" {
			providerErr := &ProviderError{
				HTTPStatus: apiErr.StatusCode,
				Code:       apiErr.Code,
				Type:       apiErr.Type,
				Param:      apiErr.Param,
				Message:    apiErr.Message,
			}
			message := "OpenAI API 错误: " + providerErr.Error()
			if IsContextWindowProviderError(providerErr.Code, providerErr.Type, providerErr.Param, providerErr.Message) {
				contextErr := &ContextWindowError{Message: message, Cause: providerErr}
				contextErr.MaxContextTokens = ContextWindowLimitFromError(contextErr)
				return contextErr
			}
			return providerErr
		}
	}
	if IsContextWindowErrorMessage(err.Error()) {
		return &ContextWindowError{Message: err.Error()}
	}
	return err
}

var _ LLMClient = (*OpenAIClient)(nil)
