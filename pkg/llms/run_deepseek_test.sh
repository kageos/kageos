#!/bin/bash

# LLM API Key 测试脚本
echo "🚀 开始测试 LLM API Keys..."
echo "=================================================="

# 进入测试目录
cd "$(dirname "$0")"

# 测试 DeepSeek
echo "🔍 测试 DeepSeek API Key..."
go test -v -run TestDeepSeekChatBasic

echo ""
echo "🔍 测试 千问3 Coder API Key..."
go test -v -run TestQwen3CoderCodeGeneration

echo ""
echo "📋 运行所有 DeepSeek 测试..."
go test -v -run TestDeepSeekAll

echo ""
echo "📋 运行所有 千问3 Coder 测试..."
go test -v -run TestQwen3CoderAll

echo ""
echo "⚡ 运行性能测试..."
go test -v -bench=BenchmarkDeepSeekChat -run=^$
go test -v -bench=BenchmarkQwen3CoderChat -run=^$

echo ""
echo "🎯 运行集成测试..."
go test -v -run TestDeepSeekIntegration
go test -v -run TestQwen3CoderIntegration

echo ""
echo "✅ 测试完成！"
echo "=================================================="
echo "💡 提示："
echo "   - 如果看到 'API返回错误'，请检查 API Key 是否有效"
echo "   - 如果看到网络错误，请检查网络连接"
echo "   - 如果测试通过，说明 API Key 工作正常"
echo "   - 千问3 Coder 需要单独的 API Key，请配置到测试文件中"
