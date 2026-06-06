package service

import (
	"context"
	"fmt"
	"slices"

	"github.com/kageos/kageos/dto"
)

// ToolRegistry 工作台工具注册与调用（仅内置工具，已移除插件）
type ToolRegistry struct {
	tools     map[string]Tool
	toolOrder []string
}

// NewToolRegistry 创建 ToolRegistry。
func NewToolRegistry() *ToolRegistry {
	r := &ToolRegistry{
		tools: make(map[string]Tool, 32),
	}
	r.registerBuiltinTools()
	return r
}

func (r *ToolRegistry) registerBuiltinTools() {
	tools := make([]Tool, 0, 32)
	tools = append(tools, workspaceTools(r)...)
	tools = append(tools, runtimeTools(r)...)
	tools = append(tools, platformTools(r)...)
	for _, tool := range tools {
		r.registerTool(tool)
	}
}

func (r *ToolRegistry) registerTool(tool Tool) {
	if tool == nil {
		panic("cannot register nil tool")
	}
	definition := tool.Definition()
	name := definition.Name
	if name == "" {
		panic("cannot register tool with empty name")
	}
	if _, exists := r.tools[name]; exists {
		panic(fmt.Sprintf("duplicate tool registration: %s", name))
	}
	r.tools[name] = tool
	r.toolOrder = append(r.toolOrder, name)
}

func (r *ToolRegistry) AllToolNames() []string {
	return slices.Clone(r.toolOrder)
}

// ListTools 返回可用工具定义（仅内置）。toolNames 非空时只返回 name 在列表中的工具，空则返回全部。
func (r *ToolRegistry) ListTools(ctx context.Context, toolNames []string) ([]dto.ToolDef, error) {
	_ = ctx

	out := make([]dto.ToolDef, 0, len(r.toolOrder))
	nameSet := make(map[string]struct{}, len(toolNames))
	if len(toolNames) > 0 {
		for _, name := range toolNames {
			nameSet[name] = struct{}{}
		}
	}

	for _, name := range r.toolOrder {
		if len(nameSet) > 0 {
			if _, ok := nameSet[name]; !ok {
				continue
			}
		}
		tool, ok := r.tools[name]
		if !ok {
			continue
		}
		out = append(out, tool.Definition())
	}
	return out, nil
}

// CallTool 执行工具；full_code_path 从会话上下文传入；files 为当前用户消息附件 refs。
func (r *ToolRegistry) CallTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files string) ToolResult {
	tool, ok := r.tools[name]
	if !ok {
		return toolResult("tool not found: "+name, true)
	}
	definition := tool.Definition()
	if err := validateToolArguments(definition.InputSchema, args); err != nil {
		return toolResult("tool 参数校验失败: "+err.Error(), true)
	}
	return tool.Execute(ctx, ToolCall{
		Args:         args,
		FullCodePath: fullCodePath,
		Files:        files,
	})
}
