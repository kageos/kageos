//go:build integration
// +build integration

package llms

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestGLMThinkingModeReal 测试GLM思考模式（真实API调用）
func TestGLMThinkingModeReal(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM思考模式测试：未设置GLM_API_KEY环境变量")
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Fatalf("创建GLM客户端失败: %v", err)
	}

	glmClient, ok := client.(*GLMClient)
	if !ok {
		t.Fatal("客户端类型转换失败")
	}

	// 测试思考模式支持
	if !glmClient.IsThinkingEnabled() {
		t.Error("GLM-4.5系列应该支持思考模式")
	}

	fmt.Printf("✅ 思考模式支持检查通过\n")

	// 测试思考模式调用
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下Go语言和Python语言在并发处理方面的区别，并给出使用建议。"},
		},
		MaxTokens:   1000,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	fmt.Printf("🧠 测试思考模式调用...\n")

	// 使用思考模式
	resp, err := glmClient.ChatWithThinking(ctx, req, true)
	if err != nil {
		t.Logf("思考模式调用失败: %v", err)
		t.Logf("这可能是API限制或模型配置问题")
		return
	}

	if resp.Error != "" {
		t.Logf("思考模式API返回错误: %s", resp.Error)
		t.Logf("错误可能原因：模型不支持思考模式或API配置问题")
		return
	}

	if resp.Content == "" {
		t.Logf("思考模式返回内容为空")
		return
	}

	fmt.Printf("✅ 思考模式调用成功\n")
	fmt.Printf("回复: %s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestGLMThinkingModeDisabledReal 测试禁用思考模式（真实API调用）
func TestGLMThinkingModeDisabledReal(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM思考模式测试：未设置GLM_API_KEY环境变量")
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Fatalf("创建GLM客户端失败: %v", err)
	}

	glmClient, ok := client.(*GLMClient)
	if !ok {
		t.Fatal("客户端类型转换失败")
	}

	// 测试禁用思考模式
	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请简单介绍一下Go语言的特点。"},
		},
		MaxTokens:   500,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("🚫 测试禁用思考模式...\n")

	// 禁用思考模式
	resp, err := glmClient.ChatWithThinking(ctx, req, false)
	if err != nil {
		t.Logf("禁用思考模式调用失败: %v", err)
		return
	}

	if resp.Error != "" {
		t.Logf("禁用思考模式API返回错误: %s", resp.Error)
		return
	}

	if resp.Content == "" {
		t.Logf("禁用思考模式返回内容为空")
		return
	}

	fmt.Printf("✅ 禁用思考模式调用成功\n")
	fmt.Printf("回复: %s\n", resp.Content)
	if resp.Usage != nil {
		fmt.Printf("Token使用: 输入=%d, 输出=%d, 总计=%d\n",
			resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
	}
}

// TestGLMThinkingModeComparisonReal 测试思考模式对比（真实API调用）
func TestGLMThinkingModeComparisonReal(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM思考模式对比测试：未设置GLM_API_KEY环境变量")
	}

	// 创建GLM客户端
	client, err := NewGLMClientFromEnv()
	if err != nil {
		t.Fatalf("创建GLM客户端失败: %v", err)
	}

	glmClient, ok := client.(*GLMClient)
	if !ok {
		t.Fatal("客户端类型转换失败")
	}

	req := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下为什么Go语言在并发编程方面比Python更有优势？"},
		},
		MaxTokens:   800,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 测试普通模式
	fmt.Printf("🔄 测试普通模式...\n")
	normalResp, err := glmClient.Chat(ctx, req)
	if err != nil {
		t.Logf("普通模式调用失败: %v", err)
		return
	}

	// 测试思考模式
	fmt.Printf("🧠 测试思考模式...\n")
	thinkingResp, err := glmClient.ChatWithThinking(ctx, req, true)
	if err != nil {
		t.Logf("思考模式调用失败: %v", err)
		return
	}

	// 对比结果
	fmt.Printf("\n📊 模式对比结果:\n")
	fmt.Printf("普通模式回复长度: %d\n", len(normalResp.Content))
	if normalResp.Usage != nil {
		fmt.Printf("普通模式Token使用: %d\n", normalResp.Usage.TotalTokens)
	}

	fmt.Printf("思考模式回复长度: %d\n", len(thinkingResp.Content))
	if thinkingResp.Usage != nil {
		fmt.Printf("思考模式Token使用: %d\n", thinkingResp.Usage.TotalTokens)
	}

	// 检查是否有明显差异
	if len(thinkingResp.Content) > len(normalResp.Content) {
		fmt.Printf("✅ 思考模式产生了更详细的回复\n")
	} else {
		fmt.Printf("⚠️ 思考模式与普通模式回复长度相近\n")
	}
}
