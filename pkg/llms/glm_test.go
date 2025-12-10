package llms

import (
	"os"
	"testing"
	"time"
)

// TestGLMClientCreation 测试GLM客户端创建
func TestGLMClientCreation(t *testing.T) {
	// 测试基本创建
	client := NewGLMClient("test-api-key")
	if client == nil {
		t.Fatal("创建GLM客户端失败")
	}

	// 测试默认值
	if client.GetProvider() != "GLM" {
		t.Errorf("期望提供商为 GLM，实际为 %s", client.GetProvider())
	}

	if client.GetModelName() != "glm-4.5" {
		t.Errorf("期望默认模型为 glm-4.5，实际为 %s", client.GetModelName())
	}

	// 测试带配置的创建
	options := DefaultClientOptions().WithTimeout(30 * time.Second)
	clientWithOptions := NewGLMClientWithOptions("test-api-key", options)
	if clientWithOptions == nil {
		t.Fatal("创建带配置的GLM客户端失败")
	}

	if clientWithOptions.Options.Timeout != 30*time.Second {
		t.Errorf("期望超时时间为30秒，实际为 %v", clientWithOptions.Options.Timeout)
	}
}

// TestGLMModelSwitching 测试模型切换
func TestGLMModelSwitching(t *testing.T) {
	client := NewGLMClient("test-api-key")

	// 测试设置不同模型
	models := []string{"glm-4.5", "glm-4.5-air", "glm-4.5-x", "glm-4.5-airx", "glm-4.5-flash"}

	for _, model := range models {
		client.SetModel(model)
		if client.GetModelName() != model {
			t.Errorf("期望模型为 %s，实际为 %s", model, client.GetModelName())
		}
	}
}

// TestGLMSupportedModels 测试支持的模型列表
func TestGLMSupportedModels(t *testing.T) {
	client := NewGLMClient("test-api-key")
	models := client.GetSupportedModels()

	expectedModels := []string{
		"glm-4.5",
		"glm-4.5-air",
		"glm-4.5-x",
		"glm-4.5-airx",
		"glm-4.5-flash",
	}

	if len(models) != len(expectedModels) {
		t.Errorf("期望支持 %d 个模型，实际支持 %d 个", len(expectedModels), len(models))
	}

	for _, expectedModel := range expectedModels {
		found := false
		for _, model := range models {
			if model == expectedModel {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("期望支持的模型 %s 未找到", expectedModel)
		}
	}
}

// TestGLMThinkingMode 测试思考模式
func TestGLMThinkingMode(t *testing.T) {
	client := NewGLMClient("test-api-key")

	// 测试思考模式支持
	if !client.IsThinkingEnabled() {
		t.Error("GLM-4.5系列应该支持思考模式")
	}

	// 测试不同模型的思考模式支持
	models := client.GetSupportedModels()
	for _, model := range models {
		client.SetModel(model)
		if !client.IsThinkingEnabled() {
			t.Errorf("模型 %s 应该支持思考模式", model)
		}
	}
}

// TestGLMFromEnv 测试从环境变量创建客户端
func TestGLMFromEnv(t *testing.T) {
	// 设置测试环境变量
	originalKey := os.Getenv("GLM_API_KEY")
	defer func() {
		if originalKey != "" {
			os.Setenv("GLM_API_KEY", originalKey)
		} else {
			os.Unsetenv("GLM_API_KEY")
		}
	}()

	// 测试没有环境变量的情况
	os.Unsetenv("GLM_API_KEY")
	_, err := NewGLMClientFromEnv()
	if err == nil {
		t.Error("期望在没有环境变量时返回错误")
	}

	// 测试有环境变量的情况
	os.Setenv("GLM_API_KEY", "test-api-key")
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Errorf("期望从环境变量创建客户端成功，但得到错误: %v", err)
	}

	if client == nil {
		t.Fatal("期望创建客户端成功，但得到nil")
	}

	if client.GetProvider() != "GLM" {
		t.Errorf("期望提供商为 GLM，实际为 %s", client.GetProvider())
	}
}

// TestGLMFactoryIntegration 测试工厂模式集成
func TestGLMFactoryIntegration(t *testing.T) {
	// 测试通过工厂创建
	client, err := NewLLMClient(ProviderGLM, "test-api-key")
	if err != nil {
		t.Errorf("期望通过工厂创建GLM客户端成功，但得到错误: %v", err)
	}

	if client == nil {
		t.Fatal("期望创建客户端成功，但得到nil")
	}

	glmClient, ok := client.(*GLMClient)
	if !ok {
		t.Fatal("期望返回GLMClient类型")
	}

	if glmClient.GetProvider() != "GLM" {
		t.Errorf("期望提供商为 GLM，实际为 %s", glmClient.GetProvider())
	}
}

// TestGLMProviderConstants 测试提供商常量
func TestGLMProviderConstants(t *testing.T) {
	if ProviderGLM != "glm" {
		t.Errorf("期望ProviderGLM为 'glm'，实际为 %s", ProviderGLM)
	}

	// 测试提供商列表包含GLM
	providers := GetSupportedProviders()
	found := false
	for _, provider := range providers {
		if provider == ProviderGLM {
			found = true
			break
		}
	}
	if !found {
		t.Error("期望支持的提供商列表包含GLM")
	}

	// 测试显示名称
	displayName := GetProviderDisplayName(ProviderGLM)
	if displayName != "GLM" {
		t.Errorf("期望GLM显示名称为 'GLM'，实际为 %s", displayName)
	}
}

// TestGLMChatRequest 测试聊天请求结构
func TestGLMChatRequest(t *testing.T) {
	client := NewGLMClient("test-api-key")

	// 创建测试请求
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "你好，请介绍一下GLM-4.5模型"},
		},
		Model:       "glm-4.5",
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	// 测试请求参数设置
	if req.Model == "" {
		req.Model = client.GetModelName()
	}

	if req.Model != "glm-4.5" {
		t.Errorf("期望模型为 glm-4.5，实际为 %s", req.Model)
	}

	if req.MaxTokens != 1000 {
		t.Errorf("期望最大token数为1000，实际为 %d", req.MaxTokens)
	}

	if req.Temperature != 0.7 {
		t.Errorf("期望温度为0.7，实际为 %f", req.Temperature)
	}
}

// TestGLMThinkingConfig 测试思考模式配置
func TestGLMThinkingConfig(t *testing.T) {
	// 测试启用思考模式
	enabledConfig := &GLMThinkingConfig{Type: "enabled"}
	if enabledConfig.Type != "enabled" {
		t.Errorf("期望思考模式为enabled，实际为 %s", enabledConfig.Type)
	}

	// 测试禁用思考模式
	disabledConfig := &GLMThinkingConfig{Type: "disabled"}
	if disabledConfig.Type != "disabled" {
		t.Errorf("期望思考模式为disabled，实际为 %s", disabledConfig.Type)
	}
}

// TestGLMWithTimeout 测试带超时的请求
func TestGLMWithTimeout(t *testing.T) {
	// 创建带超时的请求
	timeout := 10 * time.Second
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "测试超时"},
		},
		Timeout: &timeout,
	}

	// 验证超时设置
	if req.Timeout == nil {
		t.Fatal("期望设置超时时间")
	}

	if *req.Timeout != 10*time.Second {
		t.Errorf("期望超时时间为10秒，实际为 %v", *req.Timeout)
	}
}

// TestGLMErrorHandling 测试错误处理
func TestGLMErrorHandling(t *testing.T) {
	// 测试空API Key
	client := NewGLMClient("")
	if client.APIKey != "" {
		t.Error("期望空API Key时客户端APIKey为空")
	}

	// 测试空模型名称
	client.SetModel("")
	if client.GetModelName() != "" {
		t.Error("期望设置空模型名称时返回空字符串")
	}
}

// TestGLMClientOptions 测试客户端选项
func TestGLMClientOptions(t *testing.T) {
	options := DefaultClientOptions()
	client := NewGLMClientWithOptions("test-api-key", options)

	// 测试默认选项
	if client.Options.Timeout != 60*time.Second {
		t.Errorf("期望默认超时时间为60秒，实际为 %v", client.Options.Timeout)
	}

	if client.Options.MaxIdleConns != 10 {
		t.Errorf("期望默认最大空闲连接数为10，实际为 %d", client.Options.MaxIdleConns)
	}

	if client.Options.IdleConnTimeout != 90*time.Second {
		t.Errorf("期望默认空闲连接超时时间为90秒，实际为 %v", client.Options.IdleConnTimeout)
	}

	// 测试自定义选项
	customOptions := &ClientOptions{
		Timeout:         30 * time.Second,
		MaxIdleConns:    20,
		IdleConnTimeout: 120 * time.Second,
		EnableLogging:   true,
	}

	customClient := NewGLMClientWithOptions("test-api-key", customOptions)
	if customClient.Options.Timeout != 30*time.Second {
		t.Errorf("期望自定义超时时间为30秒，实际为 %v", customClient.Options.Timeout)
	}

	if customClient.Options.MaxIdleConns != 20 {
		t.Errorf("期望自定义最大空闲连接数为20，实际为 %d", customClient.Options.MaxIdleConns)
	}

	if !customClient.Options.EnableLogging {
		t.Error("期望启用日志记录")
	}
}

// TestGLMBaseURL 测试BaseURL设置
func TestGLMBaseURL(t *testing.T) {
	// 测试默认BaseURL
	client := NewGLMClient("test-api-key")
	expectedURL := "https://open.bigmodel.cn/api/paas/v4/chat/completions"
	if client.BaseURL != expectedURL {
		t.Errorf("期望默认BaseURL为 %s，实际为 %s", expectedURL, client.BaseURL)
	}

	// 测试自定义BaseURL
	customURL := "https://custom-api.example.com/v1/chat/completions"
	options := DefaultClientOptions().WithBaseURL(customURL)
	customClient := NewGLMClientWithOptions("test-api-key", options)
	if customClient.BaseURL != customURL {
		t.Errorf("期望自定义BaseURL为 %s，实际为 %s", customURL, customClient.BaseURL)
	}
}
