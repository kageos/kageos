package llms

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

// GLMThinkingDemo GLM思考模式演示
func GLMThinkingDemo() {
	fmt.Println("🧠 GLM-4.5 思考模式演示")
	fmt.Println("========================")

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

	// 演示思考模式
	demonstrateThinkingModeNew(glmClient)

	// 演示普通模式
	demonstrateNormalMode(glmClient)

	// 演示模式对比
	demonstrateModeComparison(glmClient)

	fmt.Println("\n✅ GLM思考模式演示完成！")
}

// demonstrateThinkingModeNew 演示思考模式（新版本）
func demonstrateThinkingModeNew(client *GLMClient) {
	fmt.Println("\n🤔 思考模式演示:")
	fmt.Println("问题: 请分析一下Go语言和Python语言在并发处理方面的区别")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下Go语言和Python语言在并发处理方面的区别，并给出使用建议。"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := client.ChatWithThinking(ctx, req, true)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 思考模式调用失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("❌ 思考模式API返回错误: %s\n", resp.Error)
		return
	}

	fmt.Printf("⏱️ 响应时间: %v\n", duration)
	fmt.Printf("📝 回复长度: %d 字符\n", len(resp.Content))
	if resp.Usage != nil {
		fmt.Printf("📊 Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
	fmt.Printf("💭 思考模式回复:\n%s\n", resp.Content)
}

// demonstrateNormalMode 演示普通模式
func demonstrateNormalMode(client *GLMClient) {
	fmt.Println("\n🚀 普通模式演示:")
	fmt.Println("问题: 请简单介绍一下Go语言的特点")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下Go语言的特点。"},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()
	resp, err := client.Chat(ctx, req)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("❌ 普通模式调用失败: %v\n", err)
		return
	}

	if resp.Error != "" {
		fmt.Printf("❌ 普通模式API返回错误: %s\n", resp.Error)
		return
	}

	fmt.Printf("⏱️ 响应时间: %v\n", duration)
	fmt.Printf("📝 回复长度: %d 字符\n", len(resp.Content))
	if resp.Usage != nil {
		fmt.Printf("📊 Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
	fmt.Printf("💬 普通模式回复:\n%s\n", resp.Content)
}

// demonstrateModeComparison 演示模式对比
func demonstrateModeComparison(client *GLMClient) {
	fmt.Println("\n🔄 模式对比演示:")
	fmt.Println("问题: 请分析一下为什么Go语言在并发编程方面比Python更有优势？")

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下为什么Go语言在并发编程方面比Python更有优势？"},
		},
		MaxTokens:   800,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 普通模式
	fmt.Println("\n🚀 普通模式:")
	start1 := time.Now()
	normalResp, err1 := client.Chat(ctx, req)
	duration1 := time.Since(start1)

	// 思考模式
	fmt.Println("\n🧠 思考模式:")
	start2 := time.Now()
	thinkingResp, err2 := client.ChatWithThinking(ctx, req, true)
	duration2 := time.Since(start2)

	// 对比结果
	fmt.Println("\n📊 对比结果:")
	fmt.Printf("普通模式 - 响应时间: %v, 回复长度: %d 字符\n", duration1, len(normalResp.Content))
	if normalResp.Usage != nil {
		fmt.Printf("普通模式 - Token使用: %d\n", normalResp.Usage.TotalTokens)
	}

	fmt.Printf("思考模式 - 响应时间: %v, 回复长度: %d 字符\n", duration2, len(thinkingResp.Content))
	if thinkingResp.Usage != nil {
		fmt.Printf("思考模式 - Token使用: %d\n", thinkingResp.Usage.TotalTokens)
	}

	// 分析差异
	if err1 == nil && err2 == nil {
		lengthDiff := len(thinkingResp.Content) - len(normalResp.Content)
		timeDiff := duration2 - duration1

		fmt.Printf("\n📈 差异分析:\n")
		fmt.Printf("回复长度差异: %+d 字符\n", lengthDiff)
		fmt.Printf("响应时间差异: %+v\n", timeDiff)

		if lengthDiff > 0 {
			fmt.Printf("✅ 思考模式产生了更详细的回复\n")
		} else {
			fmt.Printf("⚠️ 思考模式与普通模式回复长度相近\n")
		}

		if timeDiff > 0 {
			fmt.Printf("⏱️ 思考模式需要更多时间进行深度思考\n")
		} else {
			fmt.Printf("⚡ 思考模式响应时间与普通模式相近\n")
		}
	}
}

// GLMThinkingUsageGuide GLM思考模式使用指南
func GLMThinkingUsageGuide() {
	fmt.Println("📚 GLM思考模式使用指南")
	fmt.Println("======================")

	fmt.Println("\n🎯 什么时候使用思考模式？")
	fmt.Println("✅ 复杂推理任务 - 需要深度分析的问题")
	fmt.Println("✅ 技术对比分析 - 需要详细对比不同技术方案")
	fmt.Println("✅ 代码架构设计 - 需要全面考虑的设计问题")
	fmt.Println("✅ 问题诊断 - 需要深入分析的问题排查")
	fmt.Println("✅ 学习指导 - 需要详细解释的概念")

	fmt.Println("\n❌ 什么时候不使用思考模式？")
	fmt.Println("❌ 简单问答 - 直接回答的问题")
	fmt.Println("❌ 快速查询 - 需要快速响应的查询")
	fmt.Println("❌ 代码补全 - 简单的代码补全任务")
	fmt.Println("❌ 格式化输出 - 简单的格式化任务")

	fmt.Println("\n💡 使用建议:")
	fmt.Println("1. 对于复杂问题，启用思考模式可以获得更详细、更深入的回答")
	fmt.Println("2. 对于简单问题，使用普通模式可以更快获得响应")
	fmt.Println("3. 思考模式会消耗更多Token，但回答质量更高")
	fmt.Println("4. 可以根据问题复杂度动态选择模式")

	fmt.Println("\n🔧 代码示例:")
	fmt.Println("```go")
	fmt.Println("// 启用思考模式")
	fmt.Println("resp, err := glmClient.ChatWithThinking(ctx, req, true)")
	fmt.Println("")
	fmt.Println("// 禁用思考模式")
	fmt.Println("resp, err := glmClient.ChatWithThinking(ctx, req, false)")
	fmt.Println("")
	fmt.Println("// 普通模式（默认）")
	fmt.Println("resp, err := glmClient.Chat(ctx, req)")
	fmt.Println("```")
}
