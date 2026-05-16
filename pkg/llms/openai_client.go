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
)

const defaultOpenAIModel = "gpt-4o-mini"

// OpenAIClient is the only concrete LLM client. Non-OpenAI vendors must expose
// an OpenAI-compatible Chat Completions endpoint and be configured via BaseURL.
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

	params, err := c.buildChatParams(req)
	if err != nil {
		return nil, err
	}
	params.StreamOptions.IncludeUsage = openai.Bool(true)

	callCtx, cancel := context.WithTimeout(ctx, resolveRequestTimeout(c.options, req))
	stream := c.client.Chat.Completions.NewStreaming(callCtx, params)

	chunkChan := make(chan *StreamChunk)
	go func() {
		defer close(chunkChan)
		defer cancel()
		defer func() {
			_ = stream.Close()
		}()

		var lastUsage *Usage
		for stream.Next() {
			chunk := stream.Current()
			if usage := convertOpenAIUsage(chunk.Usage); usage != nil {
				lastUsage = usage
			}
			for _, choice := range chunk.Choices {
				out := &StreamChunk{
					Content:   choice.Delta.Content,
					ToolCalls: convertOpenAIDeltaToolCalls(choice.Delta.ToolCalls),
					Done:      false,
				}
				if out.Content != "" || len(out.ToolCalls) > 0 {
					chunkChan <- out
				}
			}
		}

		if err := stream.Err(); err != nil {
			chunkChan <- &StreamChunk{Error: formatOpenAIError(err).Error(), Done: true, Usage: lastUsage}
			return
		}
		chunkChan <- &StreamChunk{Done: true, Usage: lastUsage}
	}()

	return chunkChan, nil
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
		params.MaxTokens = openai.Int(int64(req.MaxTokens))
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
	return params, nil
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
		fn := tc.AsFunction()
		if fn.ID == "" && fn.Function.Name == "" && fn.Function.Arguments == "" {
			continue
		}
		out = append(out, newToolCall(fn.ID, "function", nil, fn.Function.Name, fn.Function.Arguments))
	}
	return out
}

func convertOpenAIDeltaToolCalls(toolCalls []openai.ChatCompletionChunkChoiceDeltaToolCall) []ToolCall {
	out := make([]ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		idx := int(tc.Index)
		out = append(out, newToolCall(tc.ID, tc.Type, &idx, tc.Function.Name, tc.Function.Arguments))
	}
	return out
}

func newToolCall(id, typ string, index *int, name, arguments string) ToolCall {
	if typ == "" {
		typ = "function"
	}
	tc := ToolCall{ID: id, Type: typ, Index: index}
	tc.Function.Name = name
	tc.Function.Arguments = arguments
	return tc
}

func sanitizeToolCallArguments(arguments string) string {
	if strings.TrimSpace(arguments) == "" {
		return "{}"
	}
	return arguments
}

func convertOpenAIUsage(usage openai.CompletionUsage) *Usage {
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return &Usage{
		PromptTokens:     int(usage.PromptTokens),
		CompletionTokens: int(usage.CompletionTokens),
		TotalTokens:      int(usage.TotalTokens),
	}
}

func formatOpenAIError(err error) error {
	if err == nil {
		return nil
	}
	var apiErr *openai.Error
	if errors.As(err, &apiErr) {
		if apiErr.Message != "" {
			return fmt.Errorf("OpenAI API 错误: %s", apiErr.Message)
		}
	}
	return err
}

var _ LLMClient = (*OpenAIClient)(nil)
