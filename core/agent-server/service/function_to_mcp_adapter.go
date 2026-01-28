package service

import (
	"fmt"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

// FunctionToMCPToolDef 将 function 表的 Request（请求参数）、Response（响应参数）转为 MCP ToolDef。
//
// 入参：
//   - name, description: 工具名与描述（可来自 function 或 service_tree 元数据）
//   - requestJSON: function.Request 的原始 JSON，即 []*widget.Field
//   - responseJSON: function.Response 的原始 JSON，即 []*widget.Field
//
// 转换规则：
//   - InputSchema  ← 解析 requestJSON 为 []*widget.Field，按 code/name/desc/data.type 等组装为 JSON Schema
//   - OutputSchema ← 解析 responseJSON 为 []*widget.Field，同理生成 JSON Schema
//
// 数据来源：function 表在 app-server，需通过 API（如按 full_code_path 获取函数元数据）取得 Request、Response 后再传入。
// 后续实现：可依赖 sdk/agent-app/widget，解析 JSON 并遍历 Field 构建 properties/required。
func FunctionToMCPToolDef(name, description string, requestJSON, responseJSON []byte) (*dto.ToolDef, error) {
	// 占位：后续实现 widget.Field → JSON Schema 的转换
	_ = requestJSON
	_ = responseJSON
	return nil, fmt.Errorf("FunctionToMCPToolDef: 适配层尚未实现，请从 function 表或 API 获取 Request/Response 后在此完成 widget.Field → InputSchema/OutputSchema 的转换")
}
