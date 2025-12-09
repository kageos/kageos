package llms

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// DouBaoClient 豆包客户端实现
type DouBaoClient struct {
	APIKey  string
	BaseURL string
	Options *ClientOptions
	Model   string // 模型名称
}

// DouBaoRequest 豆包 API请求结构
type DouBaoRequest struct {
	Model            string      `json:"model"`
	Messages         []Message   `json:"messages"`
	MaxTokens        int         `json:"max_tokens,omitempty"`
	Temperature      float64     `json:"temperature,omitempty"`
	TopP             float64     `json:"top_p,omitempty"`
	N                int         `json:"n,omitempty"`
	Stream           bool        `json:"stream,omitempty"`
	Stop             interface{} `json:"stop,omitempty"`
	PresencePenalty  float64     `json:"presence_penalty,omitempty"`
	FrequencyPenalty float64     `json:"frequency_penalty,omitempty"`
	ResponseFormat   interface{} `json:"response_format,omitempty"`
}

// DouBaoResponse 豆包 API响应结构
type DouBaoResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// NewDouBaoClient 创建豆包客户端（保持向后兼容）
func NewDouBaoClient(apiKey string) *DouBaoClient {
	return NewDouBaoClientWithOptions(apiKey, DefaultClientOptions())
}

// NewDouBaoClientWithOptions 创建带配置的豆包客户端
func NewDouBaoClientWithOptions(apiKey string, options *ClientOptions) *DouBaoClient {
	if options == nil {
		options = DefaultClientOptions()
	}

	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = "https://ark.cn-beijing.volces.com/api/v3/chat/completions"
	}

	// 设置模型（优先使用 options 中的模型，否则使用默认模型）
	model := "doubao-1-5-pro-32k-250115" // 默认模型
	if options.Model != "" {
		model = options.Model
	}

	return &DouBaoClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Options: options,
		Model:   model,
	}
}

// SetModel 设置模型名称
func (c *DouBaoClient) SetModel(model string) {
	c.Model = model
}

// Chat 实现LLMClient接口的Chat方法
func (c *DouBaoClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 验证请求
	if err := validateRequest(ctx, c.APIKey, req); err != nil {
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, err
	}

	// 转换为豆包 API请求格式
	douBaoReq := &DouBaoRequest{
		Model:       c.Model, // 使用客户端设置的模型
		Messages:    req.Messages,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	// 如果用户指定了模型，使用用户指定的模型
	if req.Model != "" {
		douBaoReq.Model = req.Model
	}

	// 设置默认值
	if douBaoReq.Temperature == 0 {
		douBaoReq.Temperature = 0.7 // 豆包推荐值
	}
	if douBaoReq.MaxTokens == 0 {
		douBaoReq.MaxTokens = 1024 // 默认值
	}

	// 序列化请求
	jsonData, err := json.Marshal(douBaoReq)
	if err != nil {
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	if c.Options != nil && c.Options.EnableLogging {
		logger.Infof(ctx, "[豆包] 发送请求到: %s, 请求体: %s", c.BaseURL, string(jsonData))
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)
	if c.Options != nil && c.Options.UserAgent != "" {
		httpReq.Header.Set("User-Agent", c.Options.UserAgent)
	}

	// 动态创建HTTP客户端，支持请求级别的超时配置
	timeout := c.Options.Timeout
	if req.Timeout != nil && *req.Timeout > 0 {
		timeout = *req.Timeout
	}
	httpClient := createHTTPClient(c.Options, timeout)

	// 发送请求
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("HTTP请求失败，状态码: %d, 响应: %s", resp.StatusCode, string(body))
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, err
	}

	// 解析响应
	var douBaoResp DouBaoResponse
	if err := json.Unmarshal(body, &douBaoResp); err != nil {
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查API错误
	if douBaoResp.Error != nil {
		err := fmt.Errorf("API错误: %s", douBaoResp.Error.Message)
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, err
	}

	// 转换为通用响应格式
	if len(douBaoResp.Choices) == 0 {
		err := fmt.Errorf("API响应中没有choices")
		if c.Options != nil && c.Options.EnableLogging {
			logger.Errorf(ctx, "[豆包] %v", err)
		}
		return nil, err
	}

	content := douBaoResp.Choices[0].Message.Content
	if c.Options != nil && c.Options.EnableLogging {
		logger.Infof(ctx, "[豆包] 响应成功，内容长度: %d", len(content))
	}

	chatResp := &ChatResponse{
		Content: content,
		Usage: &Usage{
			PromptTokens:     douBaoResp.Usage.PromptTokens,
			CompletionTokens: douBaoResp.Usage.CompletionTokens,
			TotalTokens:      douBaoResp.Usage.TotalTokens,
		},
	}

	return chatResp, nil
}

// GetModelName 获取模型名称
func (c *DouBaoClient) GetModelName() string {
	return c.Model
}

// GetProvider 获取提供商名称
func (c *DouBaoClient) GetProvider() string {
	return string(ProviderDouBao)
}

// ChatStream 实现流式聊天接口
func (c *DouBaoClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	// 创建流式响应通道
	chunkChan := make(chan *StreamChunk, 1)

	// 在goroutine中处理
	go func() {
		defer close(chunkChan)
		chunkChan <- &StreamChunk{
			Error: "豆包 客户端暂不支持流式响应，请使用 Chat 方法",
			Done:  true,
		}
	}()

	return chunkChan, nil
}

// GetSupportedModels 获取支持的模型列表
func (c *DouBaoClient) GetSupportedModels() []string {
	return []string{
		"doubao-seed-1-6-250615",
		"doubao-seed-1-6-vision-250815",
		"doubao-seed-1-6-flash-250715",
		"doubao-seed-1-6-flash-250615",
		"doubao-1-5-pro-32k-character-250715",
		"deepseek-v3-1-250821",
		"kimi-k2-250711",
		"deepseek-v3-250324",
	}
}

// GetPricingInfo 获取价格信息
func (c *DouBaoClient) GetPricingInfo() map[string]interface{} {
	return map[string]interface{}{
		"model":          "doubao-1-5-pro-32k-250115",
		"context_length": "32K",
		"input_price":    "0.0001元/1K tokens",
		"output_price":   "0.0002元/1K tokens",
		"note":           "价格信息仅供参考，请以官方最新定价为准",
	}
}
