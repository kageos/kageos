package llms

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

// Qwen3CoderClient 千问3 Coder客户端实现
type Qwen3CoderClient struct {
	APIKey  string         `json:"api_key"`
	BaseURL string         `json:"base_url"`
	Options *ClientOptions `json:"options"`
	Model   string         `json:"model"` // 模型名称
}

// NewQwen3CoderClient 创建千问3 Coder客户端（保持向后兼容）
func NewQwen3CoderClient(apiKey string) *Qwen3CoderClient {
	return NewQwen3CoderClientWithOptions(apiKey, DefaultClientOptions())
}

// NewQwen3CoderClientWithOptions 创建带配置的千问3 Coder客户端
func NewQwen3CoderClientWithOptions(apiKey string, options *ClientOptions) *Qwen3CoderClient {
	if options == nil {
		options = DefaultClientOptions()
	}

	baseURL := options.BaseURL
	if baseURL == "" {
		baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	}

	// 设置模型（优先使用 options 中的模型，否则使用默认模型）
	model := "qwen3-coder-plus" // 默认模型
	if options.Model != "" {
		model = options.Model
	}

	return &Qwen3CoderClient{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Options: options,
		Model:   model,
	}
}

// SetModel 设置模型名称
func (q *Qwen3CoderClient) SetModel(model string) {
	q.Model = model
}

// Chat 实现LLMClient接口
func (q *Qwen3CoderClient) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 验证请求
	if err := validateRequest(ctx, q.APIKey, req); err != nil {
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, err
	}

	// 构造千问3 Coder API请求
	apiReq := map[string]interface{}{
		"model":       req.Model,
		"messages":    req.Messages,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
	}

	// 设置默认值
	if req.Model == "" {
		apiReq["model"] = q.Model // 使用客户端设置的模型
	}
	if req.MaxTokens == 0 {
		apiReq["max_tokens"] = 8000 // 代码生成需要更多token
	}
	if req.Temperature == 0 {
		apiReq["temperature"] = 0.1 // 代码生成需要更低的温度，提高准确性
	}

	// 发送HTTP请求
	jsonData, err := json.Marshal(apiReq)
	if err != nil {
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, fmt.Errorf("序列化请求失败: %v", err)
	}

	// 启用日志记录（优化：不打印完整请求体，只记录长度）
	if q.Options != nil && q.Options.EnableLogging {
		requestLen := len(jsonData)
		logger.Infof(ctx, "[Qwen3Coder] 发送请求到: %s, 请求体长度: %d", q.BaseURL, requestLen)
	}

	httpReq, err := newBearerJSONRequest(ctx, q.BaseURL, q.APIKey, jsonData, q.Options)
	if err != nil {
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, fmt.Errorf("创建HTTP请求失败: %v", err)
	}

	httpClient := createHTTPClient(q.Options, resolveRequestTimeout(q.Options, req))

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 解析响应
	var apiResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查错误
	if errMsg, exists := apiResp["error"]; exists {
		err := fmt.Errorf("API错误: %v", errMsg)
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, err
	}

	// 提取回答内容
	choices, ok := apiResp["choices"].([]interface{})
	if !ok || len(choices) == 0 {
		err := fmt.Errorf("响应格式错误：没有找到choices")
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, err
	}

	choice, ok := choices[0].(map[string]interface{})
	if !ok {
		err := fmt.Errorf("响应格式错误：choice格式错误")
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, err
	}

	message, ok := choice["message"].(map[string]interface{})
	if !ok {
		err := fmt.Errorf("响应格式错误：message格式错误")
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, err
	}

	content, ok := message["content"].(string)
	if !ok {
		err := fmt.Errorf("响应格式错误：content格式错误")
		if q.Options != nil && q.Options.EnableLogging {
			logger.Errorf(ctx, "[Qwen3Coder] %v", err)
		}
		return nil, err
	}

	// 检查是否有工具调用
	var toolCalls []interface{}
	if toolCallsData, exists := message["tool_calls"]; exists {
		if toolCallsArray, ok := toolCallsData.([]interface{}); ok {
			toolCalls = toolCallsArray
		}
	}

	// 提取使用统计
	var usage *Usage
	if usageData, exists := apiResp["usage"]; exists {
		if usageMap, ok := usageData.(map[string]interface{}); ok {
			usage = &Usage{
				PromptTokens:     int(usageMap["prompt_tokens"].(float64)),
				CompletionTokens: int(usageMap["completion_tokens"].(float64)),
				TotalTokens:      int(usageMap["total_tokens"].(float64)),
			}
		}
	}

	// 如果有工具调用，在内容中添加提示
	if len(toolCalls) > 0 {
		content += "\n\n🔧 检测到工具调用，请查看tool_calls字段获取详细信息"
	}

	if q.Options != nil && q.Options.EnableLogging {
		logger.Infof(ctx, "[Qwen3Coder] 响应成功，内容长度: %d", len(content))
	}

	return &ChatResponse{
		Content: content,
		Usage:   usage,
		// 可以在这里添加工具调用信息
	}, nil
}

// GetModelName 获取模型名称
func (q *Qwen3CoderClient) GetModelName() string {
	return q.Model
}

// GetProvider 获取提供商名称
func (q *Qwen3CoderClient) GetProvider() string {
	return string(ProviderQwen3Coder)
}

// ChatStream 实现流式聊天接口
func (q *Qwen3CoderClient) ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error) {
	// 创建流式响应通道
	chunkChan := make(chan *StreamChunk, 1)

	// 在goroutine中处理
	go func() {
		defer close(chunkChan)
		chunkChan <- &StreamChunk{
			Error: "Qwen3Coder 客户端暂不支持流式响应，请使用 Chat 方法",
			Done:  true,
		}
	}()

	return chunkChan, nil
}

// GetSupportedModels 获取支持的模型列表
func (q *Qwen3CoderClient) GetSupportedModels() []string {
	return []string{
		"qwen3-coder-plus",
		"qwen3-coder-plus-2025-07-22",
		"qwen3-coder-flash",
		"qwen3-coder-flash-2025-07-28",
	}
}

// GetPricingInfo 获取价格信息
func (q *Qwen3CoderClient) GetPricingInfo() map[string]interface{} {
	return map[string]interface{}{
		"model": "qwen3-coder-plus",
		"pricing": map[string]interface{}{
			"0-32K": map[string]interface{}{
				"input":  "0.004元/千Token",
				"output": "0.016元/千Token",
			},
			"32K-128K": map[string]interface{}{
				"input":  "0.0042元/千Token",
				"output": "0.0168元/千Token",
			},
			"128K-256K": map[string]interface{}{
				"input":  "0.005元/千Token",
				"output": "0.02元/千Token",
			},
			"256K-1M": map[string]interface{}{
				"input":  "0.01元/千Token",
				"output": "0.1元/千Token",
			},
		},
		"free_quota":     "各100万Token（百炼开通后180天内）",
		"context_length": "1,000,000 Token",
		"max_output":     "65,536 Token",
	}
}
