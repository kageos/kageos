package llms

import (
	"context"
	"fmt"
	"time"
)

// QuickTestDeepSeek 快速测试DeepSeek API Key
func QuickTestDeepSeek(apiKey string) {
	fmt.Println("🚀 开始快速测试 DeepSeek API Key...")
	fmt.Printf("API Key: %s...\n", apiKey[:20])
	fmt.Println("==================================================")

	// 创建客户端
	client := NewDeepSeekClient(apiKey)

	// 测试基本聊天
	testBasicChat(client)

	// 测试系统提示
	testSystemPrompt(client)

	// 测试错误处理
	testErrorHandling()

	fmt.Println("==================================================")
	fmt.Println("✅ 快速测试完成！")
}

// QuickTestQwen3Coder 快速测试千问3 Coder API Key
func QuickTestQwen3Coder(apiKey string) {
	fmt.Println("🚀 开始快速测试 千问3 Coder API Key...")
	fmt.Printf("API Key: %s...\n", apiKey[:20])
	fmt.Println("==================================================")

	// 创建客户端
	client := NewQwen3CoderClient(apiKey)

	// 测试代码生成
	testCodeGeneration(client)

	// 测试函数调用
	testFunctionCalling(client)

	// 测试错误处理
	testQwen3CoderErrorHandling()

	fmt.Println("==================================================")
	fmt.Println("✅ 快速测试完成！")
}

func testBasicChat(client *DeepSeekClient) {
	fmt.Println("\n📝 测试1: 基本聊天功能")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "你好，请简单介绍一下你自己"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("⚠️  API返回错误: %s\n", resp.Error)
		fmt.Println("   这可能是API key无效或网络问题，请检查配置")
		return
	}

	if resp.Content == "" {
		fmt.Println("❌ 响应内容为空")
		return
	}

	fmt.Printf("✅ AI回答: %s\n", resp.Content)

	if resp.Usage != nil {
		fmt.Printf("📊 Token使用: 输入%d, 输出%d, 总计%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

func testSystemPrompt(client *DeepSeekClient) {
	fmt.Println("\n🎯 测试2: 系统提示功能")

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
		Temperature: 0.3,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("⚠️  API返回错误: %s\n", resp.Error)
		return
	}

	if resp.Content == "" {
		fmt.Println("❌ 响应内容为空")
		return
	}

	fmt.Printf("✅ AI回答: %s\n", resp.Content)
}

func testErrorHandling() {
	fmt.Println("\n🔍 测试3: 错误处理")

	// 测试无效的API key
	client := NewDeepSeekClient("invalid-api-key")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "测试"},
		},
		MaxTokens: 100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("✅ 预期错误: %v\n", err)
		return
	}

	if resp != nil && resp.Error != "" {
		fmt.Printf("✅ API返回错误: %s\n", resp.Error)
	} else {
		fmt.Println("⚠️  使用无效API key时，应该返回错误信息")
	}
}

// 千问3 Coder 测试函数
func testCodeGeneration(client *Qwen3CoderClient) {
	fmt.Println("\n📝 测试1: 代码生成功能")

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
		MaxTokens:   1500,
		Temperature: 0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("⚠️  API返回错误: %s\n", resp.Error)
		fmt.Println("   这可能是API key无效或网络问题，请检查配置")
		return
	}

	if resp.Content == "" {
		fmt.Println("❌ 响应内容为空")
		return
	}

	fmt.Printf("✅ 生成的代码: %s\n", resp.Content)

	if resp.Usage != nil {
		fmt.Printf("📊 Token使用: 输入%d, 输出%d, 总计%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

func testFunctionCalling(client *Qwen3CoderClient) {
	fmt.Println("\n🎯 测试2: 函数调用功能")

	req := &ChatRequest{
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一个代码助手，可以使用工具来读写文件",
			},
			{
				Role:    "user",
				Content: "请创建一个Python文件，包含一个计算斐波那契数列的函数",
			},
		},
		MaxTokens:   1500,
		Temperature: 0.1,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("⚠️  API返回错误: %s\n", resp.Error)
		return
	}

	if resp.Content == "" {
		fmt.Println("❌ 响应内容为空")
		return
	}

	fmt.Printf("✅ 函数调用结果: %s\n", resp.Content)
}

func testQwen3CoderErrorHandling() {
	fmt.Println("\n🔍 测试3: 错误处理")

	// 测试无效的API key
	client := NewQwen3CoderClient("invalid-api-key")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "测试"},
		},
		MaxTokens: 100,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("✅ 预期错误: %v\n", err)
		return
	}

	if resp != nil && resp.Error != "" {
		fmt.Printf("✅ API返回错误: %s\n", resp.Error)
	} else {
		fmt.Println("⚠️  使用无效API key时，应该返回错误信息")
	}
}
