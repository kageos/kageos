package llms

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// GLMUseThinkingDemo GLM UseThinking参数演示
func GLMUseThinkingDemo() {
	fmt.Println("🎯 GLM-4.5 UseThinking参数演示")
	fmt.Println("===============================")

	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		fmt.Println("❌ 请设置GLM_API_KEY环境变量")
		return
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		log.Fatal("创建GLM客户端失败:", err)
	}

	glmClient, ok := client.(*GLMClient)
	if !ok {
		log.Fatal("客户端类型转换失败")
	}

	// 演示UseThinking参数的使用
	demonstrateUseThinkingParameter(glmClient)

	// 演示不同场景的使用建议
	demonstrateUseThinkingScenarios(glmClient)

	fmt.Println("\n✅ GLM UseThinking参数演示完成！")
}

// demonstrateUseThinkingParameter 演示UseThinking参数的使用
func demonstrateUseThinkingParameter(client *GLMClient) {
	fmt.Println("\n🔧 UseThinking参数使用演示:")

	// 1. 启用思考模式
	fmt.Println("\n1️⃣ 启用思考模式 (UseThinking: true)")
	enableThinking := true
	req1 := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下Go语言和Python语言在并发处理方面的区别。"},
		},
		MaxTokens:   800,
		Temperature: 0.7,
		UseThinking: &enableThinking, // 启用思考模式
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start1 := time.Now()
	resp1, err1 := client.Chat(ctx, req1)
	duration1 := time.Since(start1)

	if err1 != nil {
		fmt.Printf("❌ 启用思考模式调用失败: %v\n", err1)
	} else {
		fmt.Printf("✅ 启用思考模式调用成功\n")
		fmt.Printf("⏱️ 响应时间: %v\n", duration1)
		fmt.Printf("📝 回复长度: %d 字符\n", len(resp1.Content))
		if resp1.Usage != nil {
			fmt.Printf("📊 Token使用: %d\n", resp1.Usage.TotalTokens)
		}
		fmt.Printf("💭 回复预览: %s...\n", resp1.Content[:min(200, len(resp1.Content))])
	}

	// 2. 禁用思考模式
	fmt.Println("\n2️⃣ 禁用思考模式 (UseThinking: false)")
	disableThinking := false
	req2 := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下Go语言的特点。"},
		},
		MaxTokens:   500,
		Temperature: 0.7,
		UseThinking: &disableThinking, // 禁用思考模式
	}

	start2 := time.Now()
	resp2, err2 := client.Chat(ctx, req2)
	duration2 := time.Since(start2)

	if err2 != nil {
		fmt.Printf("❌ 禁用思考模式调用失败: %v\n", err2)
	} else {
		fmt.Printf("✅ 禁用思考模式调用成功\n")
		fmt.Printf("⏱️ 响应时间: %v\n", duration2)
		fmt.Printf("📝 回复长度: %d 字符\n", len(resp2.Content))
		if resp2.Usage != nil {
			fmt.Printf("📊 Token使用: %d\n", resp2.Usage.TotalTokens)
		}
		fmt.Printf("💭 回复预览: %s...\n", resp2.Content[:min(200, len(resp2.Content))])
	}

	// 3. 默认模式（不设置UseThinking）
	fmt.Println("\n3️⃣ 默认模式 (UseThinking: nil)")
	req3 := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请介绍一下function-go框架的特点。"},
		},
		MaxTokens:   600,
		Temperature: 0.7,
		// UseThinking: nil, // 不设置，使用默认值
	}

	start3 := time.Now()
	resp3, err3 := client.Chat(ctx, req3)
	duration3 := time.Since(start3)

	if err3 != nil {
		fmt.Printf("❌ 默认模式调用失败: %v\n", err3)
	} else {
		fmt.Printf("✅ 默认模式调用成功\n")
		fmt.Printf("⏱️ 响应时间: %v\n", duration3)
		fmt.Printf("📝 回复长度: %d 字符\n", len(resp3.Content))
		if resp3.Usage != nil {
			fmt.Printf("📊 Token使用: %d\n", resp3.Usage.TotalTokens)
		}
		fmt.Printf("💭 回复预览: %s...\n", resp3.Content[:min(200, len(resp3.Content))])
	}
}

// demonstrateUseThinkingScenarios 演示不同场景的使用建议
func demonstrateUseThinkingScenarios(client *GLMClient) {
	fmt.Println("\n🎯 不同场景的使用建议:")

	scenarios := []struct {
		name        string
		question    string
		useThinking *bool
		reason      string
	}{
		{
			name:        "复杂技术分析",
			question:    "请详细分析微服务架构和单体架构的优缺点，并给出选择建议。",
			useThinking: boolPtr(true),
			reason:      "需要深度思考和分析，适合启用思考模式",
		},
		{
			name:        "简单问答",
			question:    "Go语言是什么时候发布的？",
			useThinking: boolPtr(false),
			reason:      "简单事实查询，不需要深度思考",
		},
		{
			name:        "代码生成",
			question:    "请写一个Go语言的Hello World程序。",
			useThinking: boolPtr(false),
			reason:      "简单的代码生成，普通模式即可",
		},
		{
			name:        "架构设计",
			question:    "请设计一个高并发的Web API架构，包括数据库选型、缓存策略、负载均衡等。",
			useThinking: boolPtr(true),
			reason:      "复杂的架构设计需要全面考虑，适合思考模式",
		},
		{
			name:        "学习指导",
			question:    "请解释一下Go语言的goroutine和channel的工作原理。",
			useThinking: boolPtr(true),
			reason:      "教学解释需要详细和深入，适合思考模式",
		},
	}

	for i, scenario := range scenarios {
		fmt.Printf("\n%d️⃣ %s\n", i+1, scenario.name)
		fmt.Printf("问题: %s\n", scenario.question)
		fmt.Printf("建议: %s\n", scenario.reason)
		fmt.Printf("UseThinking: %v\n", scenario.useThinking)

		req := &ChatRequest{
			Messages: []Message{
				{Role: "user", Content: scenario.question},
			},
			MaxTokens:   600,
			Temperature: 0.7,
			UseThinking: scenario.useThinking,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		start := time.Now()
		resp, err := client.Chat(ctx, req)
		duration := time.Since(start)
		cancel()

		if err != nil {
			fmt.Printf("❌ 调用失败: %v\n", err)
		} else {
			fmt.Printf("✅ 调用成功 - 响应时间: %v, 回复长度: %d 字符\n", duration, len(resp.Content))
			if resp.Usage != nil {
				fmt.Printf("📊 Token使用: %d\n", resp.Usage.TotalTokens)
			}
		}
	}
}

// GLMUseThinkingUsageGuide GLM UseThinking参数使用指南
func GLMUseThinkingUsageGuide() {
	fmt.Println("📚 GLM UseThinking参数使用指南")
	fmt.Println("==============================")

	fmt.Println("\n🎯 参数说明:")
	fmt.Println("UseThinking: *bool - 是否使用思考模式（可选参数）")
	fmt.Println("  - true:  启用思考模式，产生详细深入的回答")
	fmt.Println("  - false: 禁用思考模式，产生简洁快速的回答")
	fmt.Println("  - nil:   使用默认设置（启用思考模式）")

	fmt.Println("\n💡 使用建议:")
	fmt.Println("✅ 适合启用思考模式的场景:")
	fmt.Println("  - 复杂技术分析和对比")
	fmt.Println("  - 架构设计和系统规划")
	fmt.Println("  - 问题诊断和解决方案")
	fmt.Println("  - 学习指导和概念解释")
	fmt.Println("  - 需要深入思考的开放性问题")

	fmt.Println("\n❌ 适合禁用思考模式的场景:")
	fmt.Println("  - 简单事实查询")
	fmt.Println("  - 代码补全和格式化")
	fmt.Println("  - 快速问答和确认")
	fmt.Println("  - 简单的计算和转换")
	fmt.Println("  - 需要快速响应的场景")

	fmt.Println("\n🔧 代码示例:")
	fmt.Println("```go")
	fmt.Println("// 启用思考模式")
	fmt.Println("enableThinking := true")
	fmt.Println("req := &llms.ChatRequest{")
	fmt.Println("    Messages: []llms.Message{")
	fmt.Println("        {Role: \"user\", Content: \"复杂问题\"},")
	fmt.Println("    },")
	fmt.Println("    UseThinking: &enableThinking,")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// 禁用思考模式")
	fmt.Println("disableThinking := false")
	fmt.Println("req := &llms.ChatRequest{")
	fmt.Println("    Messages: []llms.Message{")
	fmt.Println("        {Role: \"user\", Content: \"简单问题\"},")
	fmt.Println("    },")
	fmt.Println("    UseThinking: &disableThinking,")
	fmt.Println("}")
	fmt.Println("")
	fmt.Println("// 默认模式（推荐）")
	fmt.Println("req := &llms.ChatRequest{")
	fmt.Println("    Messages: []llms.Message{")
	fmt.Println("        {Role: \"user\", Content: \"问题\"},")
	fmt.Println("    },")
	fmt.Println("    // UseThinking: nil, // 不设置，使用默认值")
	fmt.Println("}")
	fmt.Println("```")

	fmt.Println("\n⚡ 性能对比:")
	fmt.Println("思考模式: 响应时间较长，回答更详细，适合复杂问题")
	fmt.Println("普通模式: 响应时间较短，回答较简洁，适合简单问题")
	fmt.Println("默认模式: 平衡性能和效果，适合大多数场景")

	fmt.Println("\n🎯 最佳实践:")
	fmt.Println("1. 根据问题复杂度选择模式")
	fmt.Println("2. 复杂问题使用思考模式，简单问题使用普通模式")
	fmt.Println("3. 不确定时使用默认模式")
	fmt.Println("4. 可以通过回复长度判断模式是否生效")
	fmt.Println("5. 思考模式会消耗更多时间但提供更高质量的回答")
}

// boolPtr 辅助函数，返回bool指针
func boolPtr(b bool) *bool {
	return &b
}

// min 辅助函数，返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
