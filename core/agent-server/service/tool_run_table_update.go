package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/functionschema"
	"github.com/kageos/kageos/pkg/logger"
)

type RunTableUpdateTool struct{}

type runTableUpdateArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"表格函数路径；同目录表格可用 ./xxx.table 或 <./xxx.table>，相对当前 execute_directory 解析" schema_required:"true"`
	Body         string `json:"body" schema_desc:"JSON 数组字符串，每项含 id 和 updates" schema_required:"true"`
}

var runTableUpdateToolDef = toolDefinition[runTableUpdateArgs](
	"run_table_update",
	"执行工作区内 Table 更新接口，批量更新表格记录（每条都会触发 OnTableUpdateRow）。执行前必须已通过 search 字段摘要或 read_go_file 确认表格具备编辑能力，并确认 model 的 json 字段名、可更新字段、枚举值和文件字段；不要猜 updates。full_code_path 必须为带 `.table` 后缀的具体表格函数路径；同目录表格可用 `./xxx.table` 或 `<./xxx.table>`。body 必须为 JSON 数组字符串，每项为 { \"id\": 行ID, \"updates\": { \"字段名\": 新值, ... } }；不传 old_values，由 app-server 自动查表填充。返回 updated_count、data_list、failed_count、errors。",
)

func (t *RunTableUpdateTool) Definition() dto.ToolDef {
	return runTableUpdateToolDef
}

func (t *RunTableUpdateTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runTableUpdateArgs](call.Args)
	if err != nil {
		return toolResult("run_table_update 参数解析失败: "+err.Error(), true)
	}
	return runTableUpdateTool(ctx, args, call.FullCodePath)
}

// runTableUpdateTool 执行 Table 批量更新；body 为 JSON 数组，每项 { id, updates }
func runTableUpdateTool(ctx context.Context, args runTableUpdateArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".table")
	if fullCodePath == "" {
		return toolResult("run_table_update 需传 full_code_path（表格函数路径，如 /luobei/myapp/nps/nps_questionnaire_list.table）。", true)
	}
	bodyStr := strings.TrimSpace(args.Body)
	if bodyStr == "" {
		return toolResult("run_table_update 需传 body（JSON 数组字符串，每项为 { \"id\": 行ID, \"updates\": { \"字段\": 新值 } }）。", true)
	}
	var bodyArr []interface{}
	if err := json.Unmarshal([]byte(bodyStr), &bodyArr); err != nil {
		return toolResult("run_table_update 的 body 需为合法 JSON 数组: "+err.Error(), true)
	}
	if len(bodyArr) == 0 {
		return toolResult("run_table_update 的 body 不能为空数组。", true)
	}
	payloads := make([]runWriteValidationPayload, 0, len(bodyArr))
	for i, row := range bodyArr {
		rowMap, ok := row.(map[string]interface{})
		if !ok || rowMap == nil {
			return toolResult("run_table_update 的 body 写入前校验失败，本次未提交任何数据：每项必须为 JSON 对象，且含 id 与 updates。", true)
		}
		if _, hasID := rowMap["id"]; !hasID {
			return toolResult("run_table_update 的 body 写入前校验失败，本次未提交任何数据：存在记录缺少 id。", true)
		}
		updates, ok := rowMap["updates"].(map[string]interface{})
		if !ok || updates == nil {
			return toolResult("run_table_update 的 body 写入前校验失败，本次未提交任何数据：存在记录缺少 updates 或 updates 非对象。", true)
		}
		payloads = append(payloads, runWriteValidationPayload{
			Label: fmt.Sprintf("[%d].updates", i),
			Body:  updates,
		})
	}
	if msg := runWritePreflight(ctx, "run_table_update", fullCodePath, functionschema.TypeTable, runWriteModeTableUpdate, payloads); msg != "" {
		return toolResult(msg, true)
	}

	dataList := make([]interface{}, 0, len(bodyArr))
	var errorsList []map[string]interface{}
	updatedCount := 0
	failedCount := 0

	for i, row := range bodyArr {
		rowMap, ok := row.(map[string]interface{})
		if !ok || rowMap == nil {
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{"index": i, "error": "每项必须为 JSON 对象，且含 id 与 updates"})
			continue
		}
		if _, hasID := rowMap["id"]; !hasID {
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{"index": i, "error": "缺少 id"})
			continue
		}
		updates, ok := rowMap["updates"].(map[string]interface{})
		if !ok || updates == nil {
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{"index": i, "error": "缺少 updates 或 updates 非对象"})
			continue
		}
		result, err := apicall.TableUpdate(ctx, fullCodePath, rowMap)
		if err != nil {
			logger.Errorf(ctx, "[RunTableUpdate] 第 %d 条 TableUpdate 失败: %v", i+1, err)
			failedCount++
			errorsList = append(errorsList, map[string]interface{}{"index": i, "error": err.Error()})
			continue
		}
		_ = updates
		updatedCount++
		dataList = append(dataList, result)
	}

	out := map[string]interface{}{
		"updated_count": updatedCount,
		"failed_count":  failedCount,
		"data_list":     dataList,
	}
	if len(errorsList) > 0 {
		out["errors"] = errorsList
	}
	return toolResultWithStructuredData(out, false, pathNotice)
}
