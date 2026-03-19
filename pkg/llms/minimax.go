package llms

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// MiniMax 使用 OpenAI 兼容接口，响应格式与 DeepSeek 一致
const (
	miniMaxDefaultBaseURL = "https://api.minimaxi.com/v1/chat/completions"
	miniMaxDefaultModel   = "MiniMax-M2.5-highspeed"
)

// MiniMaxAPIResponse MiniMax API 响应（OpenAI 兼容格式）
type MiniMaxAPIResponse struct {
	Error *struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
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

// MiniMaxStreamResponse MiniMax 流式响应（OpenAI 兼容格式）
type MiniMaxStreamResponse struct {
	Error *struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
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

// MiniMaxClient MiniMax 客户端（OpenAI 兼容）
type MiniMaxClient struct {
	APIKey  string
	BaseURL string
	Options *ClientOptions
	Model   string
}

// NewMiniMaxClient 创建 MiniMax 客户端
func NewMiniMaxClient(apiKey string) *MiniMaxClient {
	if apiKey == "" {
		apiKey = os.Getenv("MINIMAX_API_KEY")
	}
	return NewMiniMaxClientWithOptions(apiKey, DefaultClientOptions())
}

// NewMiniMaxClientWithOptions 创建带配置的 MiniMax 客户端
func NewMiniMaxClientWithOptions(apiKey string, options *ClientOptions) *MiniMaxClient {
	if options == nil {
		options = DefaultClientOptions()
	}
	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = miniMaxDefaultBaseURL
	}
	model := miniMaxDefaultModel
	if options.Model != "" {
		model = options.Model
	}
	return &MiniMaxClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Options: options,
		Model:   model,
	}
}

// SetModel 设置模型名称
func (c *MiniMaxClient) SetModel(model string) {
	c.Model = model
}

// GetModelName 获取模型名称
func (c *MiniMaxClient) GetModelName() string {
	return c.Model
}

// GetProvider 获取提供商名称
func (c *MiniMaxClient) GetProvider() string {
	return string(ProviderMiniMax)
}

// minimaxSanitizeMessages 确保发往 MiniMax 的 messages 里所有 tool_calls 的 arguments 均为合法 JSON 字符串（避免 2013 invalid function arguments）
func minimaxSanitizeMessages(msgs []Message) []Message {
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

// Chat 实现 LLMClient 接口
func (c *MiniMaxClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	msgs := minimaxSanitizeMessages(req.Messages)
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
		apiReq["temperature"] = 1.0 // MiniMax 推荐 1.0
	}

	timeout := c.Options.Timeout
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout
	}
	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       c.Options.MaxIdleConns,
			IdleConnTimeout:    c.Options.IdleConnTimeout,
			DisableCompression: true,
		},
	}

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
		logger.Infof(ctx, "[MiniMax] 发送请求, 请求体长度: %d", len(jsonData))
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	var apiResp MiniMaxAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}
	if apiResp.Error != nil {
		return nil, fmt.Errorf("MiniMax API错误: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
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

// ChatStream 实现流式聊天接口
func (c *MiniMaxClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	msgs := minimaxSanitizeMessages(req.Messages)
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
			apiReq["temperature"] = 1.0
		}

		timeout := c.Options.Timeout
		if req.Timeout != nil && *req.Timeout > 0 {
			timeout = *req.Timeout
		}
		httpClient := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       c.Options.MaxIdleConns,
				IdleConnTimeout:    c.Options.IdleConnTimeout,
				DisableCompression: true,
			},
		}

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
			var streamResp MiniMaxStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				chunkChan <- &StreamChunk{Error: fmt.Sprintf("解析流式响应失败: %v", err), Done: true}
				return
			}
			if streamResp.Error != nil {
				chunkChan <- &StreamChunk{Error: fmt.Sprintf("MiniMax API错误: %s - %s", streamResp.Error.Code, streamResp.Error.Message), Done: true}
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
