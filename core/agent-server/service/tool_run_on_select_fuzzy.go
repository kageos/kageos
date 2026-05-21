package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

type RunOnSelectFuzzyTool struct{}

type runOnSelectFuzzyArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"配置了回调的 Form 或 Table 路径" schema_required:"true"`
	Code         string `json:"code" schema_desc:"字段 code" schema_required:"true"`
	Keyword      string `json:"keyword" schema_desc:"搜索关键词"`
	Request      string `json:"request" schema_desc:"当前表单或行数据 JSON 字符串"`
}

var runOnSelectFuzzyToolDef = toolDefinition[runOnSelectFuzzyArgs](
	"run_on_select_fuzzy",
	"执行工作区内 OnSelectFuzzy 回调，用于测试带「下拉模糊搜索/回调查询」的 Form 或 Table。执行前先通过字段摘要或 read_go_file 确认该字段配置了 OnSelectFuzzy，以及正确的字段 code 和依赖字段。**仅支持按关键词搜索**：type 固定为 by_keyword，value 为关键词字符串（可为空表示空搜索）。不支持 by_value、by_values。full_code_path 为配置了该回调的 Form 或 Table 的完整路径（如 .../cashier_desk.form）；code 为字段 code（如 product_id、member_id）；request 可选，为当前表单的 JSON（用于依赖其他字段时）。返回 items（选项列表）及可选 statistics、error_msg。",
)

func (t *RunOnSelectFuzzyTool) Definition() dto.ToolDef {
	return runOnSelectFuzzyToolDef
}

func (t *RunOnSelectFuzzyTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runOnSelectFuzzyArgs](call.Args)
	if err != nil {
		return toolResult("run_on_select_fuzzy 参数解析失败: "+err.Error(), true)
	}
	return runOnSelectFuzzyTool(ctx, args, call.FullCodePath)
}

// runOnSelectFuzzyTool 执行 OnSelectFuzzy 回调；仅支持按关键词或空关键词
func runOnSelectFuzzyTool(ctx context.Context, args runOnSelectFuzzyArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath := resolveFullCodePathArg(args.FullCodePath, currentFullCodePath)
	if fullCodePath == "" {
		return toolResult("run_on_select_fuzzy 需传 full_code_path（配置了 OnSelectFuzzy 的 Form/Table 路径，如 .../cashier_desk.form 或 .../member_list.table）。", true)
	}
	code := strings.TrimSpace(args.Code)
	if code == "" {
		return toolResult("run_on_select_fuzzy 需传 code（字段 code，与 OnSelectFuzzyMap 的键一致）。", true)
	}

	keyword := strings.TrimSpace(args.Keyword)
	body := map[string]interface{}{
		"code":  code,
		"type":  "by_keyword",
		"value": keyword,
	}
	if s := strings.TrimSpace(args.Request); s != "" {
		var reqObj interface{}
		if err := json.Unmarshal([]byte(s), &reqObj); err != nil {
			body["request"] = map[string]interface{}{}
		} else {
			body["request"] = reqObj
		}
	} else {
		body["request"] = map[string]interface{}{}
	}

	result, err := apicall.CallbackOnSelectFuzzy(ctx, fullCodePath, body)
	if err != nil {
		logger.Errorf(ctx, "[RunOnSelectFuzzy] CallbackOnSelectFuzzy 失败: %v", err)
		return toolResult("run_on_select_fuzzy 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(result, false)
}
