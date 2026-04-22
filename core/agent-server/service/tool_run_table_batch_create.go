package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type RunTableBatchCreateTool struct{}

type runTableBatchCreateArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"表格函数完整路径" schema_required:"true"`
	Body         string `json:"body" schema_desc:"JSON 对象字符串，格式为 {\"data\":[{...}]}" schema_required:"true"`
}

var runTableBatchCreateToolDef = toolDefinition[runTableBatchCreateArgs](
	"run_table_batch_create",
	"执行工作区内 Table 批量导入接口（触发 OnTableCreateInBatches）。执行前必须已通过 search_tools 字段摘要或 read_go_file 确认表格具备 batch-create 能力，并确认 model 的 json 字段名、必填项、枚举值和文件字段；不要猜 body。full_code_path 必须为带 `.table` 后缀的具体表格函数完整路径。body 必须为 JSON 对象字符串，格式为 {\"data\":[{\"title\":\"A\"},{\"title\":\"B\"}]}。普通逐条新增优先用 run_table_create；只有能力摘要明确支持 batch-create 时才用本工具。",
)

func (t *RunTableBatchCreateTool) Definition() dto.ToolDef {
	return runTableBatchCreateToolDef
}

func (t *RunTableBatchCreateTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runTableBatchCreateArgs](call.Args)
	if err != nil {
		return toolResult("run_table_batch_create 参数解析失败: "+err.Error(), true)
	}
	return runTableBatchCreateTool(ctx, args, call.FullCodePath)
}

func runTableBatchCreateTool(ctx context.Context, args runTableBatchCreateArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".table")
	if fullCodePath == "" {
		return toolResult("run_table_batch_create 需传 full_code_path（表格函数路径，如 /luobei/myapp/nps/nps_questionnaire_list.table）。", true)
	}
	bodyStr := strings.TrimSpace(args.Body)
	if bodyStr == "" {
		return toolResult("run_table_batch_create 需传 body（JSON 对象字符串，格式为 {\"data\":[...]}）。", true)
	}
	body, err := normalizeRunTableBatchCreateBody(bodyStr)
	if err != nil {
		return toolResult("run_table_batch_create 的 body 非法: "+err.Error(), true)
	}
	result, err := apicall.TableBatchCreate(ctx, fullCodePath, body)
	if err != nil {
		return toolResult("run_table_batch_create 调用失败: "+err.Error(), true)
	}
	out := map[string]interface{}{
		"imported_count": len(body.Data),
		"result":         result,
	}
	return toolResultWithStructuredData(out, false, pathNotice)
}

type runTableBatchCreateBody struct {
	Data []map[string]interface{} `json:"data"`
}

func normalizeRunTableBatchCreateBody(bodyStr string) (*runTableBatchCreateBody, error) {
	var body runTableBatchCreateBody
	if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
		return nil, err
	}
	if len(body.Data) == 0 {
		return nil, fmt.Errorf("data 必须是非空数组")
	}
	for i, row := range body.Data {
		if row == nil {
			return nil, fmt.Errorf("data[%d] 必须是对象", i)
		}
	}
	return &body, nil
}
