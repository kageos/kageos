package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/timex"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// ToolRegistry 工作台工具注册与调用（仅内置工具，已移除插件）
// list_tools：仅内置；call_tool(name, args, full_code_path) 路由到对应实现
type ToolRegistry struct{}

// NewToolRegistry 创建 ToolRegistry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{}
}

// ListTools 返回可用工具定义（仅内置）。toolNames 非空时只返回 name 在列表中的工具，空则返回全部。
func (r *ToolRegistry) ListTools(ctx context.Context, toolNames []string) ([]dto.ToolDef, error) {
	out := make([]dto.ToolDef, 0, 32)

	// 1. 读代码文件：read_go_file（仅工作区 Go 代码）
	out = append(out, dto.ToolDef{
		Name:        "read_go_file",
		Description: "读取工作区内指定目录下的 Go 代码文件内容。参数：directory（可选，不传则当前工作目录）、file_name（可选，单文件如 a.go，或多文件逗号分隔如 a.go,b.go；不传则返回该目录下所有代码文件）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名（可选），单文件如 a.go，或多文件逗号分隔如 a.go,b.go；不传则返回该目录下所有代码文件",
				},
			},
			"required": []interface{}{},
		},
	})

	// 1.2 按行号读取代码文件：read_go_file_lines（带行号，便于对照编译错误）
	out = append(out, dto.ToolDef{
		Name:        "read_go_file_lines",
		Description: "按指定行号范围读取工作区内的 Go 代码文件，输出带行号，便于对照编译错误信息。参数：directory（可选）、file_name（必填）、line_ranges（可选，如 \"10-12,20-30\" 表示第 10-12 行和第 20-30 行；不传则返回整个文件并带行号）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 task.go 或 task",
				},
				"line_ranges": map[string]interface{}{
					"type":        "string",
					"description": "行号范围（可选），如 10-12,20-30 表示只返回第 10-12 行和第 20-30 行；不传则返回整个文件（带行号）",
				},
			},
			"required": []interface{}{"file_name"},
		},
	})

	// 2. 读文档：read_doc（directory 唯一定位，内置或工作区；支持逗号分隔多路径）
	out = append(out, dto.ToolDef{
		Name:        "read_doc",
		Description: "读取文档内容。传 directory 定位文档（单路径如 /builtin/doc/sdk/agent-app-sdk-readme，或多路径逗号分隔如 /builtin/doc/a,/builtin/doc/b）。系统消息中会列出可读文档的 directory 及名称。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "文档路径，单路径或逗号分隔多路径，如 /builtin/doc/sdk/agent-app-sdk-readme 或 /builtin/doc/a,/builtin/doc/b",
				},
			},
			"required": []interface{}{"directory"},
		},
	})

	// 1.1. 读取目录工具：read_dir
	out = append(out, dto.ToolDef{
		Name:        "read_dir",
		Description: "读取指定目录下的所有子目录和文件，以树形方式展开。默认返回当前目录及其下一层的目录、函数、代码文件（tree 格式）。recursive=true 时递归显示整棵目录树；include_files 默认 true 会列出 .go 等代码文件。不传 directory 则使用当前工作目录。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选，不传则当前工作目录），如 /luobei/myapp/task_management",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "是否递归显示子目录（可选，默认 false）。true 时递归显示所有子目录的树形结构，类似 tree 命令。",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "最大深度（可选，仅在 recursive=true 时有效，默认不限制），如 3 表示最多显示 3 层",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "输出格式（可选，默认 tree）。可选值：tree（树形格式，推荐）、list（列表格式，适合详细查看）",
				},
				"include_functions": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含函数节点（可选，默认 true），展示函数可以快速了解功能",
				},
				"include_files": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码文件（可选，默认 true），会列出目录下的 .go 等文件；设为 false 则只显示目录和函数节点",
				},
				"include_code": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码内容（可选，默认 false）。true 时在列表中带出文件内容，消耗更多 token。",
				},
			},
			"required": []interface{}{},
		},
	})

	out = append(out, dto.ToolDef{
		Name:        "create_directory",
		Description: "在当前目录或指定 directory（父目录）下创建一个子目录（package 类型）。必填：name（显示名称）、code（代码标识）。可选：directory（父目录）、description、tags、admins。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "父目录（可选），不传则使用当前目录",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "目录显示名称，如「文档」",
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "目录代码标识，如 docs",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "目录描述（可选），如「存放项目文档」",
				},
				"tags": map[string]interface{}{
					"type":        "string",
					"description": "标签，逗号分隔（可选），如 api,user,management",
				},
				"admins": map[string]interface{}{
					"type":        "string",
					"description": "管理员列表，逗号分隔（可选）；不填则默认为当前用户；需要为他人加管理员时可填写，如 user1,user2",
				},
			},
			"required": []interface{}{"name", "code"},
		},
	})

	// write_doc：写文档（目录 + name + code + content）
	out = append(out, dto.ToolDef{
		Name:        "write_doc",
		Description: "在指定目录下创建或更新一篇文档。必填：name（显示名称）、code（英文标识）、content（正文）。可选：directory（父目录，不传则当前工作目录）、format（默认 markdown）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "父目录（可选），不传则当前工作目录",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "文档显示名称，如「项目说明」",
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "文档英文标识，用于路径，如 readme、api_docs",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文档正文（Markdown 等）",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "文档格式（可选），默认 markdown",
				},
			},
			"required": []interface{}{"name", "code", "content"},
		},
	})

	// write_go_file：写 Go 代码文件（file_name + content；build_workspace 预留）
	out = append(out, dto.ToolDef{
		Name:        "write_go_file",
		Description: "在当前工作目录或指定 directory 下写入一个 .go 代码文件。必填：file_name（如 attendance.go）、content（Go 源码）。可选：directory（目标目录）、build_workspace（是否立即编译，默认 true）。使用原则：若本次任务只需新增一个文件即可完成，直接写并编译即可（不传或传 true，省事）；若本次任务需要新增多个文件，则每个 write_go_file 传 build_workspace=false 仅写不编译，全部写完后调用一次 build_workspace 再编译。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 attendance.go、biz_vote_system.go",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Go 源码全文",
				},
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目标目录（可选），不传则当前工作目录",
				},
				"build_workspace": map[string]interface{}{
					"type":        "boolean",
					"description": "是否立即编译（可选，默认 true）。单文件任务用默认即可；多文件任务时传 false，全部写完后调用 build_workspace 再编译。",
				},
			},
			"required": []interface{}{"file_name", "content"},
		},
	})

	// build_workspace：编译当前工作空间（不写文件，仅触发编译并部署）；无需参数
	out = append(out, dto.ToolDef{
		Name:        "build_workspace",
		Description: "编译当前工作空间（Go 应用）。不写文件，仅基于当前已落盘的代码触发一次编译并部署。无需传参。连续写多个文件后可调用一次 build_workspace 再编译。",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []interface{}{},
		},
	})

	// search_replace_file：文件内容 search-replace（统一批量：多组替换同一文件，全部生效才落盘）
	out = append(out, dto.ToolDef{
		Name:        "search_replace_file",
		Description: "在指定目录下的 .go 文件中做「查找并替换」：只改匹配到的片段，不重写整文件。必填：directory（或当前目录）、file_name、replacements（替换列表，每项含 search_string、replace_string、expected_count 可选默认 1）。all_or_nothing 默认 true：仅当所有项的实际匹配次数等于 expected_count 时才落盘，否则不写入。search_string 必须与文件内容完全一致（含空格、制表符、换行）；使用前建议先用 read_go_file 读取后从实际内容复制。示例：replacements: [{ \"search_string\": \"原文\", \"replace_string\": \"新文\", \"expected_count\": 1 }]。仅修改代码、不编译工作空间；若需生效改完后需调用 build_workspace。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录，如 /user/app/pkg1",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 handler 或 handler.go",
				},
				"replacements": map[string]interface{}{
					"type":        "array",
					"description": "替换列表，按顺序执行；每项含 search_string（必填）、replace_string、expected_count（可选，默认 1，表示该项预期匹配次数；若实际次数不符且 all_or_nothing 则不落盘）",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"search_string":  map[string]interface{}{"type": "string", "description": "要被替换的原文（必须与文件内容完全一致）"},
							"replace_string": map[string]interface{}{"type": "string", "description": "替换后的内容"},
							"expected_count": map[string]interface{}{"type": "integer", "description": "该项预期匹配次数，不传或 0 表示 1"},
						},
						"required": []interface{}{"search_string"},
					},
				},
				"all_or_nothing": map[string]interface{}{
					"type":        "boolean",
					"description": "为 true 时仅当所有项 actual_count==expected_count 才落盘，默认 true",
				},
				"return_full_content": map[string]interface{}{
					"type":        "boolean",
					"description": "是否在结果中返回替换后的完整文件内容（可选，默认 true）",
				},
			},
			"required": []interface{}{"file_name", "replacements"},
		},
	})

	// delete_file：删除目录下指定 .go 文件（删磁盘+删节点）
	out = append(out, dto.ToolDef{
		Name:        "delete_file",
		Description: "删除指定目录下的一个 .go 代码文件。必填：directory（或当前目录）、file_name。会同时删除磁盘文件和 DB 节点。不能删除 init_.go。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 handler 或 handler.go",
				},
			},
			"required": []interface{}{"file_name"},
		},
	})

	// 执行模式专用：查表 / 提交表单 / 查图表（调用工作区标准接口）
	// Table 搜索参数遵循 pkg/gormx/query 约定，可搜字段由表格 model 的 search 标签决定（见 readme）
	out = append(out, dto.ToolDef{
		Name:        "run_table_search",
		Description: "执行工作区内 Table 查询接口，返回分页表格数据。full_code_path 必须为「具体表格函数的完整路径」，包含函数名（如 .../nps/nps_questionnaire_list），不能只填包路径（如 .../nps），否则会查不到数据。若只知包路径，请先用 read_dir 看该包下 .go 文件，根据 init() 中 GET(\"xxx_list\",...) 确定函数名，再拼成 full_code_path=.../包名/函数名。查询参数遵循 pkg/gormx/query：page、page_size、sorts、eq/like/in/gte/lte 等；可传 url_query 或单独传 page、page_size、sorts。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "表格函数的完整路径，必须包含函数名，如 /luobei/myapp/nps/nps_questionnaire_list；不能只填包路径如 .../nps，否则返回空。",
				},
				"url_query": map[string]interface{}{
					"type":        "string",
					"description": "完整 URL 查询串（可选），与 pkg/gormx/query 一致。时间范围可用时间函数，工具内部会转为时间戳：Now() 当前时间、Today() 今天 0 点、Yesterday() 昨天 0 点、Now(-7d) 七天前、Now(2026-02-01 13:05:05) 指定时间。例：page=1&page_size=20&gte=created_at:Now(-7d)&lte=created_at:Now()。不传则用默认分页。",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码（可选，默认 1）；若已传 url_query 则优先用 url_query 内参数",
				},
				"page_size": map[string]interface{}{
					"type":        "integer",
					"description": "每页条数（可选，默认 20）",
				},
				"sorts": map[string]interface{}{
					"type":        "string",
					"description": "排序（可选），如 id:desc,name:asc 或 -updated_at",
				},
			},
			"required": []interface{}{"full_code_path"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "run_form_submit",
		Description: "执行工作区内 Form 函数的提交接口，提交表单数据。full_code_path 为表单函数的完整路径，如 /luobei/myapp/plugins/cashier_desk。body 为 JSON 对象字符串，包含表单字段（如 {\"name\":\"张三\",\"amount\":100}）；若表单无必填字段可传 {}。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "表单函数的完整路径，如 /luobei/myapp/plugins/cashier_desk",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "表单字段的 JSON 字符串（可选），如 {\"name\":\"张三\",\"amount\":100}；无字段时传 {}",
				},
			},
			"required": []interface{}{"full_code_path"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "run_chart_query",
		Description: "执行工作区内 Chart 查询接口，返回图表数据。full_code_path 为图表函数路径，如 /luobei/myapp/charts/sales。图表查询参数不固定，由具体 Chart 的 handler 定义（如 year、month、dimension 等），请用 read_go_file 查看对应 .go 的 Req 结构。传 url_query 为完整查询串（如 year=2024&month=1），不传则无额外参数。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "图表函数的完整路径，如 /luobei/myapp/charts/sales",
				},
				"url_query": map[string]interface{}{
					"type":        "string",
					"description": "完整 URL 查询串（可选），参数由该 Chart handler 定义，不固定，如 year=2024&month=1&dimension=region",
				},
			},
			"required": []interface{}{"full_code_path"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "run_table_create",
		Description: "执行工作区内 Table 新增接口，批量新增表格记录（每条都会触发 OnTableAddRow）。full_code_path 为表格函数的完整路径（必须包含函数名，如 /luobei/myapp/nps/nps_questionnaire_list）。body 必须为 JSON 数组字符串，每项为一条记录的字段对象，如 [{\"title\":\"问卷A\"},{\"title\":\"问卷B\"}]；字段名与表格 model 的 json 标签一致，必填项需包含。返回 data_list 为成功插入的每条记录（后端返回的数据列表），以及 created_count、failed_count、errors。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "表格函数的完整路径（必须包含函数名），如 /luobei/myapp/nps/nps_questionnaire_list",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "必须为 JSON 数组字符串，每项为一条记录，如 [{\"title\":\"问卷A\",\"description\":\"描述\"},{\"title\":\"问卷B\"}]。字段名与表格 model 的 json 标签一致。",
				},
			},
			"required": []interface{}{"full_code_path", "body"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "run_table_update",
		Description: "执行工作区内 Table 更新接口，批量更新表格记录（每条都会触发 OnTableUpdateRow）。full_code_path 为表格函数的完整路径（必须包含函数名）。body 必须为 JSON 数组字符串，每项为 { \"id\": 行ID, \"updates\": { \"字段名\": 新值, ... } }；不传 old_values，由 app-server 自动查表填充。返回 updated_count、data_list、failed_count、errors。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "表格函数的完整路径（必须包含函数名），如 /luobei/myapp/nps/nps_questionnaire_list",
				},
				"body": map[string]interface{}{
					"type":        "string",
					"description": "必须为 JSON 数组字符串，每项为 { \"id\": 行ID, \"updates\": { \"字段名\": 新值 } }，如 [{\"id\":1,\"updates\":{\"status\":\"已处理\"}},{\"id\":2,\"updates\":{\"status\":\"已关闭\"}}]",
				},
			},
			"required": []interface{}{"full_code_path", "body"},
		},
	})

	// 按模式过滤：若指定了 toolNames，只保留 name 在列表中的工具
	if len(toolNames) > 0 {
		nameSet := make(map[string]struct{}, len(toolNames))
		for _, n := range toolNames {
			nameSet[n] = struct{}{}
		}
		filtered := make([]dto.ToolDef, 0, len(out))
		for _, t := range out {
			if _, ok := nameSet[t.Name]; ok {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	}
	return out, nil
}

// CallTool 执行工具；full_code_path 从会话上下文传入；files 为当前用户消息附件，供插件 InputFiles，可为 nil
// 返回 content 字符串，若 isError 则表示执行失败，content 为错误信息
func (r *ToolRegistry) CallTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files *types.Files) (content string, isError bool) {
	switch name {
	case "read_go_file":
		return r.callReadGoFile(ctx, args, fullCodePath)
	case "read_go_file_lines":
		return r.callReadGoFileLines(ctx, args, fullCodePath)
	case "read_doc":
		return r.callReadDocTool(ctx, args, fullCodePath)
	case "read_dir":
		return r.callReadDir(ctx, args, fullCodePath)
	case "write_doc":
		return r.callWriteDoc(ctx, args, fullCodePath)
	case "write_go_file":
		return r.callWriteGoFile(ctx, args, fullCodePath)
	case "create_directory":
		return RunCreateDirectoryTool(ctx, args, fullCodePath)
	case "build_workspace":
		return r.callWorkspaceBuild(ctx, args, fullCodePath)
	case "search_replace_file":
		return r.callSearchReplaceFile(ctx, args, fullCodePath)
	case "delete_file":
		return r.callDeleteFile(ctx, args, fullCodePath)
	case "run_table_search":
		return r.callRunTableSearch(ctx, args, fullCodePath)
	case "run_form_submit":
		return r.callRunFormSubmit(ctx, args, fullCodePath)
	case "run_chart_query":
		return r.callRunChartQuery(ctx, args, fullCodePath)
	case "run_table_create":
		return r.callRunTableCreate(ctx, args, fullCodePath)
	case "run_table_update":
		return r.callRunTableUpdate(ctx, args, fullCodePath)
	}
	return "tool not found: " + name, true
}

// callReadGoFile 读取工作区 Go 代码文件；若传入的是文档路径则降级为用 read_doc 拉取并提示
func (r *ToolRegistry) callReadGoFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	fileName := GetStringArg(args, "file_name")

	// 降级：若 directory 是内置文档路径，用文档工具拉取内容并提示应使用 read_doc
	if strings.HasPrefix(targetPath, "/builtin/") {
		docPath := strings.TrimSpace(targetPath)
		if !strings.HasPrefix(docPath, "/") {
			docPath = "/" + docPath
		}
		docName, content := prompt.GetBuiltinDocContent(docPath)
		if content != "" {
			hint := "【提示】你当前用 read_go_file 读取的是文档路径。应使用 read_doc(directory: \"" + docPath + "\") 读取文档；已为你拉取内容，下次请用 read_doc。\n\n"
			if docName == "" {
				docName = docPath
			}
			return hint + "## " + docName + "\n\n" + content, false
		}
		return "该路径是内置文档路径，请使用 read_doc(directory: \"" + docPath + "\") 读取，不要用 read_go_file。", true
	}

	// 读代码文件时从 runtime 磁盘实时读，保证内容与当前磁盘一致（快照表可能不准）
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return fmt.Sprintf("获取代码失败: %v", err), true
	}

	if len(workspaceCtx.Files) == 0 {
		return fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false
	}

	var matchedFiles []dto.WorkspaceContextFile
	if fileName != "" {
		// 支持逗号分隔多文件，如 a.go,b.go
		names := splitFileNames(fileName)
		seen := make(map[string]bool)
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			for _, file := range workspaceCtx.Files {
				key := file.RelativePath
				if seen[key] {
					continue
				}
				if file.FileName == name {
					matchedFiles = append(matchedFiles, file)
					seen[key] = true
					continue
				}
				fullFileName := file.FileName + "." + file.FileType
				if fullFileName == name {
					matchedFiles = append(matchedFiles, file)
					seen[key] = true
					continue
				}
				if file.RelativePath == name {
					matchedFiles = append(matchedFiles, file)
					seen[key] = true
					continue
				}
			}
		}
		if len(matchedFiles) == 0 {
			return fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
		}
	} else {
		matchedFiles = workspaceCtx.Files
	}

	var header string
	if fileName != "" {
		header = fmt.Sprintf("文件 %s 的内容（目录：%s）：\n\n", fileName, targetPath)
	} else {
		header = fmt.Sprintf("目录 %s 下的代码文件（共 %d 个）：\n\n", targetPath, len(matchedFiles))
	}

	var filesContent string
	for i, file := range matchedFiles {
		lineCount := file.LineCount
		if lineCount == 0 && file.Content != "" {
			lines := strings.Split(file.Content, "\n")
			lineCount = len(lines)
			if lineCount > 0 && lines[lineCount-1] == "" {
				lineCount--
			}
		}
		// 文件路径：展示完整路径（目录 + 相对路径），便于定位；后端可能只返回文件名
		fullFilePath := strings.TrimRight(targetPath, "/") + "/" + file.RelativePath
		fileHeader := ""
		if len(matchedFiles) > 1 {
			fileHeader = fmt.Sprintf("## 文件 %d: %s\n", i+1, fullFilePath)
		}
		filesContent += fmt.Sprintf(`%s- 文件名: %s
- 文件路径: %s
- 文件类型: %s
- 总行数: %d 行
- 内容长度: %d 字符
- 代码内容:
`+"```%s\n%s\n```\n\n", fileHeader, file.FileName, fullFilePath, file.FileType, lineCount, file.ContentLength, file.FileType, file.Content)
	}
	return header + filesContent, false
}

// callReadGoFileLines 按行号范围读取工作区 Go 文件，输出带行号（便于对照编译错误）
func (r *ToolRegistry) callReadGoFileLines(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	fileName := GetStringArg(args, "file_name")
	lineRangesStr := strings.TrimSpace(GetStringArg(args, "line_ranges"))

	if fileName == "" {
		return "read_go_file_lines 需传 file_name。", true
	}

	// 降级：内置文档路径不处理
	if strings.HasPrefix(targetPath, "/builtin/") {
		return "read_go_file_lines 仅用于工作区 Go 文件，不能读内置文档路径；请用 read_doc 读取文档。", true
	}

	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return fmt.Sprintf("获取代码失败: %v", err), true
	}
	if len(workspaceCtx.Files) == 0 {
		return fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false
	}

	var matched *dto.WorkspaceContextFile
	for i := range workspaceCtx.Files {
		f := &workspaceCtx.Files[i]
		if f.FileName == fileName || f.FileName+"."+f.FileType == fileName || f.RelativePath == fileName {
			matched = f
			break
		}
	}
	if matched == nil {
		return fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
	}

	lines := strings.Split(matched.Content, "\n")
	totalLines := len(lines)
	if totalLines > 0 && lines[totalLines-1] == "" {
		totalLines--
	}

	// 解析 line_ranges：如 "10-12,20-30" -> [{10,12},{20,30}]，行号 1-based
	ranges := parseLineRanges(lineRangesStr, totalLines)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("文件 %s（目录：%s）共 %d 行\n\n", matched.RelativePath, targetPath, totalLines))

	// 行号显示宽度
	width := 1
	for n := totalLines; n >= 10; n /= 10 {
		width++
	}

	for idx, rng := range ranges {
		for i := rng.Start; i <= rng.End && i <= totalLines; i++ {
			lineContent := ""
			if i-1 < len(lines) {
				lineContent = lines[i-1]
			}
			sb.WriteString(fmt.Sprintf("%*d | %s\n", width, i, lineContent))
		}
		if len(ranges) > 1 && idx < len(ranges)-1 {
			sb.WriteString("...\n")
		}
	}

	return sb.String(), false
}

// lineRange 行号范围，1-based 包含两端
type lineRange struct{ Start, End int }

// parseLineRanges 解析 "10-12,20-30" 或 "10,20-22"；空字符串表示全文，返回 []{1, totalLines}
func parseLineRanges(s string, totalLines int) []lineRange {
	s = strings.TrimSpace(s)
	if totalLines <= 0 {
		totalLines = 1
	}
	if s == "" {
		return []lineRange{{1, totalLines}}
	}
	var out []lineRange
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "-")
		if idx < 0 {
			// 单行，如 "10"
			var n int
			if _, err := fmt.Sscanf(part, "%d", &n); err == nil && n >= 1 {
				if n > totalLines {
					n = totalLines
				}
				out = append(out, lineRange{n, n})
			}
			continue
		}
		var start, end int
		if _, err := fmt.Sscanf(part, "%d-%d", &start, &end); err == nil && start >= 1 {
			if start > totalLines {
				start = totalLines
			}
			if end < start {
				end = start
			}
			if end > totalLines {
				end = totalLines
			}
			out = append(out, lineRange{start, end})
		}
	}
	if len(out) == 0 {
		return []lineRange{{1, totalLines}}
	}
	return out
}

// callReadDocTool 读取文档（directory 唯一定位，内置或工作区；支持逗号分隔多路径）
func (r *ToolRegistry) callReadDocTool(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	dirArg := strings.TrimSpace(GetStringArg(args, "directory"))
	if dirArg == "" {
		dirArg = strings.TrimSpace(getDirectory(args, currentFullCodePath))
	}
	if dirArg == "" {
		return "read_doc 需传 directory。", true
	}
	// 支持逗号分隔多路径，去重且保持顺序
	paths := splitDirectoryPaths(dirArg)
	if len(paths) == 0 {
		return "read_doc 需传 directory。", true
	}

	var sb strings.Builder
	var hasError bool
	for i, fullCodePath := range paths {
		if fullCodePath == "" {
			continue
		}
		if !strings.HasPrefix(fullCodePath, "/") {
			fullCodePath = "/" + fullCodePath
		}

		if strings.HasPrefix(fullCodePath, "/builtin/") {
			docName, content := prompt.GetBuiltinDocContent(fullCodePath)
			if content == "" {
				if i > 0 {
					sb.WriteString("\n\n")
				}
				sb.WriteString(fmt.Sprintf("## %s\n\n未找到：directory=%s。请使用系统消息中列出的 directory。", fullCodePath, fullCodePath))
				hasError = true
				continue
			}
			if docName == "" {
				docName = fullCodePath
			}
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n%s", docName, content))
			continue
		}

		doc, err := apicall.GetDoc(ctx, fullCodePath)
		if err != nil {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n获取文档失败: %v", fullCodePath, err))
			hasError = true
			continue
		}
		if doc == nil || doc.Content == "" {
			if i > 0 {
				sb.WriteString("\n\n")
			}
			sb.WriteString(fmt.Sprintf("## %s\n\n文档《%s》无正文内容。", fullCodePath, fullCodePath))
			continue
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n%s", doc.Name, doc.Content))
	}
	return sb.String(), hasError
}

// callReadDir 读取目录工具：读取指定目录下所有子节点和文件，支持列表模式和递归树形模式
func (r *ToolRegistry) callReadDir(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)

	// 获取参数
	recursive := false
	if recursiveArg, ok := args["recursive"].(bool); ok {
		recursive = recursiveArg
	}

	maxDepth := -1 // 默认不限制
	if maxDepthArg, ok := args["max_depth"].(float64); ok {
		maxDepth = int(maxDepthArg)
	}

	outputFormat := "tree" // 默认树形格式，便于展开查看目录和文件
	if formatArg, ok := args["output_format"].(string); ok && formatArg != "" {
		outputFormat = formatArg
	}

	includeFunctions := true // 默认包含函数
	if includeFunctionsArg, ok := args["include_functions"].(bool); ok {
		includeFunctions = includeFunctionsArg
	}

	includeFiles := true // 默认包含代码文件，与「读取该文件夹下所有文件和目录」预期一致
	if includeFilesArg, ok := args["include_files"].(bool); ok {
		includeFiles = includeFilesArg
	}

	includeCode := false // 默认不包含代码内容
	if includeCodeArg, ok := args["include_code"].(bool); ok {
		includeCode = includeCodeArg
	}

	// 需要文件列表时用 runtime 从磁盘读，否则用快照；默认读取文件
	fileSource := ""
	if includeFiles {
		fileSource = "runtime"
	}
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, fileSource)
	if err != nil {
		return fmt.Sprintf("获取目录信息失败: %v", err), true
	}

	// 树形格式：recursive=true 时整棵树，recursive=false 时只展开当前一层（max_depth=1）
	if outputFormat == "tree" {
		treeMaxDepth := maxDepth
		if !recursive {
			treeMaxDepth = 1
		}
		return r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, treeMaxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
	}
	if recursive {
		return r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
	}

	// 列表格式显示当前目录
	return r.buildListFormat(ctx, workspaceCtx, targetPath, includeFunctions, includeFiles, includeCode, outputFormat)
}

// callReadDoc 按文档名称从嵌入的 content/doc/文档目录 查 full_code_path 后返回内置文档正文（兼容 doc_name 调用）
func (r *ToolRegistry) callReadDoc(ctx context.Context, args map[string]interface{}) (string, bool) {
	docName := strings.TrimSpace(GetStringArg(args, "doc_name"))
	if docName == "" {
		return "read_file 读文档需传 doc_name（与「当前可读文档」列表中的名称一致）", true
	}
	name, content := prompt.GetBuiltinDocContentByName(docName)
	if content == "" {
		return "read_file 未找到文档：\"" + docName + "\"。请使用系统消息「当前可读文档」列表中列出的文档名称。", true
	}
	if name == "" {
		name = docName
	}
	return fmt.Sprintf("## %s\n\n%s", name, content), false
}

// buildListFormat 构建列表格式输出
func (r *ToolRegistry) buildListFormat(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, includeFunctions, includeFiles, includeCode bool, outputFormat string) (string, bool) {

	// 分类子节点
	var directories []dto.WorkspaceContextNode
	var functions []dto.WorkspaceContextNode
	for _, child := range workspaceCtx.Children {
		if child.Type == "package" || child.Type == "docs" {
			directories = append(directories, child)
		} else if child.Type == "function" && includeFunctions {
			functions = append(functions, child)
		}
	}

	// 构建目录信息部分
	dirInfo := fmt.Sprintf(`## 目录信息：%s

- 目录名称：%s
- 目录代码：%s
- 完整路径：%s`, targetPath, workspaceCtx.Directory.Name, workspaceCtx.Directory.Code, workspaceCtx.Directory.FullCodePath)

	if workspaceCtx.Directory.Description != "" {
		dirInfo += fmt.Sprintf("\n- 目录描述：%s", workspaceCtx.Directory.Description)
	}
	dirInfo += "\n\n"

	// 构建子目录部分
	dirsSection := ""
	if len(directories) > 0 {
		dirsSection = fmt.Sprintf("### 子目录（共 %d 个）\n\n", len(directories))
		for i, dir := range directories {
			dirsSection += fmt.Sprintf(`#### 目录 %d: %s
- 目录代码：%s
- 类型：%s
- 完整路径：%s`, i+1, dir.Name, dir.Code, dir.Type, dir.FullCodePath)
			if dir.Description != "" {
				dirsSection += fmt.Sprintf("\n- 描述：%s", dir.Description)
			}
			dirsSection += "\n\n"
		}
	}

	// 构建函数部分（含 template_type，与 env 中「当前目录下的可执行函数」一致，便于 run_table_search 等直接使用 full_code_path）
	funcsSection := ""
	if len(functions) > 0 {
		funcsSection = fmt.Sprintf("### 函数/文件（共 %d 个）\n\n", len(functions))
		for i, fn := range functions {
			tpl := fn.TemplateType
			if tpl == "" {
				tpl = "function"
			}
			funcsSection += fmt.Sprintf(`#### 函数 %d: %s
- 函数代码：%s
- 类型：%s
- 完整路径：%s`, i+1, fn.Name, fn.Code, tpl, fn.FullCodePath)
			if fn.Description != "" {
				funcsSection += fmt.Sprintf("\n- 描述：%s", fn.Description)
			}
			funcsSection += "\n\n"
		}
	}

	// 构建文件部分
	filesSection := ""
	if includeFiles && len(workspaceCtx.Files) > 0 {
		filesSection = fmt.Sprintf("### 代码文件（共 %d 个）\n\n", len(workspaceCtx.Files))
		for i, file := range workspaceCtx.Files {
			// 降级处理：如果行数为0（可能是旧数据），则动态计算
			lineCount := file.LineCount
			if lineCount == 0 && file.Content != "" {
				lines := strings.Split(file.Content, "\n")
				lineCount = len(lines)
				if lineCount > 0 && lines[lineCount-1] == "" {
					lineCount--
				}
			}

			filesSection += fmt.Sprintf(`#### 文件 %d: %s
- 文件名：%s
- 文件类型：%s
- 总行数：%d 行
- 内容长度：%d 字符`, i+1, file.RelativePath, file.FileName, file.FileType, lineCount, file.ContentLength)

			if includeCode {
				filesSection += fmt.Sprintf("\n- 代码内容：\n```%s\n%s\n```", file.FileType, file.Content)
			} else {
				filesSection += "\n- 提示：如需查看代码内容，请使用 read_go_file 工具或设置 include_code=true"
			}
			filesSection += "\n\n"
		}
	} else if !includeFiles && len(workspaceCtx.Files) > 0 {
		filesSection = fmt.Sprintf("### 代码文件\n当前目录下有 %d 个代码文件（使用 include_files=true 查看详情）\n\n", len(workspaceCtx.Files))
	}

	// 如果没有子节点
	if len(directories) == 0 && len(functions) == 0 {
		if dirsSection == "" && funcsSection == "" {
			dirsSection = "### 子节点\n当前目录下没有子节点。\n\n"
		}
	}

	return dirInfo + dirsSection + funcsSection + filesSection, false
}

// buildRecursiveTree 构建递归树形结构
func (r *ToolRegistry) buildRecursiveTree(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, fileSource string, outputFormat string) (string, bool) {
	// 检查深度限制
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return "", false
	}

	// 构建树形结构
	treeLines := r.buildTreeLines(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, fileSource, "")

	if outputFormat == "tree" {
		return fmt.Sprintf(`目录树：%s

%s`, targetPath, treeLines), false
	} else {
		// list 格式的递归输出（可以后续优化）
		return fmt.Sprintf(`目录树（递归）：%s

%s`, targetPath, treeLines), false
	}
}

// buildTreeLines 递归构建树形结构的字符串（不使用 strings.Builder）
func (r *ToolRegistry) buildTreeLines(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, currentPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, fileSource string, prefix string) string {
	// 检查深度限制
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return ""
	}

	var result string

	// 输出当前目录（根目录时显示 full-code-path）
	if currentDepth == 0 {
		result = fmt.Sprintf("%s [%s]\n", workspaceCtx.Directory.Name, workspaceCtx.Directory.FullCodePath)
	}

	// 获取当前目录的子节点
	children := workspaceCtx.Children
	// 只展示当前目录直接下的文件（RelativePath 不含 "/"），避免 app-runtime 递归返回的子目录文件被重复列在根下
	files := make([]dto.WorkspaceContextFile, 0, len(workspaceCtx.Files))
	for _, f := range workspaceCtx.Files {
		if f.RelativePath != "" && !strings.Contains(f.RelativePath, "/") {
			files = append(files, f)
		}
	}

	// 处理子目录和函数
	directories := make([]dto.WorkspaceContextNode, 0)
	functions := make([]dto.WorkspaceContextNode, 0)
	for _, child := range children {
		if child.Type == "package" || child.Type == "docs" {
			directories = append(directories, child)
		} else if child.Type == "function" && includeFunctions {
			functions = append(functions, child)
		}
	}

	// 计算总项目数（用于判断是否是最后一项）
	totalItems := len(directories)
	if includeFunctions {
		totalItems += len(functions)
	}
	if includeFiles {
		totalItems += len(files)
	}

	// 输出子目录
	for i, dir := range directories {
		isLast := i == len(directories)-1 && (!includeFunctions || len(functions) == 0) && (!includeFiles || len(files) == 0)
		connector := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			nextPrefix = prefix + "    "
		}

		// 格式：code(名称-描述)[类型]
		descPart := ""
		if dir.Description != "" {
			descPart = "-" + dir.Description
		}
		result += fmt.Sprintf("%s%s%s(%s%s)[%s]\n", prefix, connector, dir.Code, dir.Name, descPart, dir.Type)

		// 递归查询子目录的内容（需要文件时用 runtime 从磁盘读）
		childCtx, err := apicall.GetWorkspaceContext(ctx, dir.FullCodePath, fileSource)
		if err == nil {
			result += r.buildTreeLines(ctx, childCtx, dir.FullCodePath, currentDepth+1, maxDepth, includeFunctions, includeFiles, fileSource, nextPrefix)
		} else {
			result += fmt.Sprintf("%s    (无法获取子目录内容: %v)\n", nextPrefix, err)
		}
	}

	// 输出函数节点（含 template_type 与 full_code_path，与 env「当前目录下的可执行函数」一致）
	if includeFunctions && len(functions) > 0 {
		for i, fn := range functions {
			isLast := i == len(functions)-1 && (!includeFiles || len(files) == 0)
			connector := "├── "
			if isLast {
				connector = "└── "
			}
			tpl := fn.TemplateType
			if tpl == "" {
				tpl = "function"
			}
			descPart := ""
			if fn.Description != "" {
				descPart = "-" + fn.Description
			}
			result += fmt.Sprintf("%s%s%s(%s%s)[%s]", prefix, connector, fn.Code, fn.Name, descPart, tpl)
			if fn.FullCodePath != "" {
				result += fmt.Sprintf(" → %s", fn.FullCodePath)
			}
			result += "\n"
		}
	}

	// 处理文件
	if includeFiles {
		for i, file := range files {
			isLast := i == len(files)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			// 降级处理：如果行数为0（可能是旧数据），则动态计算
			lineCount := file.LineCount
			if lineCount == 0 && file.Content != "" {
				lines := strings.Split(file.Content, "\n")
				lineCount = len(lines)
				if lineCount > 0 && lines[lineCount-1] == "" {
					lineCount--
				}
			}

			result += fmt.Sprintf("%s%s%s.go (%d 行)\n", prefix, connector, file.FileName, lineCount)
		}
	}

	return result
}

// callWriteDoc 写文档：目录 + name + code + content，内部调用 RunWriteDocTool
func (r *ToolRegistry) callWriteDoc(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	// RunWriteDocTool 使用 full_code_path；将 directory 写入 args 供其使用
	argsWithPath := make(map[string]interface{}, len(args)+1)
	for k, v := range args {
		argsWithPath[k] = v
	}
	argsWithPath["full_code_path"] = targetPath
	return RunWriteDocTool(ctx, argsWithPath, currentFullCodePath)
}

// callWriteGoFile 写 Go 代码文件；build_workspace=false 时仅写不编译
func (r *ToolRegistry) callWriteGoFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	if fileName == "" {
		return "write_go_file 缺少参数 file_name。", true
	}
	content := GetStringArg(args, "content")
	if content == "" {
		content = GetStringArg(args, "source_code")
	}
	if content == "" {
		return "write_go_file 缺少参数 content。", true
	}
	if !strings.HasSuffix(fileName, ".go") {
		fileName = fileName + ".go"
	}
	nameWithoutExt := strings.TrimSuffix(fileName, ".go")
	if nameWithoutExt == "init_" {
		return "不允许创建该文件，由脚手架自动生成。", true
	}
	buildWorkspace := true
	if v, ok := args["build_workspace"]; ok {
		if b, ok := v.(bool); ok {
			buildWorkspace = b
		}
	}

	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	writeArgs := map[string]interface{}{
		"file_name":   fileName,
		"source_code": content,
	}
	return RunAddFunctionsTool(ctx, writeArgs, targetPath, true, buildWorkspace)
}

// callWorkspaceBuild 编译当前工作空间（不写文件，仅触发编译并部署）；从当前工作目录解析 user/app，无需参数
func (r *ToolRegistry) callWorkspaceBuild(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	dir := strings.Trim(strings.TrimSpace(currentFullCodePath), "/")
	if dir == "" {
		return "build_workspace 无法获取当前工作目录，请确保在有效的工作台会话中操作", true
	}
	parts := strings.Split(dir, "/")
	if len(parts) < 2 {
		return "build_workspace 当前目录格式应为 /user/app 或更长路径（如 /luobei/demo）", true
	}
	user, app := parts[0], parts[1]
	resp, err := apicall.UpdateAppBuild(ctx, user, app)
	if err != nil {
		logger.Errorf(ctx, "[WorkspaceBuild] UpdateAppBuild 失败: %v", err)
		return "build_workspace 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("工作空间已编译并部署: app=%s, 旧版本=%s, 新版本=%s", resp.App, resp.OldVersion, resp.NewVersion), false
}

// parseReplacementsFromArgs 从工具参数中解析 replacements 数组为 []dto.ReplaceItem
func parseReplacementsFromArgs(args map[string]interface{}) ([]dto.ReplaceItem, bool) {
	raw, ok := args["replacements"]
	if !ok {
		return nil, false
	}
	slice, ok := raw.([]interface{})
	if !ok || len(slice) == 0 {
		return nil, false
	}
	out := make([]dto.ReplaceItem, 0, len(slice))
	for _, v := range slice {
		item, ok := v.(map[string]interface{})
		if !ok {
			return nil, false
		}
		search := GetStringArg(item, "search_string")
		if search == "" {
			return nil, false
		}
		expected := 0
		if n, ok := item["expected_count"]; ok {
			switch t := n.(type) {
			case float64:
				expected = int(t)
			case int:
				expected = t
			}
		}
		out = append(out, dto.ReplaceItem{
			SearchString:  search,
			ReplaceString: GetStringArg(item, "replace_string"),
			ExpectedCount: expected,
		})
	}
	return out, true
}

// callSearchReplaceFile 文件 search-replace（统一批量：多组替换同一文件，全部生效才落盘）
func (r *ToolRegistry) callSearchReplaceFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	if fileName == "" {
		return "search_replace_file 缺少参数 file_name。", true
	}
	replacements, ok := parseReplacementsFromArgs(args)
	if !ok || len(replacements) == 0 {
		return "search_replace_file 缺少参数 replacements（替换列表，每项含 search_string、replace_string、expected_count 可选）。", true
	}
	allOrNothing := true
	if v, ok := args["all_or_nothing"]; ok {
		if b, ok := v.(bool); ok {
			allOrNothing = b
		}
	}
	returnFullContent := true
	if v, ok := args["return_full_content"]; ok {
		if b, ok := v.(bool); ok {
			returnFullContent = b
		}
	}
	req := &dto.ReplaceFileContentReq{
		FullCodePath:      targetPath,
		FileName:          fileName,
		Replacements:      replacements,
		AllOrNothing:      allOrNothing,
		ReturnFullContent: returnFullContent,
	}
	resp, err := apicall.ReplaceFileContent(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[SearchReplaceFile] ReplaceFileContent 失败: %v", err)
		return "search_replace_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		msg := "search_replace_file: " + resp.Message
		if len(resp.Details) > 0 {
			for _, d := range resp.Details {
				msg += fmt.Sprintf("\n第 %d 项预期匹配 %d 次，实际匹配 %d 次；请将该项 expected_count 改为 %d 或核对内容后再调用。", d.Index+1, d.ExpectedCount, d.ActualCount, d.ActualCount)
			}
		}
		return msg, true
	}
	msg := fmt.Sprintf("已替换: 目录=%s, 文件=%s, 共 %d 处。修改已落盘，但未编译工作空间；若需生效请调用 build_workspace 更新工作空间。", targetPath, fileName, resp.ReplaceCount)
	if resp.FullContent != "" {
		msg += "\n\n替换后完整内容：\n```go\n" + resp.FullContent + "\n```"
	}
	return msg, false
}

// callDeleteFile 删除目录下指定 .go 文件（删磁盘+删节点）
func (r *ToolRegistry) callDeleteFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	if fileName == "" {
		return "delete_file 缺少参数 file_name。", true
	}
	req := &dto.DeleteFileReq{
		FullCodePath: targetPath,
		FileName:     fileName,
	}
	resp, err := apicall.DeleteFile(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[DeleteFile] DeleteFile 失败: %v", err)
		return "delete_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "delete_file: " + resp.Message, true
	}
	return fmt.Sprintf("已删除: 目录=%s, 文件=%s", targetPath, fileName), false
}

// callRunTableSearch 执行 Table 查询（执行模式专用）；参数遵循 pkg/gormx/query，可传 url_query 或 page/page_size/sorts
func (r *ToolRegistry) callRunTableSearch(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	if fullCodePath == "" {
		return "run_table_search 需传 full_code_path（表格函数路径，如 /luobei/myapp/tables/hr）。", true
	}
	var params url.Values
	if q := strings.TrimSpace(GetStringArg(args, "url_query")); q != "" {
		parsed, err := url.ParseQuery(q)
		if err != nil {
			return "run_table_search 的 url_query 需为合法查询串: " + err.Error(), true
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
		if v, ok := args["page"]; ok {
			if n, ok := toInt(v); ok {
				params.Set("page", strconv.Itoa(n))
			}
		}
		if v, ok := args["page_size"]; ok {
			if n, ok := toInt(v); ok {
				params.Set("page_size", strconv.Itoa(n))
			}
		}
		if s := GetStringArg(args, "sorts"); s != "" {
			params.Set("sorts", s)
		}
	}
	// 将 url_query 中的时间表达式（Now()、Today()、Now(2026-02-01 13:05:05)、Now(-7d) 等）替换为毫秒时间戳
	for key := range params {
		params.Set(key, timex.ReplaceTimeExprsInParamValue(params.Get(key)))
	}
	result, err := apicall.TableSearch(ctx, fullCodePath, params)
	if err != nil {
		logger.Errorf(ctx, "[RunTableSearch] TableSearch 失败: %v", err)
		return "run_table_search 调用失败: " + err.Error(), true
	}
	return formatJSONResult(result)
}

// callRunFormSubmit 执行 Form 提交（执行模式专用）
func (r *ToolRegistry) callRunFormSubmit(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	if fullCodePath == "" {
		return "run_form_submit 需传 full_code_path（表单函数路径，如 /luobei/myapp/plugins/xxx）。", true
	}
	bodyStr := GetStringArg(args, "body")
	var body interface{}
	if bodyStr != "" {
		if err := json.Unmarshal([]byte(bodyStr), &body); err != nil {
			return "run_form_submit 的 body 需为合法 JSON 字符串: " + err.Error(), true
		}
	} else {
		body = map[string]interface{}{}
	}
	result, err := apicall.FormSubmit(ctx, fullCodePath, body)
	if err != nil {
		logger.Errorf(ctx, "[RunFormSubmit] FormSubmit 失败: %v", err)
		return "run_form_submit 调用失败: " + err.Error(), true
	}
	return formatJSONResult(result)
}

// callRunChartQuery 执行 Chart 查询（执行模式专用）；图表参数不固定，由 handler 定义，可传 url_query
func (r *ToolRegistry) callRunChartQuery(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	if fullCodePath == "" {
		return "run_chart_query 需传 full_code_path（图表函数路径，如 /luobei/myapp/charts/xxx）。", true
	}
	var params url.Values
	if q := strings.TrimSpace(GetStringArg(args, "url_query")); q != "" {
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
	return formatJSONResult(result)
}

// callRunTableCreate 执行 Table 新增（执行模式专用）；body 必须为 JSON 数组，逐条调用 table/create 触发 OnTableAddRow，返回 data_list（成功记录列表）及汇总
func (r *ToolRegistry) callRunTableCreate(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	if fullCodePath == "" {
		return "run_table_create 需传 full_code_path（表格函数路径，如 /luobei/myapp/nps/nps_questionnaire_list）。", true
	}
	bodyStr := GetStringArg(args, "body")
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
		// 每条必须是对象（map），否则后端无法当作单条记录处理
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
		// 后端 table/create 返回的 result 是 ApiResult 的 data 部分，即 OnTableAddRowResp 序列化：{ "data": <行数据> }，需取出 result["data"] 作为单条记录
		record := extractTableCreateRecord(result)
		dataList = append(dataList, record)
	}

	out := map[string]interface{}{
		"created_count": createdCount,
		"failed_count":  failedCount,
		"data_list":     dataList,
	}
	if len(errorsList) > 0 {
		out["errors"] = errorsList
	}
	return formatJSONResult(out)
}

// callRunTableUpdate 执行 Table 批量更新（执行模式专用）；body 为 JSON 数组，每项 { id, updates }，app-server 自动填充 old_values
func (r *ToolRegistry) callRunTableUpdate(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	if fullCodePath == "" {
		return "run_table_update 需传 full_code_path（表格函数路径，如 /luobei/myapp/nps/nps_questionnaire_list）。", true
	}
	bodyStr := GetStringArg(args, "body")
	if bodyStr == "" {
		return "run_table_update 需传 body（JSON 数组字符串，每项为 { \"id\": 行ID, \"updates\": { \"字段\": 新值 } }）。", true
	}
	var bodyArr []interface{}
	if err := json.Unmarshal([]byte(bodyStr), &bodyArr); err != nil {
		return "run_table_update 的 body 需为合法 JSON 数组: " + err.Error(), true
	}
	if len(bodyArr) == 0 {
		return "run_table_update 的 body 不能为空数组。", true
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
	return formatJSONResult(out)
}

// extractTableCreateRecord 从 table/create 的返回值中提取单条记录。
// 后端返回的 result 是 ApiResult 的 data 部分：OnTableAddRowResp 序列化为 { "data": <行数据> }，取小写 "data" 作为行数据。
func extractTableCreateRecord(result map[string]interface{}) interface{} {
	if result == nil {
		return result
	}
	if v, ok := result["data"]; ok && v != nil {
		return v
	}
	return result
}

// toInt 从 interface{} 转 int（支持 float64/int）
func toInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// formatJSONResult 将 map 序列化为可读字符串
func formatJSONResult(m map[string]interface{}) (string, bool) {
	if m == nil {
		return "{}", false
	}
	b, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", m), false
	}
	return string(b), false
}

// ToToolArgs 将 interface{} 转为 map[string]interface{}，供 CallTool 使用
// JSON 反序列化后，object→map[string]interface{}；nil/null/缺省→nil，按空 map 处理
func ToToolArgs(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// GetStringArg 从 args 取 string
func GetStringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// splitFileNames 将 file_name 按逗号拆成多个文件名（如 "a.go,b.go" -> ["a.go","b.go"]），单文件返回单元素
func splitFileNames(fileName string) []string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return nil
	}
	parts := strings.Split(fileName, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// splitDirectoryPaths 将 directory 按逗号拆成多个路径并去重（如 "/a,/b,/a" -> ["/a","/b"]），保持顺序
func splitDirectoryPaths(directory string) []string {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return nil
	}
	parts := strings.Split(directory, ",")
	seen := make(map[string]bool)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	return out
}

// getDirectory 取目录参数：模型侧用 directory，兼容旧参数 full_code_path；未传则用 defaultPath
func getDirectory(args map[string]interface{}, defaultPath string) string {
	s := strings.TrimSpace(GetStringArg(args, "directory"))
	if s != "" {
		return s
	}
	s = strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if s != "" {
		return s
	}
	return defaultPath
}
