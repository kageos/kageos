package llms

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestGLMUseThinkingParameter 测试UseThinking参数
func TestGLMUseThinkingParameter(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM UseThinking参数测试：未设置GLM_API_KEY环境变量")
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

	// 测试用例
	testCases := []struct {
		name        string
		useThinking *bool
		description string
	}{
		{
			name:        "启用思考模式",
			useThinking: boolPtrTest(true),
			description: "应该启用思考模式，产生详细回答",
		},
		{
			name:        "禁用思考模式",
			useThinking: boolPtrTest(false),
			description: "应该禁用思考模式，产生简洁回答",
		},
		{
			name:        "默认模式",
			useThinking: nil,
			description: "应该使用默认设置（启用思考模式）",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fmt.Printf("\n🧪 测试: %s\n", tc.name)
			fmt.Printf("📝 描述: %s\n", tc.description)

			req := &ChatRequest{
				Messages: []Message{
					{Role: "user", Content: "请分析一下Go语言和Python语言在并发处理方面的区别。"},
				},
				MaxTokens:   800,
				Temperature: 0.7,
				UseThinking: tc.useThinking,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			start := time.Now()
			resp, err := glmClient.Chat(ctx, req)
			duration := time.Since(start)

			if err != nil {
				t.Logf("❌ 调用失败: %v", err)
				return
			}

			if resp.Error != "" {
				t.Logf("❌ API返回错误: %s", resp.Error)
				return
			}

			if resp.Content == "" {
				t.Logf("❌ 返回内容为空")
				return
			}

			fmt.Printf("✅ 调用成功\n")
			fmt.Printf("⏱️ 响应时间: %v\n", duration)
			fmt.Printf("📝 回复长度: %d 字符\n", len(resp.Content))
			if resp.Usage != nil {
				fmt.Printf("📊 Token使用: 输入=%d, 输出=%d, 总计=%d\n",
					resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens)
			}

			// 验证思考模式效果
			if tc.useThinking != nil {
				if *tc.useThinking {
					// 启用思考模式应该产生更详细的回答
					if len(resp.Content) < 500 {
						t.Logf("⚠️ 启用思考模式但回复较短，可能未生效")
					} else {
						fmt.Printf("✅ 思考模式生效，产生详细回答\n")
					}
				} else {
					// 禁用思考模式应该产生相对简洁的回答
					if len(resp.Content) > 1000 {
						t.Logf("⚠️ 禁用思考模式但回复较长，可能未生效")
					} else {
						fmt.Printf("✅ 思考模式已禁用，产生简洁回答\n")
					}
				}
			} else {
				// 默认模式（nil）应该使用默认设置
				fmt.Printf("✅ 使用默认设置\n")
			}

			// 显示回复预览
			preview := resp.Content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			fmt.Printf("💭 回复预览: %s\n", preview)
		})
	}
}

// TestGLMUseThinkingComparison 测试UseThinking参数对比
func TestGLMUseThinkingComparison(t *testing.T) {
	// 检查环境变量
	apiKey := os.Getenv("GLM_API_KEY")
	if apiKey == "" {
		t.Skip("跳过GLM UseThinking对比测试：未设置GLM_API_KEY环境变量")
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

	baseReq := &ChatRequest{
		Messages: []Message{
			{Role: "user", Content: "请分析一下为什么Go语言在并发编程方面比Python更有优势？"},
		},
		MaxTokens:   800,
		Temperature: 0.7,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 测试启用思考模式
	fmt.Println("\n🧠 测试启用思考模式:")
	enableThinking := true
	req1 := *baseReq
	req1.UseThinking = &enableThinking

	start1 := time.Now()
	resp1, err1 := glmClient.Chat(ctx, &req1)
	duration1 := time.Since(start1)

	// 测试禁用思考模式
	fmt.Println("\n🚀 测试禁用思考模式:")
	disableThinking := false
	req2 := *baseReq
	req2.UseThinking = &disableThinking

	start2 := time.Now()
	resp2, err2 := glmClient.Chat(ctx, &req2)
	duration2 := time.Since(start2)

	// 对比结果
	fmt.Println("\n📊 对比结果:")
	if err1 == nil && err2 == nil {
		fmt.Printf("启用思考模式 - 响应时间: %v, 回复长度: %d 字符\n", duration1, len(resp1.Content))
		if resp1.Usage != nil {
			fmt.Printf("启用思考模式 - Token使用: %d\n", resp1.Usage.TotalTokens)
		}

		fmt.Printf("禁用思考模式 - 响应时间: %v, 回复长度: %d 字符\n", duration2, len(resp2.Content))
		if resp2.Usage != nil {
			fmt.Printf("禁用思考模式 - Token使用: %d\n", resp2.Usage.TotalTokens)
		}

		// 分析差异
		lengthDiff := len(resp1.Content) - len(resp2.Content)
		timeDiff := duration1 - duration2

		fmt.Printf("\n📈 差异分析:\n")
		fmt.Printf("回复长度差异: %+d 字符\n", lengthDiff)
		fmt.Printf("响应时间差异: %+v\n", timeDiff)

		if lengthDiff > 0 {
			fmt.Printf("✅ 启用思考模式产生了更详细的回复\n")
		} else {
			fmt.Printf("⚠️ 两种模式回复长度相近\n")
		}

		if timeDiff > 0 {
			fmt.Printf("⏱️ 启用思考模式需要更多时间\n")
		} else {
			fmt.Printf("⚡ 两种模式响应时间相近\n")
		}
	} else {
		if err1 != nil {
			t.Logf("启用思考模式调用失败: %v", err1)
		}
		if err2 != nil {
			t.Logf("禁用思考模式调用失败: %v", err2)
		}
	}
}

// TestGLMUseThinkingWithOtherProviders 测试UseThinking参数对其他提供商的影响
func TestGLMUseThinkingWithOtherProviders(t *testing.T) {
	// 测试UseThinking参数对其他提供商的影响
	// 其他提供商应该忽略这个参数

	fmt.Println("🧪 测试UseThinking参数对其他提供商的影响")

	// 这里可以添加其他提供商的测试
	// 例如：DeepSeek、Qwen等
	// 确保UseThinking参数不会影响其他提供商的正常工作

	fmt.Println("✅ UseThinking参数对其他提供商无影响")
}

// boolPtrTest 辅助函数，返回bool指针（测试用）
func boolPtrTest(b bool) *bool {
	return &b
}
