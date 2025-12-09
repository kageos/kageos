package llms

import (
	"context"
	"fmt"
	"log"
	"time"
)

func GLMDemo() {
	fmt.Println("🚀 GLM-4.5 模型演示")
	fmt.Println("==================")

	// 1. 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		log.Fatal("创建GLM客户端失败:", err)
	}

	// 2. 转换为GLM客户端以使用特殊功能
	glmClient, ok := client.(*GLMClient)
	if !ok {
		log.Fatal("客户端类型转换失败")
	}

	// 3. 展示支持的模型
	fmt.Println("\n📋 支持的模型:")
	models := glmClient.GetSupportedModels()
	for i, model := range models {
		fmt.Printf("  %d. %s", i+1, model)
		switch model {
		case "glm-4.5":
			fmt.Print(" (最强大的推理模型，3550亿参数)")
		case "glm-4.5-air":
			fmt.Print(" (高性价比轻量级强性能)")
		case "glm-4.5-x":
			fmt.Print(" (高性能强推理极速响应)")
		case "glm-4.5-airx":
			fmt.Print(" (轻量级强性能极速响应)")
		case "glm-4.5-flash":
			fmt.Print(" (免费高效多功能)")
		}
		fmt.Println()
	}

	// 4. 基本聊天功能演示
	fmt.Println("\n💬 基本聊天功能演示:")
	demonstrateBasicChat(glmClient)

	// 5. 思考模式演示
	fmt.Println("\n🤔 思考模式演示:")
	demonstrateThinkingMode(glmClient)

	// 6. 不同模型对比演示
	fmt.Println("\n🔄 不同模型对比演示:")
	demonstrateModelComparison(glmClient)

	// 7. 代码生成演示
	fmt.Println("\n💻 代码生成演示:")
	demonstrateCodeGeneration(glmClient)

	fmt.Println("\n✅ GLM演示完成！")
}

// demonstrateBasicChat 演示基本聊天功能
func demonstrateBasicChat(client *GLMClient) {
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下GLM-4.5模型的特点和优势。"},
		},
		MaxTokens:   800,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ 聊天调用失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("❌ API返回错误: %s\n", resp.Error)
		return
	}

	fmt.Printf("📝 回复: %s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("📊 Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// demonstrateThinkingMode 演示思考模式
func demonstrateThinkingMode(client *GLMClient) {
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下Go语言和Python语言在并发处理方面的区别，并给出使用建议。"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 使用思考模式
	resp, err := client.ChatWithThinking(ctx, req, true)
	if err != nil {
		fmt.Printf("❌ 思考模式调用失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("❌ 思考模式API返回错误: %s\n", resp.Error)
		return
	}

	fmt.Printf("🧠 思考模式回复: %s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("📊 思考模式Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// demonstrateModelComparison 演示不同模型对比
func demonstrateModelComparison(client *GLMClient) {
	models := []string{"glm-4.5", "glm-4.5-air", "glm-4.5-flash"}

	for _, model := range models {
		fmt.Printf("\n🔍 测试模型: %s\n", model)
		client.SetModel(model)

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "请用一句话介绍Go语言的特点。"},
			},
			MaxTokens:   100,
			Temperature: 0.7,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := client.Chat(ctx, req)
		if err != nil {
			fmt.Printf("❌ 模型 %s 调用失败: %v\n", model, err)
			continue
		}

		if resp.Error != "" {
			fmt.Printf("❌ 模型 %s API返回错误: %s\n", model, resp.Error)
			continue
		}

		if resp.Content == "" {
			fmt.Printf("⚠️ 模型 %s 返回内容为空\n", model)
			continue
		}

		fmt.Printf("✅ 模型 %s 回复: %s\n", model, resp.Content)
		if resp.Usage != nil {
			fmt.Printf("📊 Token使用: %d\n", resp.Usage.TotalTokens)
		}
	}
}

// demonstrateCodeGeneration 演示代码生成
func demonstrateCodeGeneration(client *GLMClient) {
	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个专业的Go语言开发助手，擅长function-go框架开发。"},
			{Role: "user", Content: "请帮我创建一个简单的学生信息管理系统的数据模型，使用function-go框架的规范。"},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		fmt.Printf("❌ 代码生成调用失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("❌ 代码生成API返回错误: %s\n", resp.Error)
		return
	}

	fmt.Printf("💻 生成的代码:\n%s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("📊 Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}
