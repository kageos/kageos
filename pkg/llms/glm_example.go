package llms

import (
	"context"
	"fmt"
	"log"
	"time"
)

// GLMExample GLM使用示例
func GLMExample() {
	// 1. 创建GLM客户端
	client, err := NewLLMClient(ProviderGLM, "your-glm-api-key")
	if err != nil {
		log.Fatal("创建GLM客户端失败:", err)
	}

	// 2. 转换为GLM客户端以使用特殊功能
	glmClient, ok := client.(*GLMClient)
	if !ok {
		log.Fatal("客户端类型转换失败")
	}

	// 3. 设置模型
	glmClient.SetModel("glm-4.5") // 使用最强大的模型

	// 4. 创建对话请求
	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个专业的Go语言开发助手，擅长function-go框架开发。"},
			{Role: "user", Content: "请帮我创建一个学生选课系统，使用function-go框架。"},
		},
		MaxTokens:   4000,
		Temperature: 0.7,
	}

	// 5. 调用AI（普通模式）
	fmt.Println("=== 普通模式调用 ===")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		log.Printf("调用失败: %v", err)
		return
	}

	if resp.Error != "" {
		log.Printf("API错误: %s", resp.Error)
		return
	}

	fmt.Printf("AI回答: %s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}

	// 6. 使用思考模式（GLM特有功能）
	fmt.Println("\n=== 思考模式调用 ===")
	thinkingReq := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个专业的Go语言开发助手，擅长function-go框架开发。"},
			{Role: "user", Content: "请详细分析function-go框架的OnInputFuzzyMap功能，并给出最佳实践建议。"},
		},
		MaxTokens:   4000,
		Temperature: 0.7,
	}

	thinkingResp, err := glmClient.ChatWithThinking(ctx, thinkingReq, true)
	if err != nil {
		log.Printf("思考模式调用失败: %v", err)
		return
	}

	if thinkingResp.Error != "" {
		log.Printf("思考模式API错误: %s", thinkingResp.Error)
		return
	}

	fmt.Printf("思考模式回答: %s\n", thinkingResp.Content)
	if thinkingResp.Usage != nil {
		fmt.Printf("思考模式Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			thinkingResp.Usage.PromptTokens, thinkingResp.Usage.CompletionTokens, thinkingResp.Usage.TotalTokens)
	}

	// 7. 展示支持的模型
	fmt.Println("\n=== 支持的模型 ===")
	models := glmClient.GetSupportedModels()
	for i, model := range models {
		fmt.Printf("%d. %s", i+1, model)
		if model == "glm-4.5" {
			fmt.Print(" (最强大的推理模型，3550亿参数)")
		} else if model == "glm-4.5-air" {
			fmt.Print(" (高性价比轻量级强性能)")
		} else if model == "glm-4.5-flash" {
			fmt.Print(" (免费高效多功能)")
		}
		fmt.Println()
	}

	// 8. 检查思考模式支持
	fmt.Printf("\n思考模式支持: %v\n", glmClient.IsThinkingEnabled())
}

// GLMFromEnvExample 从环境变量创建GLM客户端的示例
func GLMFromEnvExample() {
	// 从环境变量创建客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		log.Printf("从环境变量创建GLM客户端失败: %v", err)
		log.Println("请设置环境变量: export GLM_API_KEY='your-api-key'")
		return
	}

	fmt.Println("成功从环境变量创建GLM客户端")
	fmt.Printf("提供商: %s\n", client.GetProvider())
	fmt.Printf("模型: %s\n", client.GetModelName())

	// 简单测试
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "你好，请简单介绍一下你自己。"},
		},
		MaxTokens: 500,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	resp, err := client.Chat(ctx, req)
	if err != nil {
		log.Printf("调用失败: %v", err)
		return
	}

	fmt.Printf("AI回答: %s\n", resp.Content)
}

// GLMWithOptionsExample 使用自定义配置的GLM客户端示例
func GLMWithOptionsExample() {
	// 创建自定义配置
	options := DefaultClientOptions().
		WithTimeout(60 * time.Second).
		WithBaseURL("https://open.bigmodel.cn/api/paas/v4/chat/completions").
		WithLogging()

	// 创建客户端
	client := NewGLMClientWithOptions("your-glm-api-key", options)

	fmt.Printf("GLM客户端创建成功\n")
	fmt.Printf("提供商: %s\n", client.GetProvider())
	fmt.Printf("模型: %s\n", client.GetModelName())
	fmt.Printf("BaseURL: %s\n", client.BaseURL)
	fmt.Printf("超时时间: %v\n", client.Options.Timeout)
	fmt.Printf("日志启用: %v\n", client.Options.EnableLogging)
}

// GLMModelComparisonExample 不同GLM模型对比示例
func GLMModelComparisonExample() {
	client := NewGLMClient("your-glm-api-key")

	// 测试不同模型
	models := []string{"glm-4.5", "glm-4.5-air", "glm-4.5-flash"}

	for _, model := range models {
		client.SetModel(model)
		fmt.Printf("\n=== 测试模型: %s ===\n", model)

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: "请用一句话介绍Go语言的特点。"},
			},
			MaxTokens: 100,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

		resp, err := client.Chat(ctx, req)
		cancel()

		if err != nil {
			fmt.Printf("模型 %s 调用失败: %v\n", model, err)
			continue
		}

		fmt.Printf("回答: %s\n", resp.Content)
		if resp.Usage != nil {
			fmt.Printf("Token使用: %d\n", resp.Usage.TotalTokens)
		}
	}
}
