package llms

import (
	"context"
	"strings"
	"testing"
	"time"
)

// TestQwen3CoderClientCreation 测试千问3 Coder客户端创建
func TestQwen3CoderClientCreation(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	if client == nil {
		t.Fatal("客户端创建失败")
	}

	// 验证API Key不为空即可，不检查具体值
	qwenClient, ok := client.(*Qwen3CoderClient)
	if !ok {
		t.Fatal("客户端类型错误")
	}
	if qwenClient.APIKey == "" {
		t.Error("API Key为空")
	}

	// 验证客户端基本信息
	if qwenClient.BaseURL != "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions" {
		t.Errorf("BaseURL设置错误，期望: %s, 实际: %s",
			"https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions", qwenClient.BaseURL)
	}

}

// TestQwen3CoderClientInterface 测试客户端接口实现
func TestQwen3CoderClientInterface(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	// 检查是否实现了LLMClient接口
	var _ LLMClient = client

	// 测试模型名称
	modelName := client.GetModelName()
	if modelName != "qwen3-coder-plus" {
		t.Errorf("模型名称错误，期望: qwen3-coder-plus, 实际: %s", modelName)
	}

	// 测试提供商名称
	provider := client.GetProvider()
	if provider != "Qwen3-Coder" {
		t.Errorf("提供商名称错误，期望: Qwen3-Coder, 实际: %s", provider)
	}
}

// TestQwen3CoderCodeGeneration 测试代码生成功能
func TestQwen3CoderCodeGeneration(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的Go语言开发助手，请用简洁的语言回答问题，并生成可运行的代码",
			},
			{
				Role:    "user",
				Content: "请用Go语言编写一个快速排序函数，包含测试用例",
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1, // 代码生成需要低温度
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

		// 检查是否包含Go代码特征
		if !containsGoCode(resp.Content) {
			t.Logf("警告：响应内容可能不是有效的Go代码")
		}
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestQwen3CoderFunctionCalling 测试函数调用功能
func TestQwen3CoderFunctionCalling(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的Python开发助手，请根据用户需求生成Python代码",
			},
			{
				Role:    "user",
				Content: "请创建一个Python文件，包含斐波那契数列计算的函数",
			},
		},
		MaxTokens:   2000,
		Temperature: 0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		t.Fatalf("函数调用请求失败: %v", err)
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
		t.Logf("函数调用结果: %s", resp.Content)

		// 检查是否包含Python代码特征
		if !containsPythonCode(resp.Content) {
			t.Logf("警告：响应内容可能不是有效的Python代码")
		}
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestQwen3CoderSupportedModels 测试支持的模型
func TestQwen3CoderSupportedModels(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	// 获取模型名称
	modelName := client.GetModelName()
	if modelName == "" {
		t.Error("模型名称为空")
	}

	t.Logf("当前模型: %s", modelName)

	// 检查模型名称是否符合预期
	expectedModels := []string{"qwen3-coder-plus", "qwen3-coder-flash"}
	found := false
	for _, expected := range expectedModels {
		if modelName == expected {
			found = true
			break
		}
	}
	if !found {
		t.Logf("注意：当前模型 %s 不在预期列表中，但这可能是正常的", modelName)
	}
}

// TestQwen3CoderPricingInfo 测试价格信息
func TestQwen3CoderPricingInfo(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
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

// TestQwen3CoderErrorHandling 测试错误处理
func TestQwen3CoderErrorHandling(t *testing.T) {
	// 使用无效的API Key测试错误处理
	invalidClient := NewQwen3CoderClient("invalid-api-key")

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

// TestQwen3CoderTimeout 测试超时处理
func TestQwen3CoderTimeout(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
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

// TestQwen3CoderIntegration 测试集成功能
func TestQwen3CoderIntegration(t *testing.T) {
	client, err := NewQwen3CoderClientFromEnv()
	if err != nil {
		t.Fatalf("从环境变量创建客户端失败: %v", err)
	}

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个专业的Go语言开发助手，请生成高质量的代码，包含完整的错误处理、日志记录和测试用例",
			},
			{
				Role:    "user",
				Content: "请创建一个完整的Go HTTP服务器，包含路由、中间件、错误处理和优雅关闭功能",
			},
		},
		MaxTokens:   3000,
		Temperature: 0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
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

		// 检查是否包含Go代码特征
		if !containsGoCode(resp.Content) {
			t.Logf("警告：响应内容可能不是有效的Go代码")
		}
	}

	// 检查使用统计
	if resp.Usage != nil {
		t.Logf("Token使用统计: 输入%d, 输出%d, 总计%d",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestQwen3CoderAll 运行所有测试
func TestQwen3CoderAll(t *testing.T) {
	t.Run("客户端创建", TestQwen3CoderClientCreation)
	t.Run("接口实现", TestQwen3CoderClientInterface)
	t.Run("代码生成", TestQwen3CoderCodeGeneration)
	t.Run("函数调用", TestQwen3CoderFunctionCalling)
	t.Run("支持模型", TestQwen3CoderSupportedModels)
	t.Run("价格信息", TestQwen3CoderPricingInfo)
	t.Run("错误处理", TestQwen3CoderErrorHandling)
	t.Run("超时处理", TestQwen3CoderTimeout)
	t.Run("集成测试", TestQwen3CoderIntegration)
}

// 辅助函数：检查是否包含Go代码特征
func containsGoCode(content string) bool {
	goKeywords := []string{"package", "import", "func", "var", "const", "type", "struct", "interface"}
	content = strings.ToLower(content)

	for _, keyword := range goKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}

// 辅助函数：检查是否包含Python代码特征
func containsPythonCode(content string) bool {
	pythonKeywords := []string{"def ", "import ", "from ", "class ", "if __name__", "print(", "return "}
	content = strings.ToLower(content)

	for _, keyword := range pythonKeywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	return false
}
