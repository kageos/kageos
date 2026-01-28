package llms

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"io"
	"net/http"
	"os"
	"strings"
)

// DeepSeekAPIResponse DeepSeek API响应结构体
type DeepSeekAPIResponse struct {
	Error *struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
	} `json:"error,omitempty"`
	Choices []struct {
		Message struct {
			Content   string     `json:"content"`
			ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 工具调用列表
		} `json:"message"`
		FinishReason string `json:"finish_reason,omitempty"` // "tool_calls" 表示需要调用工具
	} `json:"choices,omitempty"`
	Usage *struct {
		PromptTokens     float64 `json:"prompt_tokens"`
		CompletionTokens float64 `json:"completion_tokens"`
		TotalTokens      float64 `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// DeepSeekStreamResponse DeepSeek 流式响应结构体
type DeepSeekStreamResponse struct {
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
			ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 流式输出中的工具调用
		} `json:"delta"`
		FinishReason *string `json:"finish_reason,omitempty"` // "tool_calls" 表示需要调用工具
	} `json:"choices,omitempty"`
	Usage *struct {
		PromptTokens     float64 `json:"prompt_tokens"`
		CompletionTokens float64 `json:"completion_tokens"`
		TotalTokens      float64 `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// DeepSeekClient DeepSeek客户端实现
type DeepSeekClient struct {
	APIKey  string         `json:"api_key"`
	BaseURL string         `json:"base_url"`
	Options *ClientOptions `json:"options"`
	Model   string         `json:"model"` // 🆕 添加模型名称字段
}

// NewDeepSeekClient 创建DeepSeek客户端（保持向后兼容）
func NewDeepSeekClient(apiKey string) *DeepSeekClient {
	// 如果传入的apiKey为空，尝试从环境变量获取
	if apiKey == "" {
		apiKey = os.Getenv("DEEPSEEK_API_KEY")
	}
	return NewDeepSeekClientWithOptions(apiKey, DefaultClientOptions())
}

// NewDeepSeekClientWithOptions 创建带配置的DeepSeek客户端
func NewDeepSeekClientWithOptions(apiKey string, options *ClientOptions) *DeepSeekClient {
	if options == nil {
		options = DefaultClientOptions()
	}

	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = "https://api.deepseek.com/v1/chat/completions"
	}

	// 🎯 不再在构造函数中创建固定的HTTP客户端
	// 而是在每次Chat请求时动态创建，以支持不同的超时时间

	// 设置模型（优先使用 options 中的模型，否则使用默认模型）
	model := "deepseek-reasoner" // 默认模型
	if options.Model != "" {
		model = options.Model
	}

	return &DeepSeekClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Options: options,
		Model:   model,
	}
}

// 🆕 SetModel 设置模型名称
func (d *DeepSeekClient) SetModel(model string) {
	d.Model = model
}

// 🆕 GetModelName 获取模型名称（现在返回实际设置的模型）
func (d *DeepSeekClient) GetModelName() string {
	return d.Model
}

// GetProvider 获取提供商名称
func (d *DeepSeekClient) GetProvider() string {
	return string(ProviderDeepSeek)
}

// Chat 实现LLMClient接口
func (d *DeepSeekClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 直接使用原始上下文，超时由HTTP客户端控制

	// 构造DeepSeek API请求
	apiReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}
	
	// 添加 tools 参数（如果提供）
	if len(req.Tools) > 0 {
		apiReq["tools"] = req.Tools
		// 如果指定了 tool_choice，也添加
		if req.ToolChoice != nil {
			apiReq["tool_choice"] = req.ToolChoice
		}
	}

	if req.Model == "" {
		apiReq["model"] = "deepseek-reasoner"
	}
	if req.MaxTokens <= 10 {
		apiReq["max_tokens"] = 4000
	}
	if req.Temperature == 0 {
		apiReq["temperature"] = 0.7
	}

	// 🎯 动态创建HTTP客户端，支持请求级别的超时配置
	timeout := d.Options.Timeout // 默认使用客户端配置的超时时间
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout // 如果请求中指定了超时时间，则使用请求的超时时间
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       d.Options.MaxIdleConns,
			IdleConnTimeout:    d.Options.IdleConnTimeout,
			DisableCompression: true,
		},
	}

	// 发送HTTP请求
	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", d.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)

	// 设置自定义User-Agent
	if d.Options != nil && d.Options.UserAgent != "" {
		httpReq.Header.Set("User-Agent", d.Options.UserAgent)
	}

	// 启用日志记录（优化：不打印完整请求体，只记录长度）
	if d.Options != nil && d.Options.EnableLogging {
		requestLen := len(jsonData)
		logger.Infof(ctx, "[DeepSeek] 发送请求到: %s, 请求体长度: %d", d.BaseURL, requestLen)
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var apiResp DeepSeekAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 记录响应日志（优化：不打印完整内容，只记录关键信息和长度）
	if len(apiResp.Choices) > 0 && apiResp.Choices[0].Message.Content != "" {
		contentLen := len(apiResp.Choices[0].Message.Content)
		logger.Infof(ctx, "[DeepSeek] 响应成功 - ContentLength: %d, Usage: %+v", contentLen, apiResp.Usage)
	} else {
		logger.Infof(ctx, "[DeepSeek] 响应 - ChoicesCount: %d, Usage: %+v", len(apiResp.Choices), apiResp.Usage)
	}

	// 检查错误
	if apiResp.Error != nil {
		return nil, fmt.Errorf("DeepSeek API错误: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	// 提取回答内容和工具调用
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("响应格式错误：没有找到choices")
	}

	choice := apiResp.Choices[0]
	content := choice.Message.Content
	toolCalls := choice.Message.ToolCalls

	// 提取使用统计
	var usage *Usage
	if apiResp.Usage != nil {
		usage = &Usage{
			PromptTokens:     int(apiResp.Usage.PromptTokens),
			CompletionTokens: int(apiResp.Usage.CompletionTokens),
			TotalTokens:      int(apiResp.Usage.TotalTokens),
		}
	}

	// 启用日志记录（优化：不打印完整内容）
	if d.Options != nil && d.Options.EnableLogging {
		logger.Infof(ctx, "[DeepSeek] 响应成功，内容长度: %d", len(content))
	}

	return &ChatResponse{
		Content:   content,
		ToolCalls: toolCalls,
		Usage:     usage,
	}, nil
}

// ChatStream 实现流式聊天接口
func (d *DeepSeekClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	// 创建流式响应通道
	chunkChan := make(chan *StreamChunk, 10) // 缓冲通道，避免阻塞

	// 在goroutine中处理流式请求
	go func() {
		defer close(chunkChan)

		// 构造DeepSeek API请求
		apiReq := map[string]interface{}{
			"model":       req.Model,
			"messages":    req.Messages,
			"max_tokens":  req.MaxTokens,
			"temperature": req.Temperature,
			"stream":      true, // 启用流式
		}
		
		// 添加 tools 参数（如果提供）
		if len(req.Tools) > 0 {
			apiReq["tools"] = req.Tools
			// 如果指定了 tool_choice，也添加
			if req.ToolChoice != nil {
				apiReq["tool_choice"] = req.ToolChoice
			}
		}

		// 设置默认值
		if req.Model == "" {
			apiReq["model"] = d.Model
		}
		if req.MaxTokens <= 0 {
			apiReq["max_tokens"] = 4000
		}
		if req.Temperature == 0 {
			apiReq["temperature"] = 0.7
		}

		// 动态创建HTTP客户端，支持请求级别的超时配置
		timeout := d.Options.Timeout
		if req.Timeout != nil && *req.Timeout > 0 {
			timeout = *req.Timeout
		}

		httpClient := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       d.Options.MaxIdleConns,
				IdleConnTimeout:    d.Options.IdleConnTimeout,
				DisableCompression: true,
			},
		}

		// 序列化请求
		jsonData, err := json.Marshal(apiReq)
		if err != nil {
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("序列化请求失败: %v", err),
				Done:  true,
			}
			return
		}

		// 记录请求日志
		if d.Options != nil && d.Options.EnableLogging {
			logger.Infof(ctx, "[DeepSeek] 流式请求: %s", string(jsonData))
		}

		// 创建HTTP请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", d.BaseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("创建HTTP请求失败: %v", err),
				Done:  true,
			}
			return
		}

		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+d.APIKey)
		if d.Options.UserAgent != "" {
			httpReq.Header.Set("User-Agent", d.Options.UserAgent)
		}

		// 发送请求
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("HTTP请求失败: %v", err),
				Done:  true,
			}
			return
		}
		defer resp.Body.Close()

		// 检查HTTP状态码
		if resp.StatusCode != http.StatusOK {

			all, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("HTTP请求失败，状态码: %d body:%s", resp.StatusCode, string(all)),
				Done:  true,
			}
			return
		}

		// 解析流式响应 - DeepSeek使用SSE格式
		scanner := bufio.NewScanner(resp.Body)
		var finalUsage *Usage
		chunkCount := 0

		for scanner.Scan() {
			line := scanner.Text()

			// 跳过SSE格式的注释行和空行
			if line == "" || strings.HasPrefix(line, ":") {
				continue
			}

			// 处理SSE数据行
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")

				// 检查是否是结束标记
				if data == "[DONE]" {
					chunkChan <- &StreamChunk{
						Usage: finalUsage,
						Done:  true,
					}
					break
				}

				// 解析JSON数据
				var streamResp DeepSeekStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					chunkChan <- &StreamChunk{
						Error: fmt.Sprintf("解析流式响应失败: %v", err),
						Done:  true,
					}
					return
				}

				chunkCount++

				// 检查错误
				if streamResp.Error != nil {
					chunkChan <- &StreamChunk{
						Error: fmt.Sprintf("DeepSeek API错误: %s - %s", streamResp.Error.Code, streamResp.Error.Message),
						Done:  true,
					}
					return
				}

				// 处理选择内容
				if len(streamResp.Choices) > 0 {
					choice := streamResp.Choices[0]

					// 发送内容片段
					if choice.Delta.Content != "" {
						chunkChan <- &StreamChunk{
							Content: choice.Delta.Content,
							Done:    false,
						}
					}

					// 发送工具调用（如果有）
					if len(choice.Delta.ToolCalls) > 0 {
						chunkChan <- &StreamChunk{
							ToolCalls: choice.Delta.ToolCalls,
							Done:      false,
						}
					}

					// 检查是否完成
					if choice.FinishReason != nil && *choice.FinishReason != "" {
						// 保存使用统计
						if streamResp.Usage != nil {
							finalUsage = &Usage{
								PromptTokens:     int(streamResp.Usage.PromptTokens),
								CompletionTokens: int(streamResp.Usage.CompletionTokens),
								TotalTokens:      int(streamResp.Usage.TotalTokens),
							}
						}

						// 发送完成信号（包含最终的 tool_calls，如果有）
						chunkChan <- &StreamChunk{
							ToolCalls: choice.Delta.ToolCalls, // 最后的 tool_calls（如果有）
							Usage:     finalUsage,
							Done:      true,
						}
						break
					}
				}
			}
		}
	}()

	return chunkChan, nil
}
