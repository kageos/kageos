package llms

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const defaultOpenAIBaseURL = "https://api.openai.com/v1"

type openAICompatibleChatStreamRequest struct {
	Messages      []openAICompatibleMessage      `json:"messages"`
	Model         string                         `json:"model"`
	MaxTokens     int                            `json:"max_tokens,omitempty"`
	Temperature   *float64                       `json:"temperature,omitempty"`
	Tools         []ToolDef                      `json:"tools,omitempty"`
	ToolChoice    interface{}                    `json:"tool_choice,omitempty"`
	Stream        bool                           `json:"stream"`
	StreamOptions *openAICompatibleStreamOptions `json:"stream_options,omitempty"`
}

type openAICompatibleStreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type openAICompatibleMessage struct {
	Role       string     `json:"role"`
	Content    *string    `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

func (c *OpenAIClient) buildChatStreamPayload(req *ChatRequest) (*openAICompatibleChatStreamRequest, error) {
	messages, err := convertOpenAICompatibleMessages(req.Messages)
	if err != nil {
		return nil, err
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.model
	}
	toolChoice, err := normalizeOpenAICompatibleToolChoice(req.ToolChoice)
	if err != nil {
		return nil, err
	}
	payload := &openAICompatibleChatStreamRequest{
		Messages: messages,
		Model:    model,
		Tools:    normalizeOpenAICompatibleTools(req.Tools),
		Stream:   true,
		StreamOptions: &openAICompatibleStreamOptions{
			IncludeUsage: true,
		},
	}
	if req.MaxTokens > 0 {
		payload.MaxTokens = req.MaxTokens
	}
	if req.Temperature > 0 {
		temperature := req.Temperature
		payload.Temperature = &temperature
	}
	if toolChoice != nil {
		payload.ToolChoice = toolChoice
	}
	return payload, nil
}

func (c *OpenAIClient) chatCompletionsURL() string {
	options := normalizeClientOptions(c.options)
	baseURL := strings.TrimSpace(options.BaseURL)
	if baseURL == "" {
		baseURL = defaultOpenAIBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")
	if strings.HasSuffix(baseURL, "/chat/completions") {
		return baseURL
	}
	return baseURL + "/chat/completions"
}

func convertOpenAICompatibleMessages(messages []Message) ([]openAICompatibleMessage, error) {
	out := make([]openAICompatibleMessage, 0, len(messages))
	for i, msg := range messages {
		role := strings.ToLower(strings.TrimSpace(msg.Role))
		item := openAICompatibleMessage{Role: role}
		switch role {
		case "system", "developer", "user":
			content := msg.Content
			item.Content = &content
		case "assistant":
			if msg.Content != "" {
				content := msg.Content
				item.Content = &content
			}
			if len(msg.ToolCalls) > 0 {
				item.ToolCalls = normalizeOpenAICompatibleToolCalls(msg.ToolCalls)
			}
		case "tool":
			content := msg.Content
			item.Content = &content
			item.ToolCallID = msg.ToolCallID
		default:
			return nil, fmt.Errorf("消息 %d 的 role %q 不支持，OpenAI Chat Completions 仅支持 system/developer/user/assistant/tool", i, msg.Role)
		}
		out = append(out, item)
	}
	return out, nil
}

func normalizeOpenAICompatibleTools(tools []ToolDef) []ToolDef {
	out := make([]ToolDef, 0, len(tools))
	for _, tool := range tools {
		if strings.TrimSpace(tool.Type) == "" {
			tool.Type = "function"
		}
		out = append(out, tool)
	}
	return out
}

func normalizeOpenAICompatibleToolCalls(toolCalls []ToolCall) []ToolCall {
	out := make([]ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		if tc.Type == "" {
			tc.Type = "function"
		}
		tc.Function.Arguments = sanitizeToolCallArguments(tc.Function.Arguments)
		out = append(out, tc)
	}
	return out
}

func normalizeOpenAICompatibleToolChoice(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case string:
		choice := strings.ToLower(strings.TrimSpace(v))
		switch choice {
		case "":
			return nil, nil
		case "none", "auto", "required":
			return choice, nil
		default:
			return map[string]interface{}{
				"type": "function",
				"function": map[string]interface{}{
					"name": strings.TrimSpace(v),
				},
			}, nil
		}
	case map[string]interface{}:
		if _, err := convertMapToolChoice(v); err != nil {
			return nil, err
		}
		return v, nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return nil, fmt.Errorf("tool_choice 序列化失败: %w", err)
		}
		var obj map[string]interface{}
		if err := json.Unmarshal(data, &obj); err != nil {
			return nil, fmt.Errorf("tool_choice 必须是字符串或对象: %w", err)
		}
		if _, err := convertMapToolChoice(obj); err != nil {
			return nil, err
		}
		return obj, nil
	}
}

type openAICompatibleStreamState struct {
	lastUsage     *Usage
	pendingJSON   string
	sawParsedData bool
	done          bool
}

type openAICompatibleStreamChunk struct {
	Choices []struct {
		Delta struct {
			Content   string                           `json:"content"`
			ToolCalls []openAICompatibleStreamToolCall `json:"tool_calls"`
		} `json:"delta"`
	} `json:"choices"`
	Usage *openAICompatibleUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}

type openAICompatibleStreamToolCall struct {
	Index    *int   `json:"index"`
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type openAICompatibleUsage struct {
	PromptTokens        int                                 `json:"prompt_tokens"`
	CompletionTokens    int                                 `json:"completion_tokens"`
	TotalTokens         int                                 `json:"total_tokens"`
	PromptTokensDetails openAICompatiblePromptTokensDetails `json:"prompt_tokens_details"`
}

type openAICompatiblePromptTokensDetails struct {
	CachedTokens         int
	CachedTokensReported bool
}

func (d *openAICompatiblePromptTokensDetails) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || strings.TrimSpace(string(data)) == "null" {
		return nil
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if value, ok := raw["cached_tokens"]; ok {
		d.CachedTokensReported = true
		_ = json.Unmarshal(value, &d.CachedTokens)
	}
	return nil
}

func readOpenAICompatibleStream(ctx context.Context, body io.Reader, chunkChan chan<- *StreamChunk) {
	reader := bufio.NewReader(body)
	state := &openAICompatibleStreamState{}
	dataLines := make([]string, 0, 1)

	flush := func() bool {
		if len(dataLines) == 0 {
			return false
		}
		data := strings.Join(dataLines, "\n")
		dataLines = dataLines[:0]
		return processOpenAICompatibleStreamData(data, state, chunkChan)
	}

	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			if line == "" {
				if flush() {
					return
				}
			} else if strings.HasPrefix(line, "data:") {
				dataLines = append(dataLines, strings.TrimSpace(line[5:]))
			}
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if flush() {
					return
				}
				break
			}
			if ctx.Err() != nil {
				chunkChan <- &StreamChunk{Error: ctx.Err().Error(), Done: true, Usage: state.lastUsage}
				return
			}
			chunkChan <- &StreamChunk{Error: err.Error(), Done: true, Usage: state.lastUsage}
			return
		}
	}

	if state.done {
		return
	}
	if state.pendingJSON != "" && !state.sawParsedData {
		chunkChan <- &StreamChunk{Error: "LLM 流式响应 JSON 不完整", Done: true, Usage: state.lastUsage}
		return
	}
	chunkChan <- &StreamChunk{Done: true, Usage: state.lastUsage}
}

func processOpenAICompatibleStreamData(data string, state *openAICompatibleStreamState, chunkChan chan<- *StreamChunk) bool {
	data = strings.TrimSpace(data)
	if data == "" {
		return false
	}
	if data == "[DONE]" {
		state.done = true
		chunkChan <- &StreamChunk{Done: true, Usage: state.lastUsage}
		return true
	}

	chunk, ok, incomplete, err := decodeOpenAICompatibleStreamChunk(data)
	if state.pendingJSON != "" {
		combined := state.pendingJSON + data
		chunk, ok, incomplete, err = decodeOpenAICompatibleStreamChunk(combined)
		if ok {
			state.pendingJSON = ""
		} else if incomplete {
			state.pendingJSON = combined
			return false
		} else if state.sawParsedData {
			state.pendingJSON = ""
			chunk, ok, incomplete, err = decodeOpenAICompatibleStreamChunk(data)
		}
	}
	if !ok {
		if incomplete {
			state.pendingJSON = data
			return false
		}
		chunkChan <- &StreamChunk{Error: fmt.Sprintf("解析 LLM 流式响应失败: %v", err), Done: true, Usage: state.lastUsage}
		state.done = true
		return true
	}

	state.sawParsedData = true
	if chunk.Error != nil && strings.TrimSpace(chunk.Error.Message) != "" {
		chunkChan <- &StreamChunk{Error: chunk.Error.Message, Done: true, Usage: state.lastUsage}
		state.done = true
		return true
	}
	if usage := convertOpenAICompatibleUsage(chunk.Usage); usage != nil {
		state.lastUsage = usage
	}
	for _, choice := range chunk.Choices {
		out := &StreamChunk{
			Content:   choice.Delta.Content,
			ToolCalls: convertOpenAICompatibleDeltaToolCalls(choice.Delta.ToolCalls),
			Done:      false,
		}
		if out.Content != "" || len(out.ToolCalls) > 0 {
			chunkChan <- out
		}
	}
	return false
}

func decodeOpenAICompatibleStreamChunk(data string) (*openAICompatibleStreamChunk, bool, bool, error) {
	var chunk openAICompatibleStreamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return nil, false, isIncompleteJSONError(err), err
	}
	return &chunk, true, false, nil
}

func isIncompleteJSONError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	msg := err.Error()
	return strings.Contains(msg, "unexpected end of JSON input") || strings.Contains(msg, "unexpected EOF")
}

func convertOpenAICompatibleDeltaToolCalls(toolCalls []openAICompatibleStreamToolCall) []ToolCall {
	out := make([]ToolCall, 0, len(toolCalls))
	for _, tc := range toolCalls {
		out = append(out, newToolCall(tc.ID, tc.Type, tc.Index, tc.Function.Name, tc.Function.Arguments))
	}
	return out
}

func convertOpenAICompatibleUsage(usage *openAICompatibleUsage) *Usage {
	if usage == nil {
		return nil
	}
	if usage.PromptTokens == 0 && usage.CompletionTokens == 0 && usage.TotalTokens == 0 {
		return nil
	}
	return &Usage{
		PromptTokens:         usage.PromptTokens,
		CompletionTokens:     usage.CompletionTokens,
		TotalTokens:          usage.TotalTokens,
		CachedTokens:         usage.PromptTokensDetails.CachedTokens,
		CachedTokensReported: usage.PromptTokensDetails.CachedTokensReported,
	}
}

func formatOpenAICompatibleHTTPError(statusCode int, raw []byte) error {
	msg := strings.TrimSpace(string(raw))
	var obj struct {
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Error   struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		switch {
		case strings.TrimSpace(obj.Error.Message) != "":
			msg = obj.Error.Message
		case strings.TrimSpace(obj.Message) != "":
			msg = obj.Message
		case strings.TrimSpace(obj.Msg) != "":
			msg = obj.Msg
		}
	}
	if msg == "" {
		msg = http.StatusText(statusCode)
	}
	return fmt.Errorf("OpenAI API 错误: HTTP %d: %s", statusCode, msg)
}
