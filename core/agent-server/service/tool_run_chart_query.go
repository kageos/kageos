package service

import (
	"context"
	"net/url"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

type RunChartQueryTool struct{}

type runChartQueryArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"图表函数完整路径" schema_required:"true"`
	URLQuery     string `json:"url_query" schema_desc:"完整查询串"`
}

var runChartQueryToolDef = toolDefinition[runChartQueryArgs](
	"run_chart_query",
	"执行工作区内 Chart 查询接口，返回图表数据。full_code_path 必须为带 `.chart` 后缀的具体图表函数完整路径，如 /luobei/myapp/charts/sales_trend.chart。图表查询参数不固定，由具体 Chart 的 handler 定义（如 year、month、dimension 等），请用 read_go_file 查看对应 .go 的 Req 结构。传 url_query 为完整查询串（如 year=2024&month=1），不传则无额外参数。",
)

func (t *RunChartQueryTool) Definition() dto.ToolDef {
	return runChartQueryToolDef
}

func (t *RunChartQueryTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runChartQueryArgs](call.Args)
	if err != nil {
		return toolResult("run_chart_query 参数解析失败: "+err.Error(), true)
	}
	content, isError := runChartQueryTool(ctx, args, call.FullCodePath)
	return toolResult(content, isError)
}

// runChartQueryTool 执行 Chart 查询；图表参数不固定，由 handler 定义，可传 url_query
func runChartQueryTool(ctx context.Context, args runChartQueryArgs, currentFullCodePath string) (string, bool) {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".chart")
	if fullCodePath == "" {
		return "run_chart_query 需传 full_code_path（图表函数路径，如 /luobei/myapp/charts/sales_trend.chart）。", true
	}
	var params url.Values
	if q := strings.TrimSpace(args.URLQuery); q != "" {
		parsed, err := url.ParseQuery(q)
		if err != nil {
			return "run_chart_query 的 url_query 需为合法查询串: " + err.Error(), true
		}
		params = parsed
	} else {
		params = url.Values{}
	}
	result, err := apicall.ChartQuery(ctx, fullCodePath, params)
	if err != nil {
		logger.Errorf(ctx, "[RunChartQuery] ChartQuery 失败: %v", err)
		return "run_chart_query 调用失败: " + err.Error(), true
	}
	content, _ := formatJSONResult(result)
	if pathNotice != "" {
		return pathNotice + "\n\n" + content, false
	}
	return content, false
}
