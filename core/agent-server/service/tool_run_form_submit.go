package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
)

type RunFormSubmitTool struct{}

type runFormSubmitArgs struct {
	FullCodePath  string                 `json:"full_code_path" schema_desc:"表单函数完整路径" schema_required:"true"`
	Body          string                 `json:"body" schema_desc:"表单提交 JSON 字符串"`
	OutputDisplay map[string]interface{} `json:"output_display" schema_desc:"前端需要额外展示的结果字段映射"`
}

var runFormSubmitToolDef = toolDefinition[runFormSubmitArgs](
	"run_form_submit",
	"执行工作区内 Form 函数的提交接口，提交表单数据。执行前必须已通过 search 字段摘要或 read_go_file 确认该 Form 的 Request 字段名、必填项、枚举值、文件字段和默认值行为；不要根据函数名、路由名、相似工具或命令行习惯猜 body。full_code_path 必须为带 `.form` 后缀的具体表单函数完整路径，如 /luobei/myapp/plugins/cashier_desk.form。body 为 JSON 对象字符串，包含表单字段（如 {\"name\":\"张三\",\"amount\":100}）；字段摘要中标记为【必填】的字段必须显式传入，若表单无【必填】字段可传 {}。字段摘要中的“渲染默认值”仅表示前端界面初始值，不会自动写入 body；如需使用该值，也必须在 body 中显式传入。files 字段传 bucket/object_key 字符串，多文件用英文逗号分隔。output_display 可选，用于标记结果中需要在前端直接展示给用户的字段（避免大模型重复输出大段内容），key 为展示标签，value 为结果 JSON 中的字段名。返回中若有输出文件引用，文件已在工作台渲染为文件组件，不要编造 URL，也不要在最终回复里手写“下载文件：xxx”或 Markdown 下载链接。",
)

func (t *RunFormSubmitTool) Definition() dto.ToolDef {
	return runFormSubmitToolDef
}

func (t *RunFormSubmitTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runFormSubmitArgs](call.Args)
	if err != nil {
		return toolResult("run_form_submit 参数解析失败: "+err.Error(), true)
	}
	return runFormSubmitTool(ctx, args, call.FullCodePath)
}

// runFormSubmitTool 执行 Form 提交。body 由模型按具体表单定义自行拼装。
func runFormSubmitTool(ctx context.Context, args runFormSubmitArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".form")
	if fullCodePath == "" {
		return toolResult("run_form_submit 需传 full_code_path（表单函数路径，如 /luobei/myapp/plugins/cashier_desk.form）。", true)
	}
	bodyStr := strings.TrimSpace(args.Body)
	var body interface{}
	if bodyStr != "" {
		if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
			return toolResult("run_form_submit 的 body 需为合法 JSON 字符串: "+err.Error(), true)
		}
	} else {
		body = map[string]interface{}{}
	}
	bodyMap, ok := body.(map[string]interface{})
	if !ok {
		return toolResult("run_form_submit 的 body 必须是 JSON 对象字符串，不能是数组、数字或字符串。", true)
	}
	if msg := runWritePreflight(ctx, "run_form_submit", fullCodePath, functionschema.TypeForm, runWriteModeFormSubmit, []runWriteValidationPayload{{Body: bodyMap}}); msg != "" {
		return toolResult(msg, true)
	}
	result, err := apicall.FormSubmit(ctx, fullCodePath, body)
	if err != nil {
		logger.Errorf(ctx, "[RunFormSubmit] FormSubmit 失败: %v", err)
		return toolResult("run_form_submit 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredDataAndMetadata(result, false, metadataForDisplayFileFields(formResponseDisplayFileFields(ctx, fullCodePath)...), pathNotice)
}
