package llms

import (
	"context"
	"time"
)

// ClientOptions 客户端配置选项
type ClientOptions struct {
	// HTTP客户端配置
	Timeout         time.Duration `json:"timeout"`           // HTTP请求超时时间
	MaxIdleConns    int           `json:"max_idle_conns"`    // 最大空闲连接数
	IdleConnTimeout time.Duration `json:"idle_conn_timeout"` // 空闲连接超时时间

	// API配置
	BaseURL   string `json:"base_url"`   // 自定义BaseURL
	UserAgent string `json:"user_agent"` // 自定义User-Agent

	// LLM配置
	Model string `json:"model"` // 模型名称（可选，如果设置则覆盖客户端默认模型）

	// 重试配置
	MaxRetries int           `json:"max_retries"` // 最大重试次数
	RetryDelay time.Duration `json:"retry_delay"` // 重试延迟时间

	// 调试配置
	EnableLogging bool `json:"enable_logging"` // 是否启用日志
}

// DefaultClientOptions 返回默认的客户端配置
func DefaultClientOptions() *ClientOptions {
	return &ClientOptions{
		Timeout:         1200 * time.Second, // 默认1200秒超时（20分钟）
		MaxIdleConns:    10,                 // 默认10个空闲连接
		IdleConnTimeout: 90 * time.Second,   // 默认90秒空闲超时
		MaxRetries:      0,                  // 默认不重试
		RetryDelay:      1 * time.Second,    // 默认1秒重试延迟
		EnableLogging:   false,              // 默认不启用日志
	}
}

// WithTimeout 设置超时时间
func (o *ClientOptions) WithTimeout(timeout time.Duration) *ClientOptions {
	o.Timeout = timeout
	return o
}

// WithBaseURL 设置自定义BaseURL
func (o *ClientOptions) WithBaseURL(baseURL string) *ClientOptions {
	o.BaseURL = baseURL
	return o
}

// WithMaxRetries 设置最大重试次数
func (o *ClientOptions) WithMaxRetries(maxRetries int) *ClientOptions {
	o.MaxRetries = maxRetries
	return o
}

// WithLogging 启用日志
func (o *ClientOptions) WithLogging() *ClientOptions {
	o.EnableLogging = true
	return o
}

// WithModel 设置模型名称
func (o *ClientOptions) WithModel(model string) *ClientOptions {
	o.Model = model
	return o
}

// Message 对话消息结构
type Message struct {
	Role    string `json:"role"`    // system, user, assistant
	Content string `json:"content"` // 消息内容
}

// ChatRequest 聊天请求
type ChatRequest struct {
	Messages    []Message      `json:"messages"`               // 对话历史
	Model       string         `json:"model"`                  // 模型名称（可选）
	MaxTokens   int            `json:"max_tokens"`             // 最大token数（可选）
	Temperature float64        `json:"temperature"`            // 温度参数（可选）
	Timeout     *time.Duration `json:"timeout,omitempty"`      // 请求超时时间（可选，覆盖客户端默认超时）
	UseThinking *bool          `json:"use_thinking,omitempty"` // 是否使用思考模式（可选，GLM特有功能）
}

// ChatResponse 聊天响应
type ChatResponse struct {
	Content string `json:"content"` // AI回答内容
	Error   string `json:"error"`   // 错误信息（如果有）
	Usage   *Usage `json:"usage"`   // 使用统计（可选）
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`     // 输入token数
	CompletionTokens int `json:"completion_tokens"` // 输出token数
	TotalTokens      int `json:"total_tokens"`      // 总token数
}

// StreamChunk 流式响应数据块
type StreamChunk struct {
	Content string `json:"content"`         // 流式内容片段
	Done    bool   `json:"done"`            // 是否完成
	Error   string `json:"error,omitempty"` // 错误信息（如果有）
	Usage   *Usage `json:"usage,omitempty"` // 使用统计（完成时提供）
}

// LLMClient 大模型客户端接口
type LLMClient interface {
	// Chat 核心方法：根据对话列表返回AI回答
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

	// ChatStream 流式聊天方法：返回流式响应通道
	ChatStream(ctx context.Context, req *ChatRequest) (<-chan *StreamChunk, error)

	// GetModelName 获取模型名称
	GetModelName() string

	// GetProvider 获取提供商名称
	GetProvider() string
}
