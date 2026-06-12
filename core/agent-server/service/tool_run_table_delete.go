package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
)

type RunTableDeleteTool struct{}

type runTableDeleteArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"表格函数完整路径" schema_required:"true"`
	Body         string `json:"body" schema_desc:"JSON 数组字符串，每项为要删除的行 ID" schema_required:"true"`
}

var runTableDeleteToolDef = toolDefinition[runTableDeleteArgs](
	"run_table_delete",
	"执行工作区内 Table 删除接口，批量删除表格记录（触发 OnTableDeleteRows）。执行前必须已通过 search 字段摘要或 read_go_file 确认表格具备删除能力；不要猜 id。full_code_path 必须为带 `.table` 后缀的具体表格函数完整路径。body 必须为 JSON 数组字符串，每项为要删除的行 ID，如 [1,2,3]。返回 deleted_count、ids、result。",
)

func (t *RunTableDeleteTool) Definition() dto.ToolDef {
	return runTableDeleteToolDef
}

func (t *RunTableDeleteTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runTableDeleteArgs](call.Args)
	if err != nil {
		return toolResult("run_table_delete 参数解析失败: "+err.Error(), true)
	}
	return runTableDeleteTool(ctx, args, call.FullCodePath)
}

func runTableDeleteTool(ctx context.Context, args runTableDeleteArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".table")
	if fullCodePath == "" {
		return toolResult("run_table_delete 需传 full_code_path（表格函数路径，如 /luobei/myapp/nps/nps_questionnaire_list.table）。", true)
	}
	bodyStr := strings.TrimSpace(args.Body)
	if bodyStr == "" {
		return toolResult("run_table_delete 需传 body（JSON 数组字符串，每项为要删除的行 ID）。", true)
	}
	var rawIDs []interface{}
	if err := json.Unmarshal([]byte(bodyStr), &rawIDs); err != nil {
		return toolResult("run_table_delete 的 body 需为合法 JSON 数组: "+err.Error(), true)
	}
	ids, err := normalizeRunTableDeleteIDs(rawIDs)
	if err != nil {
		return toolResult("run_table_delete 的 body 非法: "+err.Error(), true)
	}

	result, err := apicall.TableDelete(ctx, fullCodePath, map[string]interface{}{"ids": ids})
	if err != nil {
		return toolResult("run_table_delete 调用失败: "+err.Error(), true)
	}
	out := map[string]interface{}{
		"deleted_count": len(ids),
		"ids":           ids,
		"result":        result,
	}
	return toolResultWithStructuredData(out, false, pathNotice)
}

func normalizeRunTableDeleteIDs(rawIDs []interface{}) ([]int64, error) {
	if len(rawIDs) == 0 {
		return nil, fmt.Errorf("不能为空数组")
	}
	ids := make([]int64, 0, len(rawIDs))
	for i, rawID := range rawIDs {
		id, err := normalizeRunTableDeleteID(rawID)
		if err != nil {
			return nil, fmt.Errorf("第 %d 项非法: %w", i, err)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func normalizeRunTableDeleteID(rawID interface{}) (int64, error) {
	var id int64
	switch v := rawID.(type) {
	case float64:
		id = int64(v)
		if v != float64(id) {
			return 0, fmt.Errorf("id 必须是整数")
		}
	case int:
		id = int64(v)
	case int64:
		id = v
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		if err != nil {
			return 0, fmt.Errorf("id 字符串不是合法整数")
		}
		id = parsed
	default:
		return 0, fmt.Errorf("id 类型不支持: %T", rawID)
	}
	if id <= 0 {
		return 0, fmt.Errorf("id 必须大于 0")
	}
	return id, nil
}
