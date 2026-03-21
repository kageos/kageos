package llms

import (
	"bufio"
	"bytes"
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
	Usage *struct {
		PromptTokens     float64 `json:"prompt_tokens"`
		CompletionTokens float64 `json:"completion_tokens"`
		TotalTokens      float64 `json:"total_tokens"`
	} `json:"usage,omitempty"`
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
		FinishReason *string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
	Usage *struct {
		PromptTokens     float64 `json:"prompt_tokens"`
		CompletionTokens float64 `json:"completion_tokens"`
		TotalTokens      float64 `json:"total_tokens"`
	} `json:"usage,omitempty"`
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

// Chat 实现 LLMClient 接口（OpenAI 兼容：tools / tool_calls）
func (c *KimiClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	msgs := kimiSanitizeMessages(req.Messages)
	apiReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    msgs,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
	if len(req.Tools) > 0 {
		apiReq["tools"] = req.Tools
		if req.ToolChoice != nil {
			apiReq["tool_choice"] = req.ToolChoice
		}
	}
	if apiReq["model"] == "" || apiReq["model"] == nil {
		apiReq["model"] = c.Model
	}
	if req.MaxTokens <= 0 {
		apiReq["max_tokens"] = 4096
	}
	if req.Temperature == 0 {
		apiReq["temperature"] = 0.1
	}

	timeout := c.Options.Timeout
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout
	}
	httpClient := createHTTPClient(c.Options, timeout)

	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	if c.Options != nil && c.Options.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.Options.UserAgent)
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
	var usage *Usage
	if apiResp.Usage != nil {
		usage = &Usage{
			PromptTokens:     int(apiResp.Usage.PromptTokens),
			CompletionTokens: int(apiResp.Usage.CompletionTokens),
			TotalTokens:      int(apiResp.Usage.TotalTokens),
		}
	}
	return &ChatResponse{
		Content:   choice.Message.Content,
		ToolCalls: choice.Message.ToolCalls,
		Usage:     usage,
	}, nil
}

// ChatStream 实现流式聊天接口（OpenAI 兼容 SSE）
func (c *KimiClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	msgs := kimiSanitizeMessages(req.Messages)
	chunkChan := make(chan *StreamChunk, 10)
	go func() {
		defer close(chunkChan)
		apiReq := map[string]interface{}{
			"model":       req.Model,
			"messages":    msgs,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
			"stream":      true,
		}
		if len(req.Tools) > 0 {
			apiReq["tools"] = req.Tools
			if req.ToolChoice != nil {
				apiReq["tool_choice"] = req.ToolChoice
			}
		}
		if apiReq["model"] == "" || apiReq["model"] == nil {
			apiReq["model"] = c.Model
		}
		if req.MaxTokens <= 0 {
			apiReq["max_tokens"] = 4096
		}
		if req.Temperature == 0 {
			apiReq["temperature"] = 0.1
		}

		timeout := c.Options.Timeout
		if req.Timeout != nil && *req.Timeout > 0 {
			timeout = *req.Timeout
		}
		httpClient := createHTTPClient(c.Options, timeout)

		jsonData, err := json.Marshal(apiReq)
		if err != nil {
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("序列化请求失败: %v", err), Done: true}
			return
		}
		httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			chunkChan <- &StreamChunk{Error: fmt.Sprintf("创建HTTP请求失败: %v", err), Done: true}
			return
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
		if c.Options.UserAgent != "" {
			httpReq.Header.Set("User-Agent", c.Options.UserAgent)
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
						finalUsage = &Usage{
							PromptTokens:     int(streamResp.Usage.PromptTokens),
							CompletionTokens: int(streamResp.Usage.CompletionTokens),
							TotalTokens:      int(streamResp.Usage.TotalTokens),
						}
					}
					chunkChan <- &StreamChunk{ToolCalls: choice.Delta.ToolCalls, Usage: finalUsage, Done: true}
					break
				}
			}
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
		"note":             "价格以 Moonshot 开放平台为准；接口为 OpenAI 兼容 /v1/chat/completions",
	}
}
