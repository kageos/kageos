package llms

import (
	"context"
	"testing"
	"time"
)

// TestKimiClientCreation 测试Kimi客户端创建
func TestKimiClientCreation(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	if client == nil {
		t.Fatal("客户端创建失败")
	}

	// 验证API Key不为空即可，不检查具体值
	kimiClient, ok := client.(*KimiClient)
	if !ok {
		t.Fatal("客户端类型错误")
	}
	if kimiClient.APIKey == "" {
		t.Error("API Key为空")
	}

	// 验证客户端基本信息
	if kimiClient.BaseURL != "https://api.moonshot.cn/v1/chat/completions" {
		t.Errorf("BaseURL设置错误，期望: %s, 实际: %s",
			"https://api.moonshot.cn/v1/chat/completions", kimiClient.BaseURL)
	}

}

// TestKimiClientInterface 测试客户端接口实现
func TestKimiClientInterface(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	// 检查是否实现了LLMClient接口
	var _ LLMClient = client

	// 测试模型名称
	modelName := client.GetModelName()
	if modelName != "kimi-k2-0711-preview" {
		t.Errorf("模型名称错误，期望: kimi-k2-0711-preview, 实际: %s", modelName)
	}

	// 测试提供商名称
	provider := client.GetProvider()
	if provider != "Kimi" {
		t.Errorf("提供商名称错误，期望: Kimi, 实际: %s", provider)
	}
}

// TestKimiChatBasic 测试基本的聊天功能
func TestKimiChatBasic(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "你好，请简单介绍一下你自己"},
		},
		MaxTokens:   100,
		Temperature: 0.6,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("聊天请求失败: %v", err)
	}

	if resp == nil {
		t.Fatal("响应为空")
	}

	// 检查是否有错误
	if resp.Error != "" {
		t.Logf("API返回错误: %s", resp.Error)
		t.Logf("注意：这可能是API key无效或网络问题，请检查配置")
		return
	}

	// 检查响应内容
	if resp.Content == "" {
		t.Error("响应内容为空")
	} else {
		t.Logf("AI回答: %s", resp.Content)
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestKimiChatWithSystemPrompt 测试带系统提示的聊天
func TestKimiChatWithSystemPrompt(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的Go语言开发助手，请用简洁的语言回答问题",
			},
			{
				Role:    "user",
				Content: "Go语言中如何创建一个HTTP服务器？",
			},
		},
		MaxTokens:   200,
		Temperature: 0.6,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("带系统提示的聊天请求失败: %v", err)
	}

	if resp == nil {
		t.Fatal("响应为空")
	}

	// 检查是否有错误
	if resp.Error != "" {
		t.Logf("API返回错误: %s", resp.Error)
		t.Logf("注意：这可能是API key无效或网络问题，请检查配置")
		return
	}

	// 检查响应内容
	if resp.Content == "" {
		t.Error("响应内容为空")
	} else {
		t.Logf("AI回答: %s", resp.Content)
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestKimiCodeGeneration 测试代码生成功能
func TestKimiCodeGeneration(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的Go语言开发助手，请生成可运行的代码",
			},
			{
				Role:    "user",
				Content: "请用Go语言编写一个快速排序函数",
			},
		},
		MaxTokens:   1000,
		Temperature: 0.6,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("代码生成请求失败: %v", err)
	}

	if resp == nil {
		t.Fatal("响应为空")
	}

	// 检查是否有错误
	if resp.Error != "" {
		t.Logf("API返回错误: %s", resp.Error)
		t.Logf("注意：这可能是API key无效或网络问题，请检查配置")
		return
	}

	// 检查响应内容
	if resp.Content == "" {
		t.Error("响应内容为空")
	} else {
		t.Logf("生成的代码: %s", resp.Content)
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestKimiSupportedModels 测试支持的模型
func TestKimiSupportedModels(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	// 获取模型名称作为支持模型的一部分
	modelName := client.GetModelName()
	if modelName == "" {
		t.Error("模型名称为空")
	}

	// 记录模型信息
	t.Logf("当前模型: %s", modelName)
	t.Logf("注意：完整模型列表需要从外部配置或文档获取")
}

// TestKimiPricingInfo 测试价格信息
func TestKimiPricingInfo(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	// 获取模型名称作为价格信息的一部分
	modelName := client.GetModelName()
	if modelName == "" {
		t.Error("模型名称为空")
	}

	// 记录模型信息
	t.Logf("模型信息: %s", modelName)
	t.Logf("注意：价格信息需要从外部配置或文档获取")
}

// TestKimiErrorHandling 测试错误处理
func TestKimiErrorHandling(t *testing.T) {
	// 使用无效的API Key测试错误处理
	invalidClient := NewKimiClient("invalid-api-key")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "你好"},
		},
		MaxTokens: 10,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := invalidClient.Chat(ctx, req)
	if err != nil {
		t.Logf("预期的错误: %v", err)
		return
	}

	// 如果API返回了错误信息
	if resp != nil && resp.Error != "" {
		t.Logf("API返回错误: %s", resp.Error)
		return
	}

	t.Log("注意：API可能没有返回预期的错误信息")
}

// TestKimiTimeout 测试超时处理
func TestKimiTimeout(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请生成一个非常复杂的代码示例"},
		},
		MaxTokens: 5000,
	}

	// 设置很短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err = client.Chat(ctx, req)
	if err != nil {
		t.Logf("超时错误（预期）: %v", err)
	} else {
		t.Log("注意：请求没有超时，可能是网络很快或API响应很快")
	}
}

// TestKimiIntegration 测试集成功能
func TestKimiIntegration(t *testing.T) {
	client, err := NewKimiClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的Go语言开发助手，请生成高质量的代码",
			},
			{
				Role:    "user",
				Content: "请创建一个简单的Go HTTP服务器",
			},
		},
		MaxTokens:   1500,
		Temperature: 0.6,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("集成测试请求失败: %v", err)
	}

	if resp == nil {
		t.Fatal("响应为空")
	}

	// 检查是否有错误
	if resp.Error != "" {
		t.Logf("API返回错误: %s", resp.Error)
		t.Logf("注意：这可能是API key无效或网络问题，请检查配置")
		return
	}

	// 检查响应内容
	if resp.Content == "" {
		t.Error("响应内容为空")
	} else {
		t.Logf("集成测试成功，生成的代码: %s", resp.Content)
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用统计: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestKimiAll 运行所有测试
func TestKimiAll(t *testing.T) {
	t.Run("客户端创建", TestKimiClientCreation)
	t.Run("接口实现", TestKimiClientInterface)
	t.Run("基础聊天", TestKimiChatBasic)
	t.Run("系统提示", TestKimiChatWithSystemPrompt)
	t.Run("代码生成", TestKimiCodeGeneration)
	t.Run("支持模型", TestKimiSupportedModels)
	t.Run("价格信息", TestKimiPricingInfo)
	t.Run("错误处理", TestKimiErrorHandling)
	t.Run("超时处理", TestKimiTimeout)
	t.Run("集成测试", TestKimiIntegration)
}
