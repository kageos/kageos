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

// GLMThinkingConfig GLM思考模式配置
type GLMThinkingConfig struct {
	Type string `json:"type"` // enabled 或 disabled
}

// GLMResponseFormat GLM响应格式配置
type GLMResponseFormat struct {
	Type string `json:"type"` // text 或 json_object
}

// GLMAPIRequest GLM API请求结构体
type GLMAPIRequest struct {
	Model          string             `json:"model"`
	Messages       []Message          `json:"messages"`
	MaxTokens      int                `json:"max_tokens,omitempty"`
	Temperature    float64            `json:"temperature,omitempty"`
	TopP           float64            `json:"top_p,omitempty"`
	DoSample       bool               `json:"do_sample,omitempty"`
	Stream         bool               `json:"stream,omitempty"`
	Thinking       *GLMThinkingConfig `json:"thinking,omitempty"`
	ResponseFormat *GLMResponseFormat `json:"response_format,omitempty"`
}

// GLMAPIResponse GLM API响应结构体
type GLMAPIResponse struct {
	Error *struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
	} `json:"error,omitempty"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices,omitempty"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// GLMStreamResponse GLM 流式响应结构体
type GLMStreamResponse struct {
	Error *struct {
		Code    string      `json:"code"`
		Message string      `json:"message"`
		Param   interface{} `json:"param"`
		Type    string      `json:"type"`
	} `json:"error,omitempty"`
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason,omitempty"`
	} `json:"choices,omitempty"`
	Usage *struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage,omitempty"`
}

// GLMClient GLM客户端实现
type GLMClient struct {
	APIKey  string         `json:"api_key"`
	BaseURL string         `json:"base_url"`
	Options *ClientOptions `json:"options"`
	Model   string         `json:"model"` // 模型名称
}

// NewGLMClient 创建GLM客户端（保持向后兼容）
func NewGLMClient(apiKey string) *GLMClient {
	// 如果传入的apiKey为空，尝试从环境变量获取
	if apiKey == "" {
		apiKey = os.Getenv("GLM_API_KEY")
	}
	return NewGLMClientWithOptions(apiKey, DefaultClientOptions())
}

// NewGLMClientWithOptions 创建带配置的GLM客户端
func NewGLMClientWithOptions(apiKey string, options *ClientOptions) *GLMClient {
	if options == nil {
		options = DefaultClientOptions()
	}
	if apiKey == "" {
		apiKey = os.Getenv("GLM_API_KEY")
	}

	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	}

	// 设置模型（优先使用 options 中的模型，否则使用默认模型）
	model := "glm-4.6" // 默认模型
	if options.Model != "" {
		model = options.Model
	}

	return &GLMClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Options: options,
		Model:   model,
	}
}

// SetModel 设置模型名称
func (g *GLMClient) SetModel(model string) {
	g.Model = model
}

// GetModelName 获取模型名称
func (g *GLMClient) GetModelName() string {
	return g.Model
}

// GetProvider 获取提供商名称
func (g *GLMClient) GetProvider() string {
	return string(ProviderGLM)
}

// Chat 实现LLMClient接口
func (g *GLMClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 构造GLM API请求
	apiReq := GLMAPIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false, // 默认非流式
		Thinking: &GLMThinkingConfig{
			Type: "disabled", // 默认启用思考模式
		},
	}

	// 设置默认值
	if apiReq.Model == "" {
		apiReq.Model = g.Model
	}
	if apiReq.MaxTokens <= 0 {
		apiReq.MaxTokens = 4096
	}
	if apiReq.Temperature == 0 {
		apiReq.Temperature = 0.6
	}

	// 根据请求参数控制思考模式
	if req.UseThinking != nil {
		if *req.UseThinking {
			apiReq.Thinking.Type = "enabled"
		} else {
			apiReq.Thinking.Type = "disabled"
		}
	}

	// 动态创建HTTP客户端，支持请求级别的超时配置
	timeout := g.Options.Timeout // 默认使用客户端配置的超时时间
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout // 如果请求中指定了超时时间，则使用请求的超时时间
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       g.Options.MaxIdleConns,
			IdleConnTimeout:    g.Options.IdleConnTimeout,
			DisableCompression: true,
		},
	}

	// 发送HTTP请求
	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)

	// 设置自定义User-Agent
	if g.Options != nil && g.Options.UserAgent != "" {
		httpReq.Header.Set("User-Agent", g.Options.UserAgent)
	}

	// 启用日志记录
	if g.Options != nil && g.Options.EnableLogging {
		fmt.Printf("[GLM] 请求体: %s\n", string(jsonData))
		logger.Errorf(ctx, "[GLM] 发送请求到:%s 请求体: %s\n", g.BaseURL, string(jsonData))
	}

	logger.Infof(ctx, "[GLM] 发送HTTP请求到: %s, API Key长度: %d", g.BaseURL, len(g.APIKey))
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		logger.Errorf(ctx, "[GLM] HTTP请求失败: %v", err)
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var apiResp GLMAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 记录响应日志
	jsonData, err = json.Marshal(apiResp)
	if err != nil {
		logger.Errorf(ctx, "[GLM] body 序列化失败")
		return nil, err
	}
	logger.Infof(ctx, "[GLM] body : %s", string(jsonData))

	// 检查错误
	if apiResp.Error != nil {
		return nil, fmt.Errorf("GLM API错误: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	// 提取回答内容
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("响应格式错误：没有找到choices")
	}

	content := apiResp.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("响应格式错误：content为空")
	}

	// 提取使用统计
	var usage *Usage
	if apiResp.Usage != nil {
		usage = &Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		}
	}

	// 启用日志记录
	if g.Options != nil && g.Options.EnableLogging {
		fmt.Printf("[GLM] 响应成功，内容长度: %d\n", len(content))
		logger.Infof(ctx, "[GLM] 响应成功，:%s 内容长度: %d\n", string(content), len(content))
	}

	return &ChatResponse{
		Content: content,
		Usage:   usage,
	}, nil
}

// ChatWithThinking 带思考模式的聊天（GLM特有功能）
func (g *GLMClient) ChatWithThinking(ctx context.Context, req *ChatRequest, enableThinking bool) (*ChatResponse, error) {
	// 构造GLM API请求
	apiReq := GLMAPIRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false, // 默认非流式
		Thinking: &GLMThinkingConfig{
			Type: "enabled", // 默认启用思考模式
		},
	}

	// 设置默认值
	if apiReq.Model == "" {
		apiReq.Model = g.Model
	}
	if apiReq.MaxTokens <= 0 {
		apiReq.MaxTokens = 4096
	}
	if apiReq.Temperature == 0 {
		apiReq.Temperature = 0.1
	}

	// 根据参数控制思考模式
	if !enableThinking {
		apiReq.Thinking.Type = "disabled"
	}

	// 动态创建HTTP客户端，支持请求级别的超时配置
	timeout := g.Options.Timeout // 默认使用客户端配置的超时时间
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout // 如果请求中指定了超时时间，则使用请求的超时时间
	}

	httpClient := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			MaxIdleConns:       g.Options.MaxIdleConns,
			IdleConnTimeout:    g.Options.IdleConnTimeout,
			DisableCompression: true,
		},
	}

	// 发送HTTP请求
	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", g.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)

	// 设置自定义User-Agent
	if g.Options != nil && g.Options.UserAgent != "" {
		httpReq.Header.Set("User-Agent", g.Options.UserAgent)
	}

	// 启用日志记录
	if g.Options != nil && g.Options.EnableLogging {
		fmt.Printf("[GLM] 思考模式请求体: %s\n", string(jsonData))
		logger.Errorf(ctx, "[GLM] 发送思考模式请求到:%s 请求体: %s\n", g.BaseURL, string(jsonData))
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var apiResp GLMAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 记录响应日志
	jsonData, err = json.Marshal(apiResp)
	if err != nil {
		logger.Errorf(ctx, "[GLM] 思考模式body 序列化失败")
		return nil, err
	}
	logger.Infof(ctx, "[GLM] 思考模式body : %s", string(jsonData))

	// 检查错误
	if apiResp.Error != nil {
		return nil, fmt.Errorf("GLM API错误: %s - %s", apiResp.Error.Code, apiResp.Error.Message)
	}

	// 提取回答内容
	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("响应格式错误：没有找到choices")
	}

	content := apiResp.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("响应格式错误：content为空")
	}

	// 提取使用统计
	var usage *Usage
	if apiResp.Usage != nil {
		usage = &Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		}
	}

	// 启用日志记录
	if g.Options != nil && g.Options.EnableLogging {
		fmt.Printf("[GLM] 思考模式响应成功，内容长度: %d\n", len(content))
		logger.Infof(ctx, "[GLM] 思考模式响应成功，:%s 内容长度: %d\n", string(content), len(content))
	}

	return &ChatResponse{
		Content: content,
		Usage:   usage,
	}, nil
}

// GetSupportedModels 获取支持的模型列表
func (g *GLMClient) GetSupportedModels() []string {
	return []string{
		"glm-4.5",       // 最强大的推理模型，3550亿参数
		"glm-4.5-air",   // 高性价比轻量级强性能
		"glm-4.5-x",     // 高性能强推理极速响应
		"glm-4.5-airx",  // 轻量级强性能极速响应
		"glm-4.5-flash", // 免费高效多功能
	}
}

// IsThinkingEnabled 检查当前模型是否支持思考模式
func (g *GLMClient) IsThinkingEnabled() bool {
	// GLM-4.5系列都支持思考模式
	supportedModels := g.GetSupportedModels()
	for _, model := range supportedModels {
		if g.Model == model {
			return true
		}
	}
	return false
}

// ChatStream 实现流式聊天接口
func (g *GLMClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	// 创建流式响应通道
	chunkChan := make(chan *StreamChunk, 10) // 缓冲通道，避免阻塞

	// 在goroutine中处理流式请求
	go func() {
		defer close(chunkChan)

		// 构造GLM API请求
		apiReq := GLMAPIRequest{
			Model:       req.Model,
			Messages:    req.Messages,
			MaxTokens:   req.MaxTokens,
			Temperature: req.Temperature,
			Stream:      true, // 启用流式
			Thinking: &GLMThinkingConfig{
				Type: "enabled", // 默认启用思考模式
			},
		}

		// 设置默认值
		if apiReq.Model == "" {
			apiReq.Model = g.Model
		}
		if apiReq.MaxTokens <= 0 {
			apiReq.MaxTokens = 4096
		}
		if apiReq.Temperature == 0 {
			apiReq.Temperature = 0.6
		}

		// 根据请求参数控制思考模式
		if req.UseThinking != nil {
			if *req.UseThinking {
				apiReq.Thinking.Type = "enabled"
			} else {
				apiReq.Thinking.Type = "disabled"
			}
		}

		// 动态创建HTTP客户端，支持请求级别的超时配置
		timeout := g.Options.Timeout
		if req.Timeout != nil && *req.Timeout > 0 {
			timeout = *req.Timeout
		}

		httpClient := &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:       g.Options.MaxIdleConns,
				IdleConnTimeout:    g.Options.IdleConnTimeout,
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
		if g.Options != nil && g.Options.EnableLogging {
			logger.Infof(ctx, "[GLM] 流式请求: %s", string(jsonData))
		}

		// 创建HTTP请求
		httpReq, err := http.NewRequestWithContext(ctx, "POST", g.BaseURL, bytes.NewBuffer(jsonData))
		if err != nil {
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("创建HTTP请求失败: %v", err),
				Done:  true,
			}
			return
		}

		// 设置请求头
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("Authorization", "Bearer "+g.APIKey)
		if g.Options.UserAgent != "" {
			httpReq.Header.Set("User-Agent", g.Options.UserAgent)
		}

		// 发送请求
		logger.Infof(ctx, "[GLM] 发送HTTP请求到: %s, API Key长度: %d", g.BaseURL, len(g.APIKey))
		resp, err := httpClient.Do(httpReq)
		if err != nil {
			logger.Errorf(ctx, "[GLM] HTTP请求失败: %v", err)
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("HTTP请求失败: %v", err),
				Done:  true,
			}
			return
		}
		defer resp.Body.Close()

		// 检查HTTP状态码
		logger.Infof(ctx, "[GLM] HTTP响应状态码: %d", resp.StatusCode)
		if resp.StatusCode != http.StatusOK {
			// 读取响应体获取详细错误信息
			body, _ := io.ReadAll(resp.Body)
			logger.Errorf(ctx, "[GLM] HTTP请求失败，状态码: %d, 响应体: %s", resp.StatusCode, string(body))
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("HTTP请求失败，状态码: %d", resp.StatusCode),
				Done:  true,
			}
			return
		}

		// 解析流式响应 - GLM使用SSE格式
		scanner := bufio.NewScanner(resp.Body)
		var finalUsage *Usage

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
				var streamResp GLMStreamResponse
				if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
					chunkChan <- &StreamChunk{
						Error: fmt.Sprintf("解析流式响应失败: %v", err),
						Done:  true,
					}
					return
				}

				// 检查错误
				if streamResp.Error != nil {
					chunkChan <- &StreamChunk{
						Error: fmt.Sprintf("GLM API错误: %s - %s", streamResp.Error.Code, streamResp.Error.Message),
						Done:  true,
					}
					return
				}

				// 处理选择内容
				if len(streamResp.Choices) > 0 {
					choice := streamResp.Choices[0]

					// 发送内容片段 - 优先使用reasoning_content（思考过程），其次使用content
					content := choice.Delta.ReasoningContent
					if content == "" {
						content = choice.Delta.Content
					}

					if content != "" {
						chunkChan <- &StreamChunk{
							Content: content,
							Done:    false,
						}
					}

					// 检查是否完成
					if choice.FinishReason != "" {
						// 保存使用统计
						if streamResp.Usage != nil {
							finalUsage = &Usage{
								PromptTokens:     streamResp.Usage.PromptTokens,
								CompletionTokens: streamResp.Usage.CompletionTokens,
								TotalTokens:      streamResp.Usage.TotalTokens,
							}
						}

						// 发送完成信号
						chunkChan <- &StreamChunk{
							Usage: finalUsage,
							Done:  true,
						}
						break
					}
				}
			}
		}

		// 检查scanner是否有错误
		if err := scanner.Err(); err != nil {
			chunkChan <- &StreamChunk{
				Error: fmt.Sprintf("读取流式响应失败: %v", err),
				Done:  true,
			}
			return
		}
	}()

	return chunkChan, nil
}
