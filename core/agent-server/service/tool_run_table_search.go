package service

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
	"github.com/kageos/kageos/pkg/timex"
)

type RunTableSearchTool struct{}

type runTableSearchArgs struct {
	FullCodePath string `json:"full_code_path" schema_desc:"表格函数路径；同目录表格可用 ./xxx.table，相对当前 execute_directory 解析" schema_required:"true"`
	URLQuery     string `json:"url_query" schema_desc:"完整查询串"`
	Page         *int   `json:"page" schema_desc:"页码"`
	PageSize     *int   `json:"page_size" schema_desc:"每页条数"`
	Sorts        string `json:"sorts" schema_desc:"结构化排序 JSON，例如 [{\"field\":\"created_at\",\"order\":\"desc\"}]"`
}

var runTableSearchToolDef = toolDefinition[runTableSearchArgs](
	"run_table_search",
	"执行工作区内 Table 查询接口，返回分页表格数据。筛选前必须已通过字段摘要或 read_go_file 确认 Table Request 字段；不要猜可筛选字段或 url_query 格式。full_code_path 必须为带 `.table` 后缀的具体表格函数路径，包含函数名（如 .../nps/nps_questionnaire_list.table）；同目录表格可用 `./xxx.table`，跨目录使用完整路径。不能只填包路径（如 .../nps），否则会查不到数据。若只知包路径，请先用 read_dir 看该包下 .go 文件，根据 init() 中 GET(\"xxx_list.table\",...) 确定函数名，再直接使用环境信息里带后缀的 full_code_path。查询参数使用 Request 字段名和值，例如 status=处理中&title=合同&page=1&page_size=20；排序用结构化 sorts，例如 [{\"field\":\"created_at\",\"order\":\"desc\"}]。可传 url_query 或单独传 page、page_size、sorts。datetime 字段可直接传 `YYYY-MM-DD HH:mm:ss`，也可用 SQL 风格白名单表达式如 CURRENT_TIMESTAMP、CURRENT_DATE、DATE_SUB(CURRENT_TIMESTAMP, INTERVAL 7 DAY)。",
)

func (t *RunTableSearchTool) Definition() dto.ToolDef {
	return runTableSearchToolDef
}

func (t *RunTableSearchTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[runTableSearchArgs](call.Args)
	if err != nil {
		return toolResult("run_table_search 参数解析失败: "+err.Error(), true)
	}
	return runTableSearchTool(ctx, args, call.FullCodePath)
}

// runTableSearchTool 执行 Table 查询；可传 url_query 或 page/page_size/sorts。
func runTableSearchTool(ctx context.Context, args runTableSearchArgs, currentFullCodePath string) ToolResult {
	ctx = withAgentToolClientSource(ctx)
	fullCodePath, pathNotice := resolveTypedFunctionFullCodePathArg(args.FullCodePath, currentFullCodePath, ".table")
	if fullCodePath == "" {
		return toolResult("run_table_search 需传 full_code_path（表格函数路径，如 /luobei/myapp/tables/hr_resume_list.table）。", true)
	}
	var params url.Values
	if q := strings.TrimSpace(args.URLQuery); q != "" {
		parsed, err := url.ParseQuery(q)
		if err != nil {
			return toolResult("run_table_search 的 url_query 需为合法查询串: "+err.Error(), true)
		}
		params = parsed
		if params.Get("page") == "" {
			params.Set("page", "1")
		}
		if params.Get("page_size") == "" {
			params.Set("page_size", "20")
		}
	} else {
		params = url.Values{}
		params.Set("page", "1")
		params.Set("page_size", "20")
		if args.Page != nil {
			params.Set("page", strconv.Itoa(*args.Page))
		}
		if args.PageSize != nil {
			params.Set("page_size", strconv.Itoa(*args.PageSize))
		}
		if s := strings.TrimSpace(args.Sorts); s != "" {
			params.Set("sorts", s)
		}
	}
	for key := range params {
		params.Set(key, timex.ReplaceTimeExprsInParamValue(params.Get(key)))
	}
	result, err := apicall.TableSearch(ctx, fullCodePath, params)
	if err != nil {
		logger.Errorf(ctx, "[RunTableSearch] TableSearch 失败: %v", err)
		return toolResult("run_table_search 调用失败: "+err.Error(), true)
	}
	return toolResultWithStructuredData(result, false, pathNotice)
}
