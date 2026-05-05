package llms

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// Kimi（Moonshot）使用 OpenAI 兼容接口：POST /v1/chat/completions
const (
	kimiDefaultBaseURL = "https://api.moonshot.cn/v1/chat/completions"
	kimiDefaultModel   = "kimi-k2-0711-preview"
)

type kimiUsage struct {
	PromptTokens     float64 `json:"prompt_tokens"`
	CompletionTokens float64 `json:"completion_tokens"`
	TotalTokens      float64 `json:"total_tokens"`
}

// KimiAPIResponse Kimi / Moonshot OpenAI 兼容响应
type KimiAPIResponse struct {
	Error *struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
	} `json:"error,omitempty"`
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"message"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
	Usage *kimiUsage `json:"usage,omitempty"`
}

// KimiStreamResponse Kimi 流式 SSE 分片（OpenAI 兼容）
type KimiStreamResponse struct {
	Error *struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message"`
		Type    string `json:"type,omitempty"`
	} `json:"error,omitempty"`
	Choices []struct {
		Delta struct {
			Role      string     `json:"role,omitempty"`
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"`
		} `json:"delta"`
		FinishReason *string    `json:"finish_reason,omitempty"`
		Usage        *kimiUsage `json:"usage,omitempty"`
	} `json:"choices,omitempty"`
	Usage *kimiUsage `json:"usage,omitempty"`
}

// KimiClient Kimi（Moonshot）OpenAI 兼容客户端
type KimiClient struct {
	APIKey  string
	BaseURL string
	Options *ClientOptions
	Model   string
}

// NewKimiClient 创建 Kimi 客户端
func NewKimiClient(apiKey string) *KimiClient {
	return NewKimiClientWithOptions(apiKey, DefaultClientOptions())
}

// NewKimiClientWithOptions 创建带配置的 Kimi 客户端
func NewKimiClientWithOptions(apiKey string, options *ClientOptions) *KimiClient {
	if options == nil {
		options = DefaultClientOptions()
	}
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = kimiDefaultBaseURL
	}
	model := kimiDefaultModel
	if options.Model != "" {
		model = options.Model
	}
	return &KimiClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Options: options,
		Model:   model,
	}
}

// SetModel 设置模型名称
func (c *KimiClient) SetModel(model string) {
	c.Model = model
}

// GetModelName 获取模型名称
func (c *KimiClient) GetModelName() string {
	return c.Model
}

// GetProvider 获取提供商名称
func (c *KimiClient) GetProvider() string {
	return string(ProviderKimi)
}

// kimiSanitizeMessages 与 MiniMax 等 OpenAI 兼容厂商一致：tool_calls.arguments 须为合法 JSON 字符串
func kimiSanitizeMessages(msgs []Message) []Message {
	if len(msgs) == 0 {
		return msgs
	}
	out := make([]Message, 0, len(msgs))
	for _, m := range msgs {
		msg := m
		if len(msg.ToolCalls) > 0 {
			tcCopy := make([]ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				tcCopy[i] = tc
				arg := strings.TrimSpace(tc.Function.Arguments)
				if arg == "" || !json.Valid([]byte(arg)) {
					tcCopy[i].Function.Arguments = "{}"
				} else {
					tcCopy[i].Function.Arguments = arg
				}
			}
			msg.ToolCalls = tcCopy
		}
		out = append(out, msg)
	}
	return out
}

func kimiResolvedModel(reqModel, defaultModel string) string {
	if model := strings.TrimSpace(reqModel); model != "" {
		return model
	}
	return defaultModel
}

func kimiUsesFixedSampling(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	return strings.HasPrefix(model, "kimi-k2") ||
		strings.HasPrefix(model, "kimi-latest") ||
		strings.HasPrefix(model, "kimi-thinking")
}

func kimiSupportsThinkingControl(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if strings.Contains(model, "thinking") {
		return false
	}
	return strings.HasPrefix(model, "kimi-k2.6") ||
		strings.HasPrefix(model, "kimi-k2.5") ||
		strings.HasPrefix(model, "kimi-latest")
}

func applyKimiTemperature(apiReq map[string]interface{}, model string, temperature float64) {
	if temperature == 0 || kimiUsesFixedSampling(model) {
		return
	}
	apiReq["temperature"] = temperature
}

func applyKimiThinking(apiReq map[string]interface{}, model string, req *ChatRequest) {
	if req == nil || !kimiSupportsThinkingControl(model) {
		return
	}
	if len(req.Tools) > 0 {
		apiReq["thinking"] = map[string]string{"type": "disabled"}
		return
	}
	if req.UseThinking == nil {
		return
	}
	thinkingType := "disabled"
	if *req.UseThinking {
		thinkingType = "enabled"
	}
	apiReq["thinking"] = map[string]string{"type": thinkingType}
}

func kimiUsageToUsage(usage *kimiUsage) *Usage {
	if usage == nil {
		return nil
	}
	return &Usage{
		PromptTokens:     int(usage.PromptTokens),
		CompletionTokens: int(usage.CompletionTokens),
		TotalTokens:      int(usage.TotalTokens),
	}
}

// Chat 实现 LLMClient 接口（OpenAI 兼容：tools / tool_calls）
func (c *KimiClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	msgs := kimiSanitizeMessages(req.Messages)
	model := kimiResolvedModel(req.Model, c.Model)
	apiReq := map[string]interface{}{
		"model":      model,
		"messages":   msgs,
		"max_tokens": req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		apiReq["tools"] = req.Tools
		if req.ToolChoice != nil {
			apiReq["tool_choice"] = req.ToolChoice
		}
	}
	if req.MaxTokens <= 0 {
		apiReq["max_tokens"] = 4096
	}
	applyKimiTemperature(apiReq, model, req.Temperature)
	applyKimiThinking(apiReq, model, req)

	httpClient := createHTTPClient(c.Options, resolveRequestTimeout(c.Options, req))

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}
	httpReq, err := newBearerJSONRequest(ctx, c.BaseURL, c.APIKey, jsonData, c.Options)
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	if c.Options != nil && c.Options.EnableLogging {
		logger.Infof(ctx, "[Kimi] 发送请求, 请求体长度: %d", len(jsonData))
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	var apiResp KimiAPIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if apiResp.Error != nil {
		if apiResp.Error.Code != "" {
			return nil, fmt.Errorf("Kimi API错误: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
		}
		return nil, fmt.Errorf("Kimi API错误: %s", apiResp.Error.Message)
	}
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("响应格式错误：没有找到 choices")
	}

	choice := apiResp.Choices[0]
	return &ChatResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
		Usage:     kimiUsageToUsage(apiResp.Usage),
	}, nil
}

// ChatStream 实现流式聊天接口（OpenAI 兼容 SSE）
func (c *KimiClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	msgs := kimiSanitizeMessages(req.Messages)
	chunkChan := make(chan *StreamChunk, 10)
	go func() {
		defer close(chunkChan)
		model := kimiResolvedModel(req.Model, c.Model)
		apiReq := map[string]interface{}{
			"model":      model,
			"messages":   msgs,
			"max_tokens": req.MaxTokens,
			"stream":     true,
		}
		if len(req.Tools) > 0 {
			apiReq["tools"] = req.Tools
			if req.ToolChoice != nil {
				apiReq["tool_choice"] = req.ToolChoice
			}
		}
		if req.MaxTokens <= 0 {
			apiReq["max_tokens"] = 4096
		}
		applyKimiTemperature(apiReq, model, req.Temperature)
		applyKimiThinking(apiReq, model, req)

		httpClient := createHTTPClient(c.Options, resolveRequestTimeout(c.Options, req))

		jsonData, err := json.Marshal(apiReq)
		if err != nil {
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("序列化请求失败: %v", err), Done: true}
			return
		}
		httpReq, err := newBearerJSONRequest(ctx, c.BaseURL, c.APIKey, jsonData, c.Options)
		if err != nil {
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("创建HTTP请求失败: %v", err), Done: true}
			return
		}

		resp, err := httpClient.Do(httpReq)
		if err != nil {
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("HTTP请求失败: %v", err), Done: true}
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			all, _ := io.ReadAll(resp.Body)
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(all)), Done: true}
			return
		}

		scanner := bufio.NewScanner(resp.Body)
		var finalUsage *Usage
		doneReceived := false
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				doneReceived = true
				chunkChan <- &StreamChunk{Usage: finalUsage, Done: true}
				break
			}
			var streamResp KimiStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				chunkChan <- &StreamChunk{Error: fmt.Sprintf("解析流式响应失败: %v", err), Done: true}
				return
			}
			if streamResp.Error != nil {
				msg := streamResp.Error.Message
				if streamResp.Error.Code != "" {
					msg = streamResp.Error.Code + " - " + msg
				}
				chunkChan <- &StreamChunk{Error: "Kimi API错误: " + msg, Done: true}
				return
			}
			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]
				if choice.Delta.Content != "" {
					chunkChan <- &StreamChunk{Content: choice.Delta.Content, Done: false}
				}
				if len(choice.Delta.ToolCalls) > 0 {
					chunkChan <- &StreamChunk{ToolCalls: choice.Delta.ToolCalls, Done: false}
				}
				if choice.FinishReason != nil && *choice.FinishReason != "" {
					if streamResp.Usage != nil {
						finalUsage = kimiUsageToUsage(streamResp.Usage)
					} else if choice.Usage != nil {
						finalUsage = kimiUsageToUsage(choice.Usage)
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("读取流式响应失败: %v", err), Done: true}
			return
		}
		if !doneReceived {
			chunkChan <- &StreamChunk{Error: "流式响应未收到结束标志 [DONE]", Done: true}
		}
	}()
	return chunkChan, nil
}

// GetSupportedModels 获取支持的模型列表（参考 Moonshot 文档，以实际平台为准）
func (c *KimiClient) GetSupportedModels() []string {
	return []string{
		"kimi-k2-0711-preview",
		"moonshot-v1-8k",
		"moonshot-v1-32k",
		"moonshot-v1-128k",
		"moonshot-v1-auto",
		"kimi-latest",
		"moonshot-v1-8k-vision-preview",
		"moonshot-v1-32k-vision-preview",
		"moonshot-v1-128k-vision-preview",
		"kimi-thinking-preview",
	}
}

// GetPricingInfo 获取价格信息（仅供参考）
func (c *KimiClient) GetPricingInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":          c.Model,
		"context_length": "视模型而定",
		"note":           "价格以 Moonshot 开放平台为准；接口为 OpenAI 兼容 /v1/chat/completions",
	}
}
