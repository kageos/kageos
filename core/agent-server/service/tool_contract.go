package service

import (
	"context"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

type Tool interface {
	Definition() dto.ToolDef
	Execute(ctx context.Context, call ToolCall) ToolResult
}

type ToolCall struct {
	Args         map[string]interface{}
	FullCodePath string
	Files        string
}

type ToolResult struct {
	Content string `json:"content" schema_desc:"工具执行结果内容" schema_required:"true"`
	IsError bool   `json:"is_error" schema_desc:"是否为错误结果" schema_required:"true"`
	Data    any    `json:"data,omitempty" schema_ignore:"true"`
}

type structuredToolResultSchema[T any] struct {
	Content string `json:"content" schema_desc:"工具执行结果内容" schema_required:"true"`
	IsError bool   `json:"is_error" schema_desc:"是否为错误结果" schema_required:"true"`
	Data    *T     `json:"data,omitempty" schema_desc:"工具结构化结果"`
}

func toolResult(content string, isError bool) ToolResult {
	return ToolResult{Content: content, IsError: isError}
}

func toolResultWithData(content string, isError bool, data any) ToolResult {
	return ToolResult{Content: content, IsError: isError, Data: data}
}
