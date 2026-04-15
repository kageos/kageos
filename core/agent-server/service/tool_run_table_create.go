package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type RunTableCreateTool struct{}

type runTableCreateArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"表格函数完整路径" schema_required:"true"`
	Body         string `json:"body" schema_desc:"JSON 数组字符串，每项一条记录" schema_required:"true"`
}

var runTableCreateToolDef = toolDefinition[runTableCreateArgs](
	"run_table_create",
	"执行工作区内 Table 新增接口，批量新增表格记录（每条都会触发 OnTableAddRow）。仅适用于开启新增能力的 Table；调用前应先看 search_tools/函数能力摘要，确认不是只读表。full_code_path 必须为带 `.table` 后缀的具体表格函数完整路径（如 /luobei/myapp/nps/nps_questionnaire_list.table）。body 必须为 JSON 数组字符串，每项为一条记录的字段对象，如 [{\"title\":\"问卷A\"},{\"title\":\"问卷B\"}]；字段名与表格 model 的 json 标签一致，必填项需包含。返回 data_list 为成功插入的每条记录（后端返回的数据列表），以及 created_count、failed_count、errors。",
)

func (t *RunTableCreateTool) Definition() dto.ToolDef {
	return runTableCreateToolDef
}

func (t *RunTableCreateTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runTableCreateArgs](call.Args)
	if err != nil {
		return toolResult("run_table_create 参数解析失败: "+err.Error(), true)
	}
	content, isError := runTableCreateTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runTableCreateTool 执行 Table 新增；body 必须为 JSON 数组，逐条调用 table/create 触发 OnTableAddRow
func runTableCreateTool(ctx context.Context, args runTableCreateArgs, currentFullCodePath string) (string, bool) {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".table")
	if fullCodePath == "" {
		return "run_table_create 需传 full_code_path（表格函数路径，如 /luobei/myapp/nps/nps_questionnaire_list.table）。", true
	}
	bodyStr := strings.TrimSpace(args.Body)
	if bodyStr == "" {
		return "run_table_create 需传 body（JSON 数组字符串，每项为一条记录）。", true
	}
	var bodyArr []interface{}
	if err := json.Unmarshal([]byte(bodyStr), &bodyArr); err != nil {
		return "run_table_create 的 body 需为合法 JSON 数组: " + err.Error(), true
	}
	if len(bodyArr) == 0 {
		return "run_table_create 的 body 不能为空数组。", true
	}

	dataList := make([]interface{}, 0, len(bodyArr))
	var errorsList []map[string]interface{}
	createdCount := 0
	failedCount := 0

	for i, row := range bodyArr {
		if row == nil {
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{"index": i, "error": "元素不能为 null"})
			continue
		}
		if _, ok := row.(map[string]interface{}); !ok {
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{"index": i, "error": "每条必须为 JSON 对象，不能为数组、数字或字符串"})
			continue
		}
		result, err := apicall.TableCreate(ctx, fullCodePath, row)
		if err != nil {
			logger.Errorf(ctx, "[RunTableCreate] 第 %d 条 TableCreate 失败: %v", i+1, err)
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{
				"index": i,
				"error": err.Error(),
			})
			continue
		}
		createdCount++
		dataList = append(dataList, extractTableCreateRecord(result))
	}

	out := map[string]interface{}{
		"created_count": createdCount,
		"failed_count":  failedCount,
		"data_list":     dataList,
	}
	if len(errorsList) > 0 {
		out["errors"] = errorsList
	}
	content, _ := formatJSONResult(out)
	if pathNotice != "" {
		return pathNotice + "\n\n" + content, false
	}
	return content, false
}

// extractTableCreateRecord 从 table/create 的返回值中提取单条记录。
func extractTableCreateRecord(result map[string]interface{}) interface{} {
	if result == nil {
		return result
	}
	if v, ok := result["data"]; ok && v != nil {
		return v
	}
	return result
}
