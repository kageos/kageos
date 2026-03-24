package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/contextx"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/pkg/timex"
	"github.com/ai-agent-os/ai-agent-os/pkg/websearch"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// workspaceContextKey 用于 context 中传递工作台会话 ID，便于埋点追溯
type workspaceContextKey struct{}

// WorkspaceSessionIDKey 工作台会话 ID 的 context key（在 executeToolCalls 中注入，callRecordWorkspaceEvent 中读取）
var WorkspaceSessionIDKey = workspaceContextKey{}

// runOfficialPythonPreinstallDoc 与 build/Dockerfile 中 apt/python3-* 与 pip3 install 预装保持一致；改镜像时请同步更新本文案
const runOfficialPythonPreinstallDoc = `**生产镜像已预装、可直接 import 的第三方库（对应 build/Dockerfile）：**
- 数据与图表：pandas、numpy、scipy、matplotlib、seaborn
- 网络与表格：requests、openpyxl
- 图像：PIL（Pillow，如 from PIL import Image）
- 文档与 PDF：docx（python-docx）、PyPDF2、pdfplumber
- 中文分词：jieba
- 另有 **Python 标准库**（json、re、collections、datetime、itertools、math、random 等）

**若 import 报错：** 优先改用上面列表或标准库；需要新依赖时请管理员更新 Dockerfile / 官方 requirements.txt 并重打镜像。不可在本工具参数里指定 pip 包。
**环境差异：** 本地非 Docker 运行时以本机 python 为准，可能与镜像不一致。`

func getWorkspaceSessionID(ctx context.Context) string {
	if v := ctx.Value(WorkspaceSessionIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// ToolRegistry 工作台工具注册与调用（仅内置工具，已移除插件）
// list_tools：仅内置；call_tool(name, args, full_code_path) 路由到对应实现
type ToolRegistry struct {
	eventRepo *repository.WorkspaceEventRepository
}

// NewToolRegistry 创建 ToolRegistry（eventRepo 可为 nil，则 record_workspace_event 仅打日志不落库）
func NewToolRegistry(eventRepo *repository.WorkspaceEventRepository) *ToolRegistry {
	return &ToolRegistry{eventRepo: eventRepo}
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

	// write_go_file：写 Go 代码文件（只落盘，不编译）
	out = append(out, dto.ToolDef{
		Name:        "write_go_file",
		Description: "在当前工作目录或指定 directory 下写入一个 .go 代码文件。必填：file_name（如 attendance.go）、content（Go 源码）。可选：directory（目标目录）。注意：write_go_file 只落盘、不编译；可连续多次写入多个文件，全部写完后统一调用一次 build_workspace 完成编译与部署，无需每写一次就编译。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 attendance.go、biz_vote_system.go",
				},
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目标目录（可选），不传则当前工作目录",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Go 源码全文",
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
					"description": "是否在结果中返回替换后的完整文件内容（可选，默认 false）",
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

	// read_app_log：读取应用日志（支持指定版本与关键词检索）
	out = append(out, dto.ToolDef{
		Name:        "read_app_log",
		Description: "读取应用日志（workspace/logs），用于排查 bug、报错、超时、异常行为等运行问题。默认读取当前版本日志；可传 version 指定历史版本（如 v48）。支持按关键词过滤（keyword），并返回命中上下文。参数：directory（可选，不传则当前目录）、version（可选，默认当前版本）、lines（可选，默认 200，最大 1000）、keyword（可选）、context_lines（可选，默认 2，最大 5）、max_matches（可选，默认 50，最大 200）、ignore_case（可选，默认 false）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录，如 /luobei/minimax",
				},
				"version": map[string]interface{}{
					"type":        "string",
					"description": "版本号（可选），如 v48；不传默认当前版本",
				},
				"lines": map[string]interface{}{
					"type":        "integer",
					"description": "返回行数（可选），默认 200，最大 1000",
				},
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "关键词（可选）。传入后按关键词检索并返回命中上下文",
				},
				"context_lines": map[string]interface{}{
					"type":        "integer",
					"description": "关键词命中上下文行数（可选），默认 2，最大 5",
				},
				"max_matches": map[string]interface{}{
					"type":        "integer",
					"description": "关键词模式最大命中数（可选），默认 50，最大 200",
				},
				"ignore_case": map[string]interface{}{
					"type":        "boolean",
					"description": "关键词匹配是否忽略大小写（可选），默认 false",
				},
			},
			"required": []interface{}{},
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
		Description: "执行工作区内 Form 函数的提交接口，提交表单数据。full_code_path 为表单函数的完整路径，如 /luobei/myapp/plugins/cashier_desk。body 为 JSON 对象字符串，包含表单字段（如 {\"name\":\"张三\",\"amount\":100}）；若表单无必填字段可传 {}。output_display 可选，用于标记结果中需要在前端直接展示给用户的字段（避免大模型重复输出大段内容），key 为展示标签，value 为结果 JSON 中的字段名。返回中若有输出文件 URL（多为内部地址如 host.containers.internal），勿在回复用户时贴出或写「可通过以下链接访问」；文件已在工作台展示。",
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
				"output_display": map[string]interface{}{
					"type":        "object",
					"description": "可选。标记结果中需要在前端直接展示给用户的字段，key 为展示标签，value 为结果 JSON 中的字段名。例如 {\"识别到的文本\":\"output_text\",\"页数\":\"page_count\"}，前端会自动提取对应字段值并在工具结果旁独立展示，方便用户查看和复制",
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
	// web_search：默认百度搜索，失败或无结果时回退必应；环境变量 WEB_SEARCH_ENGINE=bing|baidu 可强制单一引擎
	out = append(out, dto.ToolDef{
		Name:        "web_search",
		Description: "在互联网上搜索知识、概念或资料。默认使用百度搜索，必要时回退必应（国内可直接访问，不调用第三方付费 API）。当需要最新信息、概念解释、技术文档或事实查证时调用。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词（必填），如「Go 1.22 新特性」「REST API 设计规范」",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "最多返回条数（可选，默认 10，最大 20）",
				},
			},
			"required": []interface{}{"keyword"},
		},
	})
	// fetch_url_content：按 URL 拉取该页正文，支持单链接或多链接
	out = append(out, dto.ToolDef{
		Name:        "fetch_url_content",
		Description: "根据指定 URL（或多个 URL）访问并拉取可读正文。支持 HTML 页面（解析 DOM 取文）、纯文本、Markdown、JSON/XML 等文本类响应；非文本（如二进制）也会返回简短说明（含 Content-Type 与大小）。支持传 url（单个）或 urls（多个，最多 5 个）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "要访问的单个网页 URL，与 urls 二选一",
				},
				"urls": map[string]interface{}{
					"type":        "array",
					"description": "要访问的多个网页 URL，与 url 二选一；最多 5 个，超出只取前 5 个",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"max_chars": map[string]interface{}{
					"type":        "integer",
					"description": "每条正文最多返回字数（可选，默认 3000，最大 20000）",
				},
			},
			"required": []interface{}{},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "search_tools",
		Description: "按关键词搜索可用工具：返回「内置工具」与「system 用户下已注册的表单/表格/图表函数」。keyword 可选：不传则按调用次数返回高频已注册函数；传则按关键词匹配。多关键词用竖线 | 分隔（OR 语义），如 折线图|chart|画图。template_type 建议杂活传 form。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词（可选）。不传则按调用次数返回高频已注册函数；传则按关键词匹配，支持多关键词用竖线 | 分隔，如 折线图|chart|画图 或 视频|video|流媒体",
				},
				"template_type": map[string]interface{}{
					"type":        "string",
					"description": "按函数类型过滤（可选）：form（绝大部分杂活、画图、转换用此）/ table / chart。不传则返回全部类型；建议杂活类传 form",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "最多返回条数（可选，默认 20）",
				},
			},
			"required": []interface{}{},
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
	out = append(out, dto.ToolDef{
		Name:        "create_scheduled_task",
		Description: "创建定时任务。支持 execute/form（普通函数，form 会自动映射为 execute）、table_create（表格新增）、table_update（表格更新）、table_delete（表格删除）。full_code_path 可不传（默认当前目录）。table_update 的 payload 需包含 id 与 updates，执行时会自动补 old_values。run_at 建议用本地日期时间字符串（无 Z），与前端一致；也可用带时区偏移的 RFC3339。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "任务名称，如 每晚同步库存",
				},
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "函数完整路径（可选，不传默认当前工作目录）",
				},
				"action": map[string]interface{}{
					"type":        "string",
					"description": "动作（可选，默认 execute）：execute/form/table_create/table_update/table_delete",
				},
				"method": map[string]interface{}{
					"type":        "string",
					"description": "请求方法（可选，默认 POST）",
				},
				"payload": map[string]interface{}{
					"type":        "string",
					"description": "JSON 对象字符串（可选，默认 {}）。table_update 示例：{\"id\":1,\"updates\":{\"status\":\"done\"}}；table_delete 示例：{\"ids\":[1,2]}",
				},
				"schedule_type": map[string]interface{}{
					"type":        "string",
					"description": "调度类型：atime/cron/every",
				},
				"run_at": map[string]interface{}{
					"type":        "string",
					"description": "首次执行时间。推荐：本地日期时间 \"2006-01-02 15:04:05\" 或 \"2006-01-02T15:04:05\"（按服务器本地时区解析，与界面一致）。也可用 RFC3339 且必须带偏移，如 2026-03-20T23:00:00+08:00；勿单独使用末尾 Z（会被当作 UTC）。",
				},
				"cron_expr": map[string]interface{}{
					"type":        "string",
					"description": "cron 表达式（schedule_type=cron 时必填）",
				},
				"interval_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "间隔秒数（schedule_type=every 时必填）",
				},
				"max_runs": map[string]interface{}{
					"type":        "integer",
					"description": "最多执行次数（every 可选，0 表示不限制）",
				},
				"timezone": map[string]interface{}{
					"type":        "string",
					"description": "时区（可选）",
				},
			},
			"required": []interface{}{"name", "schedule_type", "run_at"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "list_scheduled_tasks",
		Description: "查询定时任务列表。full_code_path 可不传（默认当前工作台路径）。传入路径时返回该路径本身及所有子路径下的任务（例如在目录节点也能看到子目录/子表单上挂的定时任务）。可按 status 过滤。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "工作台路径前缀（可选，不传默认当前路径）。匹配该路径及其子路径下的定时任务，而非仅精确相等。",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "状态过滤（可选）：pending/done/failed/cancelled",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码（可选，默认 1）",
				},
				"page_size": map[string]interface{}{
					"type":        "integer",
					"description": "每页条数（可选，默认 20）",
				},
			},
			"required": []interface{}{},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "cancel_scheduled_task",
		Description: "取消定时任务（仅创建人可取消）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "integer",
					"description": "任务 ID",
				},
			},
			"required": []interface{}{"task_id"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "list_scheduled_task_executions",
		Description: "查询某个定时任务的执行记录。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_id": map[string]interface{}{
					"type":        "integer",
					"description": "任务 ID",
				},
				"status": map[string]interface{}{
					"type":        "string",
					"description": "状态过滤（可选）：success/failed",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码（可选，默认 1）",
				},
				"page_size": map[string]interface{}{
					"type":        "integer",
					"description": "每页条数（可选，默认 20）",
				},
			},
			"required": []interface{}{"task_id"},
		},
	})
	// run_official_python：调用系统空间官方 Form「Python 执行」；预装库见 runOfficialPythonPreinstallDoc（与 build/Dockerfile 同步）
	out = append(out, dto.ToolDef{
		Name: "run_official_python",
		Description: runOfficialPythonPreinstallDoc + `

**执行环境：** Python 跑在 **应用运行时容器内**（Podman 等业务容器，**不是宿主机**）。本工具调用官方路径 **/system/official/python/execute**，由 **官方应用** 对应容器执行；脚本在 **临时目录** 中运行，不把工作区源码树当作工作目录。

**无法输出文件到工作台供用户下载：** 本工具只能返回文本/JSON（output/json_result），**不能**把 Python 生成的 PNG/Excel 等变成工作台可下载附件。

**若需要「处理后的文件给用户下载」：** 请先用 **read_doc** 读取内置示例文档 **/builtin/doc/case_catalog/form/python_output**（含 PRD 与完整 Go 示例），再按文档配合 **agent-app SDK** 在用户应用内新增 Form：**pythonRuntime.NewExecutor** → **defer executor.Close()**（默认临时目录）→ Go 用 **filepath.Abs** 得到 **绝对路径**（如 GetTraceOutputDir 下文件）经请求传给 Python → Python **直接写入该路径**（如 savefig，勿用相对路径互传，Go/Python **cwd 不同**）→ 响应 **types.Files**（ResponseFiles 使用同一绝对路径）。Go 与 Python 为**同机子进程**，非网络隔离。

**两种输出方式（二选一或组合）：**
1. **结构化结果（推荐）**：脚本末尾调用 output_json(字典或列表)，键用双引号、值为 JSON 可序列化类型。返回里 json_result 为格式化后的 JSON，便于你后续取字段。
2. **纯文本/报表**：用 print(...)。返回里以 output 为准；json_result 会提示「非 JSON」，属正常降级，不要误判为失败。

**如何读返回：** status=成功 时，有结构化数据优先看 json_result；无则读 output。若 json_result 含「JSON解析失败」而 output 里已有 <python-out> 片段，以 output 内 JSON 为准或修正脚本后重试。

**参数：** args_json 为 JSON 对象字符串，字段注入脚本全局命名空间。timeout_seconds 默认 120、上限 300。

返回中可能含 _model_guidance：面向你的纠错/降级说明，请优先阅读。`,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"python_code": map[string]interface{}{
					"type":        "string",
					"description": "完整 Python 源码。需要机器可读结果用 output_json；仅需文本用 print。若要把生成文件给用户下载，本工具做不到，须 read_doc /builtin/doc/case_catalog/form/python_output 后按 SDK 写 Form（绝对路径落盘 + defer Close，勿依赖 base64）。",
				},
				"args_json": map[string]interface{}{
					"type":        "string",
					"description": "可选。JSON 对象字符串，字段注入全局（如 {\"rows\":[],\"name\":\"x\"}）。",
				},
				"timeout_seconds": map[string]interface{}{
					"type":        "integer",
					"description": "可选，超时秒数，默认 120，最大 300",
				},
			},
			"required": []interface{}{"python_code"},
		},
	})

	// run_on_select_fuzzy：执行工作区内 OnSelectFuzzy 回调，用于测试带「下拉模糊搜索」的表单/表格（仅支持按关键词或空关键词，不支持 by_value/by_values）
	out = append(out, dto.ToolDef{
		Name:        "run_on_select_fuzzy",
		Description: "执行工作区内 OnSelectFuzzy 回调，用于测试带「下拉模糊搜索/回调查询」的 Form 或 Table。**仅支持按关键词搜索**：type 固定为 by_keyword，value 为关键词字符串（可为空表示空搜索）。不支持 by_value、by_values。full_code_path 为配置了该回调的 Form 或 Table 的完整路径（如 .../cashier_desk.form）；code 为字段 code（如 product_id、member_id）；request 可选，为当前表单的 JSON（用于依赖其他字段时）。返回 items（选项列表）及可选 statistics、error_msg。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "配置了 OnSelectFuzzyMap 的 Form 或 Table 的完整路径，如 /luobei/myapp/plugins/cashier_desk.form",
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "触发回调的字段 code，如 product_id、member_id（需与该 Form/Table 的 OnSelectFuzzyMap 键一致）",
				},
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词（可选）。不传或传空字符串表示空搜索，会返回默认/全部选项",
				},
				"request": map[string]interface{}{
					"type":        "string",
					"description": "可选。当前表单/行的 JSON 字符串，用于依赖其他字段的回调（如根据上级选择过滤下级选项）",
				},
			},
			"required": []interface{}{"full_code_path", "code"},
		},
	})

	// publish_to_hub：首次将当前工作区目录或指定目录发布到应用市场（Hub）
	out = append(out, dto.ToolDef{
		Name:        "publish_to_hub",
		Description: "将当前工作区目录或指定目录首次发布到应用市场（Hub）。必填：name（在应用市场上的目录名称）。可选：directory（不传则使用当前工作目录）、description、category、tags（逗号分隔，支持自定义标签）、service_fee_personal（个人用户服务费，元）、service_fee_enterprise（企业用户服务费，元）。发布成功后返回 hub 目录路径与文件统计。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "要发布的目录（可选），不传则使用当前工作目录，如 /user/app 或 /user/app/plugins/pdf",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "在应用市场上的目录名称，如「视频处理插件」",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "目录描述（可选）",
				},
				"category": map[string]interface{}{
					"type":        "string",
					"description": "分类（可选），如 表单、表格、图表",
				},
				"tags": map[string]interface{}{
					"type":        "string",
					"description": "标签，逗号分隔（可选），支持自定义标签，如 video,media,流媒体",
				},
				"service_fee_personal": map[string]interface{}{
					"type":        "number",
					"description": "个人用户服务费（可选），单位：元，如 0 或 99.9",
				},
				"service_fee_enterprise": map[string]interface{}{
					"type":        "number",
					"description": "企业用户服务费（可选），单位：元，如 0 或 199",
				},
			},
			"required": []interface{}{"name"},
		},
	})

	// push_to_hub：更新已发布到应用市场的目录（类似 git push，递增版本）
	out = append(out, dto.ToolDef{
		Name:        "push_to_hub",
		Description: "将已发布到应用市场的目录推送更新（版本号由后端自动递增）。可选：directory（不传则当前工作目录）、update_description（本版本更新说明）、service_fee_personal（个人用户服务费，元）、service_fee_enterprise（企业用户服务费，元）。若目录尚未发布过，需先使用 publish_to_hub。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "要推送的目录（可选），不传则使用当前工作目录",
				},
				"update_description": map[string]interface{}{
					"type":        "string",
					"description": "本版本更新说明（可选），如「新增 xxx 功能」",
				},
				"service_fee_personal": map[string]interface{}{
					"type":        "number",
					"description": "个人用户服务费（可选），单位：元",
				},
				"service_fee_enterprise": map[string]interface{}{
					"type":        "number",
					"description": "企业用户服务费（可选），单位：元",
				},
			},
			"required": []interface{}{},
		},
	})

	// search_hub_directory：搜索应用中心（Hub）目录，搜到合适的可用 copy_directory 复制到本地
	out = append(out, dto.ToolDef{
		Name:        "search_hub_directory",
		Description: "在应用中心（Hub）搜索应用，或按路径查询单个目录在 Hub 上的信息。① 按关键词搜索：传 search（可选，不传或传空则返回全部应用）；支持多关键字「或」搜索，用 | 分隔，例如：美发|理发|美容|预约，表示匹配其中任意一词即可；可传 page、page_size（可选）。② 按路径查当前目录在 Hub 上的信息：传 full_code_path（如 /user/app/plugins/xxx），可查看该路径是否已上架、copy_url、star_count 等。返回含 copy_url（用于 copy_directory）、star_count、download_count 等。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "目录完整路径（可选），传入则查询该路径在应用中心的信息（是否已上架、复制链接、星数等），如 /luobei/demos/plugins/videos",
				},
				"search": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词（可选），不传或传空则返回全部应用。支持多关键字「或」搜索，用 | 分隔，例如：美发|理发|美容|预约，匹配名称、描述、标签中任意一词即命中。与 full_code_path 二选一使用。",
				},
				"page": map[string]interface{}{
					"type":        "integer",
					"description": "页码（可选，默认 1）",
				},
				"page_size": map[string]interface{}{
					"type":        "integer",
					"description": "每页条数（可选，默认 10，最大建议 20）",
				},
			},
			"required": []interface{}{},
		},
	})

	// copy_directory：通用复制目录（源可为 Hub 链接或本地路径）。target_directory 为「目标父目录」，系统会在其下自动创建与源同名的子目录。
	out = append(out, dto.ToolDef{
		Name:        "copy_directory",
		Description: "将目录复制到工作区。源：source_directory 为 Hub 链接（hub://host/path@version，来自 search_hub_directory 的 copy_url）或本地完整路径（如 /user/app/plugins/xxx）。目标：target_directory 填「目标父目录」即当前工作区路径（如 /luobei/myapp/server），不要填「父目录+子目录名」；系统会在该父目录下自动创建与源同名的子目录（如源为 .../video_tools 则得到 .../server/video_tools）。复制成功后会自动编译，无需再调用 build_workspace；返回目录数、文件数。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"source_directory": map[string]interface{}{
					"type":        "string",
					"description": "源目录：Hub 链接（hub://host/full_code_path@version）或本地完整路径（如 /luobei/app_a/plugins/pdf）",
				},
				"target_directory": map[string]interface{}{
					"type":        "string",
					"description": "目标父目录（当前工作区路径），如 /luobei/myapp/server。系统会在此路径下创建与源同名的子目录，不要传 /luobei/myapp/server/子目录名",
				},
			},
			"required": []interface{}{"source_directory", "target_directory"},
		},
	})

	// record_workspace_event：工作台埋点，记录无法实现的需求、不明确需求等，供产品分析
	out = append(out, dto.ToolDef{
		Name:        "record_workspace_event",
		Description: "记录工作台内事件，用于产品分析与改进。当判断需求无法实现或需求不明确时，在回复用户前调用。event_type 必填：unsupported_demand（平台无法实现）、unclear_requirement（需求不明确需澄清）、task_failed（执行失败）等；description 必填（一句话说明）；context、extra 可选。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"event_type": map[string]interface{}{
					"type":        "string",
					"description": "事件类型：unsupported_demand（平台无法实现）、unclear_requirement（需求不明确）、task_failed（执行失败）等",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "一句话描述，如「用户需要实时 WebSocket 推送，当前平台不支持」",
				},
				"context": map[string]interface{}{
					"type":        "string",
					"description": "可选，上下文摘要（如当前目录、用户原话摘要）",
				},
				"extra": map[string]interface{}{
					"type":        "string",
					"description": "可选，额外 JSON 字符串，供后续扩展",
				},
			},
			"required": []interface{}{"event_type", "description"},
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
	case "read_app_log":
		return r.callReadAppLog(ctx, args, fullCodePath)
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
	case "create_scheduled_task":
		return r.callCreateScheduledTask(ctx, args, fullCodePath)
	case "list_scheduled_tasks":
		return r.callListScheduledTasks(ctx, args, fullCodePath)
	case "cancel_scheduled_task":
		return r.callCancelScheduledTask(ctx, args)
	case "list_scheduled_task_executions":
		return r.callListScheduledTaskExecutions(ctx, args)
	case "run_official_python":
		return r.callRunOfficialPython(ctx, args)
	case "run_on_select_fuzzy":
		return r.callRunOnSelectFuzzy(ctx, args, fullCodePath)
	case "web_search":
		return r.callWebSearch(ctx, args)
	case "fetch_url_content":
		return r.callFetchURLContent(ctx, args)
	case "search_tools":
		return r.callSearchTools(ctx, args, fullCodePath)
	case "publish_to_hub":
		return r.callPublishToHub(ctx, args, fullCodePath)
	case "push_to_hub":
		return r.callPushToHub(ctx, args, fullCodePath)
	case "search_hub_directory":
		return r.callSearchHub(ctx, args)
	case "copy_directory":
		return r.callCopyDirectory(ctx, args)
	case "record_workspace_event":
		return r.callRecordWorkspaceEvent(ctx, args, fullCodePath)
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

	// 降级处理：检测函数路径（路径末尾含 .table/.form/.chart/.docs 等后缀），自动取父目录读取
	degraded := false
	originalPath := targetPath
	if isFunctionPath(targetPath) {
		if parentPath := getParentPath(targetPath); parentPath != "" {
			targetPath = parentPath
			degraded = true
		}
	}

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

	// 构建降级提示前缀
	degradeNotice := ""
	if degraded {
		degradeNotice = fmt.Sprintf("> 注意：`%s` 是一个函数节点（非目录），已自动读取其所在的父目录 `%s`。\n\n", originalPath, targetPath)
	}

	// 树形格式：recursive=true 时整棵树，recursive=false 时只展开当前一层（max_depth=1）
	if outputFormat == "tree" {
		treeMaxDepth := maxDepth
		if !recursive {
			treeMaxDepth = 1
		}
		result, hasErr := r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, treeMaxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
		return degradeNotice + result, hasErr
	}
	if recursive {
		result, hasErr := r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
		return degradeNotice + result, hasErr
	}

	// 列表格式显示当前目录
	result, hasErr := r.buildListFormat(ctx, workspaceCtx, targetPath, includeFunctions, includeFiles, includeCode, outputFormat)
	return degradeNotice + result, hasErr
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
	// 永远不编译：避免 write_go_file 触发 go mod tidy / go build 导致失败影响落盘结果
	buildWorkspace := false

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
	returnFullContent := false
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

// callReadAppLog 读取应用日志（支持 version、关键词检索）
func (r *ToolRegistry) callReadAppLog(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	req := &dto.ReadAppLogReq{
		FullCodePath: targetPath,
		Version:      strings.TrimSpace(GetStringArg(args, "version")),
		Keyword:      GetStringArg(args, "keyword"),
		ContextLines: 0,
		MaxMatches:   0,
		IgnoreCase:   false,
	}
	if v, ok := args["lines"]; ok {
		if n, ok := toInt(v); ok {
			req.Lines = n
		}
	}
	if v, ok := args["context_lines"]; ok {
		if n, ok := toInt(v); ok {
			req.ContextLines = n
		}
	}
	if v, ok := args["max_matches"]; ok {
		if n, ok := toInt(v); ok {
			req.MaxMatches = n
		}
	}
	if v, ok := args["ignore_case"]; ok {
		if b, ok := v.(bool); ok {
			req.IgnoreCase = b
		}
	}
	resp, err := apicall.ReadAppLog(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[ReadAppLog] ReadAppLog 失败: %v", err)
		return "read_app_log 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "read_app_log: " + resp.Message, true
	}
	msg := fmt.Sprintf("日志读取成功：版本=%s，文件=%s，总行数=%d，返回行数=%d，命中数=%d，截断=%t",
		resp.ResolvedVersion, resp.LogFile, resp.TotalLines, resp.ReturnedLines, resp.MatchCount, resp.Truncated)
	if resp.Content != "" {
		msg += "\n\n" + resp.Content
	}
	return msg, false
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

// splitSearchKeywords 将 keyword 按竖线 | 拆成多个关键词并去空，如 "视频|video|流媒体" -> ["视频","video","流媒体"]
func splitSearchKeywords(keyword string) []string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil
	}
	parts := strings.Split(keyword, "|")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// callSearchTools 按关键词搜索可用工具（内置工具 + system 用户下已注册 Form/Table/Chart）。keyword 为空时仅返回已注册函数并按调用次数降序（高频在前）
// callWebSearch 调用 pkg/websearch（默认百度、可回退必应），返回格式化文本供模型使用
func (r *ToolRegistry) callWebSearch(ctx context.Context, args map[string]interface{}) (string, bool) {
	keyword := strings.TrimSpace(GetStringArg(args, "keyword"))
	if keyword == "" {
		return "web_search 必填 keyword（搜索关键词）。", true
	}
	limit := 10
	if v, ok := args["limit"]; ok {
		if n, ok := toInt(v); ok && n > 0 {
			if n > 20 {
				n = 20
			}
			limit = n
		}
	}
	results, err := websearch.Search(ctx, keyword, limit)
	if err != nil {
		logger.Warnf(ctx, "[web_search] Search 失败: %v", err)
		return "web_search 暂时不可用，请稍后再试。", false
	}
	if len(results) == 0 {
		return "未找到与「" + keyword + "」相关的搜索结果。可尝试更换关键词。", false
	}
	const maxSnippetLen = 300
	const maxBodyLen = 1500 // 单条正文给模型的最大长度，避免 token 爆炸
	var b strings.Builder
	b.WriteString("【网络搜索结果】关键词：「" + keyword + "」共 " + fmt.Sprintf("%d", len(results)) + " 条\n\n")
	for i, r := range results {
		b.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if r.URL != "" {
			b.WriteString("   链接: " + r.URL + "\n")
		}
		if r.Snippet != "" {
			snippet := r.Snippet
			if len(snippet) > maxSnippetLen {
				snippet = snippet[:maxSnippetLen] + "..."
			}
			b.WriteString("   摘要: " + snippet + "\n")
		}
		if r.Body != "" {
			body := r.Body
			if len(body) > maxBodyLen {
				body = body[:maxBodyLen] + "..."
			}
			b.WriteString("   正文: " + body + "\n")
		}
		b.WriteString("\n")
	}
	return b.String(), false
}

const maxFetchURLContentCount = 5 // 一次最多拉取多少个链接

// callFetchURLContent 按 URL（或 urls 数组）拉取页面正文，支持多链接
func (r *ToolRegistry) callFetchURLContent(ctx context.Context, args map[string]interface{}) (string, bool) {
	var urlList []string
	if urlsRaw, ok := args["urls"]; ok && urlsRaw != nil {
		if arr, ok := urlsRaw.([]interface{}); ok && len(arr) > 0 {
			for _, v := range arr {
				if s, ok := v.(string); ok {
					s = strings.TrimSpace(s)
					if s != "" {
						urlList = append(urlList, s)
					}
				}
			}
			if len(urlList) > maxFetchURLContentCount {
				urlList = urlList[:maxFetchURLContentCount]
			}
		}
	}
	if len(urlList) == 0 {
		rawURL := strings.TrimSpace(GetStringArg(args, "url"))
		if rawURL == "" {
			return "fetch_url_content 需填 url（单个）或 urls（多个）。", true
		}
		urlList = []string{rawURL}
	}
	for i, u := range urlList {
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			urlList[i] = "https://" + u
		}
	}

	maxChars := 3000
	if v, ok := args["max_chars"]; ok {
		if n, ok := toInt(v); ok && n > 0 {
			if n > 20000 {
				n = 20000
			}
			maxChars = n
		}
	}

	var b strings.Builder
	for i, rawURL := range urlList {
		title, body := websearch.FetchURLContent(ctx, rawURL, maxChars)
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(fmt.Sprintf("【第 %d 个链接】%s\n", i+1, rawURL))
		if title != "" {
			b.WriteString("标题: " + title + "\n\n")
		}
		if body != "" {
			b.WriteString("正文: " + body)
		} else {
			b.WriteString("（请求失败、无响应体或无法建立连接）")
		}
	}
	if b.Len() == 0 {
		return "所有链接均无法拉取内容。", false
	}
	return b.String(), false
}

func (r *ToolRegistry) callSearchTools(ctx context.Context, args map[string]interface{}, fullCodePath string) (string, bool) {
	keywordRaw := strings.TrimSpace(GetStringArg(args, "keyword"))
	keywords := splitSearchKeywords(keywordRaw)
	templateType := strings.TrimSpace(GetStringArg(args, "template_type"))
	limit := 20
	if v, ok := args["limit"]; ok {
		if n, ok := toInt(v); ok && n > 0 {
			if n > 50 {
				n = 50
			}
			limit = n
		}
	}
	var buf strings.Builder

	// 1. 内置工具：仅当有关键词时按 name/description 匹配；keyword 为空时不展示内置工具，只返回按调用次数的高频已注册函数
	if len(keywords) > 0 {
		allTools, _ := r.ListTools(ctx, nil)
		lowerKeywords := make([]string, len(keywords))
		for i, k := range keywords {
			lowerKeywords[i] = strings.ToLower(k)
		}
		var matchedTools []dto.ToolDef
		for _, t := range allTools {
			text := strings.ToLower(t.Name + " " + t.Description)
			for _, k := range lowerKeywords {
				if strings.Contains(text, k) {
					matchedTools = append(matchedTools, t)
					break
				}
			}
		}
		if len(matchedTools) > 0 {
			buf.WriteString("【内置工具】\n")
			for _, t := range matchedTools {
				buf.WriteString("- ")
				buf.WriteString(t.Name)
				buf.WriteString("：")
				buf.WriteString(t.Description)
				buf.WriteString("\n")
			}
			buf.WriteString("\n")
		}
	}

	// 2. 已注册 Form/Table/Chart：system 用户下；keyword 为空时后端按调用次数降序返回高频函数
	resp, err := apicall.SearchFunctions(ctx, &dto.SearchFunctionsReq{
		User:         "system", // 限定为 system 用户下工作空间，不搜其他用户的应用
		App:          "",
		Keyword:      keywordRaw,
		TemplateType: templateType,
		Page:         1,
		PageSize:     limit,
	})
	functions := make([]*dto.FunctionSearchResult, 0)
	if err != nil {
		logger.Warnf(ctx, "[SearchTools] SearchFunctions err: %v", err)
	} else if resp != nil {
		functions = resp.Functions
	}
	if len(functions) > 0 {
		if keywordRaw == "" {
			buf.WriteString("【已注册函数】（按调用次数从高到低，仅 system 用户下）\n")
		} else {
			buf.WriteString("【已注册函数】（仅 system 用户下）调用方式：form → run_form_submit，table → run_table_search/run_table_create/run_table_update，chart → run_chart_query。\n")
		}
		for i, fn := range functions {
			buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, fn.Name))
			buf.WriteString("   full_code_path: ")
			buf.WriteString(fn.FullCodePath)
			buf.WriteString("\n")
			if fn.RunCount > 0 {
				buf.WriteString(fmt.Sprintf("   已使用 %d 次\n", fn.RunCount))
			}
			if fn.Description != "" {
				buf.WriteString("   description: ")
				buf.WriteString(fn.Description)
				buf.WriteString("\n")
			}
			if fn.TemplateType != "" {
				buf.WriteString("   type: ")
				buf.WriteString(fn.TemplateType)
				buf.WriteString("\n")
			}
			if len(fn.Request) > 0 {
				if reqJSON, err := json.MarshalIndent(fn.Request, "   ", "  "); err == nil {
					buf.WriteString("   request: ")
					buf.Write(reqJSON)
					buf.WriteString("\n")
				}
			}
		}
	} else if buf.Len() == 0 {
		if keywordRaw == "" {
			buf.WriteString("当前 system 用户下暂无已注册函数；可传 keyword 按关键词搜索，或使用 search_hub_directory 搜应用市场。")
		} else {
			buf.WriteString("未匹配到任何可用工具（内置工具或 system 用户下已注册函数），可考虑 search_hub_directory 搜应用市场，或创建新目录并按「创建项目」流程（先 PRD、用户确认后再写代码）。")
		}
	}
	return buf.String(), false
}

// parseHubSourceFromPath 从目录路径解析 source_user、source_app、source_directory_path。路径格式：/user/app 或 /user/app/xxx
func parseHubSourceFromPath(dirPath string) (sourceUser, sourceApp, sourceDirectoryPath string, errMsg string) {
	dirPath = strings.TrimSpace(dirPath)
	if dirPath == "" {
		return "", "", "", "目录路径不能为空"
	}
	if !strings.HasPrefix(dirPath, "/") {
		dirPath = "/" + dirPath
	}
	trimmed := strings.TrimPrefix(dirPath, "/")
	if trimmed == "" {
		return "", "", "", "目录路径至少需要 user/app 两段，如 /user/app"
	}
	parts := strings.SplitN(trimmed, "/", 3)
	if len(parts) < 2 {
		return "", "", "", "目录路径至少需要 user/app 两段，如 /user/app"
	}
	sourceUser = strings.TrimSpace(parts[0])
	sourceApp = strings.TrimSpace(parts[1])
	if sourceUser == "" || sourceApp == "" {
		return "", "", "", "目录路径中 user 和 app 不能为空"
	}
	sourceDirectoryPath = "/" + trimmed
	return sourceUser, sourceApp, sourceDirectoryPath, ""
}

// callPublishToHub 首次将目录发布到应用市场（Hub）
func (r *ToolRegistry) callPublishToHub(ctx context.Context, args map[string]interface{}, fullCodePath string) (string, bool) {
	dirPath := strings.TrimSpace(GetStringArg(args, "directory"))
	if dirPath == "" {
		dirPath = fullCodePath
	}
	sourceUser, sourceApp, sourceDirectoryPath, errMsg := parseHubSourceFromPath(dirPath)
	if errMsg != "" {
		return "publish_to_hub: " + errMsg, true
	}
	name := strings.TrimSpace(GetStringArg(args, "name"))
	if name == "" {
		return "publish_to_hub 必填 name（在应用市场上的目录名称）。", true
	}
	req := &dto.PublishDirectoryToHubReq{
		SourceUser:          sourceUser,
		SourceApp:           sourceApp,
		SourceDirectoryPath: sourceDirectoryPath,
		Name:                name,
		Description:         strings.TrimSpace(GetStringArg(args, "description")),
		Category:            strings.TrimSpace(GetStringArg(args, "category")),
	}
	if tagsStr := strings.TrimSpace(GetStringArg(args, "tags")); tagsStr != "" {
		for _, t := range strings.Split(tagsStr, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				req.Tags = append(req.Tags, t)
			}
		}
	}
	if v, ok := args["service_fee_personal"]; ok && v != nil {
		if f, ok := toFloat64(v); ok && f >= 0 {
			req.ServiceFeePersonal = f
		}
	}
	if v, ok := args["service_fee_enterprise"]; ok && v != nil {
		if f, ok := toFloat64(v); ok && f >= 0 {
			req.ServiceFeeEnterprise = f
		}
	}
	resp, err := apicall.PublishDirectoryToHubViaWorkspace(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[PublishToHub] 失败: %v", err)
		return "publish_to_hub 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("发布成功。Hub 目录路径: %s，子目录数: %d，文件数: %d。",
		resp.HubFullCodePath, resp.DirectoryCount, resp.FileCount), false
}

// callPushToHub 将已发布的目录推送到 Hub（更新版本）
func (r *ToolRegistry) callPushToHub(ctx context.Context, args map[string]interface{}, fullCodePath string) (string, bool) {
	dirPath := strings.TrimSpace(GetStringArg(args, "directory"))
	if dirPath == "" {
		dirPath = fullCodePath
	}
	sourceUser, sourceApp, sourceDirectoryPath, errMsg := parseHubSourceFromPath(dirPath)
	if errMsg != "" {
		return "push_to_hub: " + errMsg, true
	}
	req := &dto.PushDirectoryToHubReq{
		SourceUser:          sourceUser,
		SourceApp:           sourceApp,
		SourceDirectoryPath: sourceDirectoryPath,
		UpdateDescription:   strings.TrimSpace(GetStringArg(args, "update_description")),
		// Version 由后端自动递增，不传
	}
	if v, ok := args["service_fee_personal"]; ok && v != nil {
		if f, ok := toFloat64(v); ok && f >= 0 {
			req.ServiceFeePersonal = f
		}
	}
	if v, ok := args["service_fee_enterprise"]; ok && v != nil {
		if f, ok := toFloat64(v); ok && f >= 0 {
			req.ServiceFeeEnterprise = f
		}
	}
	resp, err := apicall.PushDirectoryToHubViaWorkspace(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[PushToHub] 失败: %v", err)
		return "push_to_hub 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("推送成功。Hub 目录路径: %s，版本: %s -> %s，子目录数: %d，文件数: %d。",
		resp.HubFullCodePath, resp.OldVersion, resp.NewVersion, resp.DirectoryCount, resp.FileCount), false
}

// callSearchHub 在应用中心（Hub）搜索应用，或按 full_code_path 查询单个目录在 Hub 上的信息
func (r *ToolRegistry) callSearchHub(ctx context.Context, args map[string]interface{}) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath != "" {
		if !strings.HasPrefix(fullCodePath, "/") {
			fullCodePath = "/" + fullCodePath
		}
		detail, err := apicall.GetHubDirectoryDetail(ctx, &dto.GetHubDirectoryDetailReq{
			FullCodePath: fullCodePath,
			IncludeTree:  false,
		})
		if err != nil {
			return fmt.Sprintf("该路径在应用中心未找到或未上架：%s。可先用 publish_to_hub 发布后再查询。", fullCodePath), false
		}
		if detail == nil {
			return fmt.Sprintf("路径 %s 在应用中心暂无信息（可能未上架）。", fullCodePath), false
		}
		var b strings.Builder
		b.WriteString(fmt.Sprintf("应用中心 - 路径 %s 的信息：\n\n", detail.FullCodePath))
		b.WriteString(fmt.Sprintf("名称: %s\n", detail.Name))
		if detail.Description != "" {
			desc := detail.Description
			if len(desc) > 300 {
				desc = desc[:300] + "..."
			}
			b.WriteString("描述: " + desc + "\n")
		}
		b.WriteString(fmt.Sprintf("路径: %s | 发布者: %s | 版本: %s\n", detail.FullCodePath, detail.PublisherUsername, detail.Version))
		b.WriteString(fmt.Sprintf("星 %d | 克隆 %d 次 | 目录 %d / 文件 %d / 函数 %d\n", detail.StarCount, detail.DownloadCount, detail.DirectoryCount, detail.FileCount, detail.FunctionCount))
		if detail.CopyURL != "" {
			b.WriteString("复制链接（用于 copy_directory）: " + detail.CopyURL + "\n")
			b.WriteString("复制时 target_directory 填当前工作区路径（目标父目录），不要填「父路径/子目录名」。\n")
		}
		return b.String(), false
	}

	req := &dto.GetHubDirectoryListReq{
		Page:     1,
		PageSize: 10,
	}
	if v, ok := args["search"]; ok && v != nil {
		if s, ok := v.(string); ok && s != "" {
			req.Search = strings.TrimSpace(s)
		}
	}
	if v, ok := args["page"]; ok && v != nil {
		if n, ok := toInt(v); ok && n > 0 {
			req.Page = n
		}
	}
	if v, ok := args["page_size"]; ok && v != nil {
		if n, ok := toInt(v); ok && n > 0 {
			if n > 50 {
				n = 50
			}
			req.PageSize = n
		}
	}
	resp, err := apicall.GetHubDirectoryList(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[SearchHub] GetHubDirectoryList 失败: %v", err)
		return "search_hub_directory 调用失败: " + err.Error(), true
	}
	if len(resp.Items) == 0 {
		return fmt.Sprintf("应用中心共 %d 条结果，当前页无数据。可调整 search（支持多关键字用 | 分隔）或 page 再试。", resp.Total), false
	}
	var b strings.Builder
	b.WriteString(fmt.Sprintf("应用中心搜索结果（共 %d 条，当前第 %d 页）：\n\n", resp.Total, resp.Page))
	for i, item := range resp.Items {
		b.WriteString(fmt.Sprintf("【%d】%s\n", i+1, item.Name))
		if item.Description != "" {
			desc := item.Description
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			b.WriteString("  描述: " + desc + "\n")
		}
		b.WriteString(fmt.Sprintf("  路径: %s | 发布者: %s | 星 %d | 克隆 %d 次\n", item.FullCodePath, item.PublisherUsername, item.StarCount, item.DownloadCount))
		if item.CopyURL != "" {
			b.WriteString("  复制链接（用于 copy_directory）: " + item.CopyURL + "\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("使用 copy_directory(source_directory=\"上面的复制链接\", target_directory=\"/你的用户/你的应用/当前目录\") 即可将应用复制到本地；target_directory 填当前工作区路径（目标父目录），会在其下自动创建与源同名的子目录，不要填「父目录/子目录名」。")
	return b.String(), false
}

// callCopyDirectory 通用复制目录：源可为 Hub 链接（hub://）或本地路径（/user/app/...），后端 CopyServiceTree 统一处理
func (r *ToolRegistry) callCopyDirectory(ctx context.Context, args map[string]interface{}) (string, bool) {
	sourcePath := strings.TrimSpace(GetStringArg(args, "source_directory"))
	if sourcePath == "" {
		return "copy_directory 必填 source_directory（Hub 链接 hub://host/path@version 或本地完整路径如 /user/app/plugins/xxx）。", true
	}
	if !strings.HasPrefix(sourcePath, "hub://") && !strings.HasPrefix(sourcePath, "/") {
		sourcePath = "/" + sourcePath
	}
	targetPath := strings.TrimSpace(GetStringArg(args, "target_directory"))
	if targetPath == "" {
		return "copy_directory 必填 target_directory（目标父目录，即当前工作区路径，如 /user/app/server；会在其下创建与源同名的子目录，不要填 .../子目录名）。", true
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	// 解析 target 得到 app 所在路径，用于获取 target_app_id（目标路径或其父路径必须在工作区存在）
	pathForDetail := targetPath
	for {
		detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, pathForDetail)
		if err == nil && detail != nil && detail.AppID > 0 {
			req := &dto.CopyDirectoryReq{
				SourceDirectoryPath: sourcePath,
				TargetDirectoryPath: targetPath,
				TargetAppID:         detail.AppID,
			}
			resp, err := apicall.CopyDirectoryViaWorkspace(ctx, req)
			if err != nil {
				logger.Errorf(ctx, "[CopyDirectory] CopyDirectory 失败: %v", err)
				return "copy_directory 复制失败: " + err.Error(), true
			}
			return fmt.Sprintf("复制成功。%s（目录数: %d，文件数: %d）", resp.Message, resp.DirectoryCount, resp.FileCount), false
		}
		// 尝试父路径
		idx := strings.LastIndex(strings.Trim(pathForDetail, "/"), "/")
		if idx <= 0 {
			break
		}
		pathForDetail = "/" + strings.Trim(pathForDetail, "/")[:idx]
		if pathForDetail == "" || pathForDetail == "/" {
			break
		}
	}
	return "copy_directory: 无法解析目标应用（target_directory 为目标父目录，须为工作区已存在路径，如 /user/app/server；不要填 .../子目录名）。", true
}

// callRecordWorkspaceEvent 工作台埋点：记录无法实现的需求、不明确需求等，落库并带 session_id 便于追溯
func (r *ToolRegistry) callRecordWorkspaceEvent(ctx context.Context, args map[string]interface{}, fullCodePath string) (string, bool) {
	eventType := strings.TrimSpace(GetStringArg(args, "event_type"))
	description := strings.TrimSpace(GetStringArg(args, "description"))
	if eventType == "" || description == "" {
		return "record_workspace_event 必填 event_type 和 description。", true
	}
	contextStr := GetStringArg(args, "context")
	extra := GetStringArg(args, "extra")
	sessionID := getWorkspaceSessionID(ctx)
	user := contextx.GetRequestUser(ctx)

	e := &model.WorkspaceEvent{
		SessionID:    sessionID,
		FullCodePath: fullCodePath,
		User:         user,
		EventType:    eventType,
		Description:  description,
		Context:      contextStr,
		Extra:        extra,
	}
	if r.eventRepo != nil {
		if err := r.eventRepo.Create(ctx, e); err != nil {
			logger.Warnf(ctx, "[workspace_event] 落库失败: %v", err)
			// 不向用户报错，仅打日志
		}
	}
	logger.Infof(ctx, "[workspace_event] event_type=%s session_id=%s full_code_path=%s description=%s",
		eventType, sessionID, fullCodePath, description)
	return "已记录。", false
}

// parseAppFromFullCodePath 从 full_code_path 解析 app（第二段），如 /luobei/demos/xxx -> demos
func parseAppFromFullCodePath(fullCodePath string) string {
	fullCodePath = strings.TrimPrefix(strings.TrimSpace(fullCodePath), "/")
	if fullCodePath == "" {
		return ""
	}
	parts := strings.SplitN(fullCodePath, "/", 3)
	if len(parts) < 2 {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

// callRunFormSubmit 执行 Form 提交（执行模式专用）。body 完全由大模型根据用户消息中的 <files> 及表单参数定义自行拼装（各表单的 files 参数字段名不同，如 input_files、csv_file、logo 等）。
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

// callCreateScheduledTask 创建定时任务（工作台工具）
func (r *ToolRegistry) callCreateScheduledTask(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	name := strings.TrimSpace(GetStringArg(args, "name"))
	if name == "" {
		return "create_scheduled_task 需传 name（任务名称）。", true
	}
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath == "" {
		return "create_scheduled_task 需传 full_code_path，或在可推导当前目录的上下文中调用。", true
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}

	scheduleType := strings.TrimSpace(GetStringArg(args, "schedule_type"))
	if scheduleType == "" {
		return "create_scheduled_task 需传 schedule_type（atime/cron/every）。", true
	}
	runAt := strings.TrimSpace(GetStringArg(args, "run_at"))
	if runAt == "" {
		return "create_scheduled_task 需传 run_at（本地时间如 2006-01-02 15:04:05，或带偏移的 RFC3339）。", true
	}

	action := strings.TrimSpace(GetStringArg(args, "action"))
	if action == "" {
		action = "execute"
	}
	if strings.EqualFold(action, "form") {
		action = "execute"
	}
	method := strings.ToUpper(strings.TrimSpace(GetStringArg(args, "method")))
	if method == "" {
		method = "POST"
	}

	payloadStr := strings.TrimSpace(GetStringArg(args, "payload"))
	if payloadStr == "" {
		payloadStr = "{}"
	}
	var payloadRaw json.RawMessage
	if err := json.Unmarshal([]byte(payloadStr), &payloadRaw); err != nil {
		return "create_scheduled_task 的 payload 必须是合法 JSON 对象字符串: " + err.Error(), true
	}

	req := &dto.CreateScheduledTaskReq{
		Name:         name,
		FullCodePath: fullCodePath,
		Action:       action,
		Method:       method,
		Payload:      payloadRaw,
		ScheduleType: scheduleType,
		RunAt:        runAt,
		CronExpr:     strings.TrimSpace(GetStringArg(args, "cron_expr")),
		Timezone:     strings.TrimSpace(GetStringArg(args, "timezone")),
	}

	if v, ok := toInt(args["interval_seconds"]); ok {
		req.IntervalSeconds = int64(v)
	}
	if v, ok := toInt(args["max_runs"]); ok {
		req.MaxRuns = v
	}

	item, err := apicall.CreateScheduledTask(ctx, req)
	if err != nil {
		return "create_scheduled_task 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"id":             item.ID,
		"name":           item.Name,
		"full_code_path": item.FullCodePath,
		"action":         item.Action,
		"schedule_type":  item.ScheduleType,
		"status":         item.Status,
		"run_at":         item.RunAt,
		"next_run_at":    item.NextRunAt,
	}
	return formatJSONResult(out)
}

// callListScheduledTasks 查询定时任务列表（工作台工具）
func (r *ToolRegistry) callListScheduledTasks(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	status := strings.TrimSpace(GetStringArg(args, "status"))
	page := 1
	if v, ok := toInt(args["page"]); ok && v > 0 {
		page = v
	}
	pageSize := 20
	if v, ok := toInt(args["page_size"]); ok && v > 0 {
		pageSize = v
	}

	resp, err := apicall.ListScheduledTasks(ctx, fullCodePath, status, page, pageSize)
	if err != nil {
		return "list_scheduled_tasks 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"total": resp.Total,
		"list":  resp.List,
	}
	return formatJSONResult(out)
}

// callCancelScheduledTask 取消定时任务（工作台工具）
func (r *ToolRegistry) callCancelScheduledTask(ctx context.Context, args map[string]interface{}) (string, bool) {
	taskID, ok := toInt64(args["task_id"])
	if !ok || taskID <= 0 {
		return "cancel_scheduled_task 需传 task_id（正整数）。", true
	}
	if err := apicall.CancelScheduledTask(ctx, taskID); err != nil {
		return "cancel_scheduled_task 调用失败: " + err.Error(), true
	}
	return "已取消定时任务。", false
}

// officialPythonFormPayload 从 FormSubmit 返回体中取出 Python 执行结果（兼容外层包一层 data/result）
func officialPythonFormPayload(m map[string]interface{}) map[string]interface{} {
	if m == nil {
		return nil
	}
	if _, ok := m["output"]; ok {
		return m
	}
	for _, key := range []string{"data", "result"} {
		inner, ok := m[key].(map[string]interface{})
		if !ok {
			continue
		}
		if _, ok2 := inner["output"]; ok2 {
			return inner
		}
	}
	return m
}

func officialPythonAnyToString(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// buildOfficialPythonModelGuidance 根据 status/output/json_result 生成给大模型的友好提示与降级说明
func buildOfficialPythonModelGuidance(raw map[string]interface{}) string {
	p := officialPythonFormPayload(raw)
	if p == nil {
		return ""
	}
	status := strings.TrimSpace(officialPythonAnyToString(p["status"]))
	out := officialPythonAnyToString(p["output"])
	jr := officialPythonAnyToString(p["json_result"])
	lowOut := strings.ToLower(out)

	var lines []string
	appendLine := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, ex := range lines {
			if ex == s {
				return
			}
		}
		lines = append(lines, s)
	}

	switch status {
	case "失败":
		appendLine("【状态为失败】请阅读 output 中的 traceback/错误信息，修正 python_code 后重试。")
		if strings.Contains(out, "ModuleNotFoundError") || strings.Contains(out, "No module named") {
			appendLine("【依赖】ModuleNotFoundError：请优先使用工具说明里已列出的预装库（pandas、numpy、jieba、requests、openpyxl、matplotlib…）或仅用标准库；若必须新库，请管理员更新 build/Dockerfile 或官方 requirements.txt 并重打镜像。")
		}
		if strings.Contains(out, "SyntaxError") || strings.Contains(out, "IndentationError") {
			appendLine("【语法】请检查引号、缩进、括号是否匹配；字符串内换行需用三引号或 \\n。")
		}
		if strings.Contains(lowOut, "timeout") || strings.Contains(out, "deadline exceeded") || strings.Contains(out, "context deadline") {
			appendLine("【超时】可适当增大 timeout_seconds（最大 300），或拆分计算、减少数据量。")
		}
	case "成功":
		if strings.Contains(jr, "JSON解析失败") {
			appendLine("【json_result 解析失败】执行已成功；结构化内容可能在 output 的 <python-out>...</python-out> 内。请以 output 为准，或改为 output_json(合法 dict/list)，避免在标记内输出非 JSON 文本。")
		}
		if strings.Contains(jr, "输出不是JSON格式") || strings.Contains(jr, "不是JSON格式") {
			appendLine("【降级·正常】当前为纯文本输出（print）。若用户只需要报告/说明，无需改代码；若你需要程序取字段，请让脚本改用 output_json({...})。")
		}
		if strings.Contains(jr, "标记内无 JSON") {
			appendLine("【output_json 空内容】请确保 output_json 传入非空 dict/list；若本意是纯文本请改用 print。")
		}
		if jr == "" && out != "" && !strings.Contains(out, "<python-out>") {
			appendLine("【提示】未使用 output_json 时 json_result 常为空，以 output 为准即可。")
		}
	default:
		if status != "" {
			appendLine("【状态】status=" + status + "：请结合 output、json_result 判断。")
		}
	}

	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

// officialPythonFormPath 系统官方 Python 执行 Form 的 full_code_path（与 namespace system/official/code/api/python 注册路由一致）
const officialPythonFormPath = "/system/official/python/execute"

// callRunOfficialPython 调用系统官方 Python 执行 Form（预装库见 runOfficialPythonPreinstallDoc）
func (r *ToolRegistry) callRunOfficialPython(ctx context.Context, args map[string]interface{}) (string, bool) {
	code := strings.TrimSpace(GetStringArg(args, "python_code"))
	if code == "" {
		return "run_official_python 需传 python_code。", true
	}
	body := map[string]interface{}{
		"python_code": code,
	}
	if argsJSON := strings.TrimSpace(GetStringArg(args, "args_json")); argsJSON != "" {
		body["args_json"] = argsJSON
	}
	timeoutSec := 120
	if v, ok := toInt(args["timeout_seconds"]); ok && v > 0 {
		timeoutSec = v
	}
	if timeoutSec > 300 {
		timeoutSec = 300
	}
	body["timeout_seconds"] = timeoutSec

	result, err := apicall.FormSubmit(ctx, officialPythonFormPath, body)
	if err != nil {
		logger.Errorf(ctx, "[RunOfficialPython] FormSubmit 失败: %v", err)
		return "run_official_python 调用失败: " + err.Error() + "\n\n【给模型】可检查 python_code 是否过长、args_json 是否为合法 JSON 对象字符串；网络或权限问题可稍后重试。", true
	}
	out := make(map[string]interface{}, len(result)+1)
	for k, v := range result {
		out[k] = v
	}
	if g := buildOfficialPythonModelGuidance(result); g != "" {
		out["_model_guidance"] = g
	}
	return formatJSONResult(out)
}

// callListScheduledTaskExecutions 查询任务执行记录（工作台工具）
func (r *ToolRegistry) callListScheduledTaskExecutions(ctx context.Context, args map[string]interface{}) (string, bool) {
	taskID, ok := toInt64(args["task_id"])
	if !ok || taskID <= 0 {
		return "list_scheduled_task_executions 需传 task_id（正整数）。", true
	}
	status := strings.TrimSpace(GetStringArg(args, "status"))
	page := 1
	if v, ok := toInt(args["page"]); ok && v > 0 {
		page = v
	}
	pageSize := 20
	if v, ok := toInt(args["page_size"]); ok && v > 0 {
		pageSize = v
	}

	resp, err := apicall.ListScheduledTaskExecutions(ctx, taskID, status, page, pageSize)
	if err != nil {
		return "list_scheduled_task_executions 调用失败: " + err.Error(), true
	}
	out := map[string]interface{}{
		"task_id": taskID,
		"total":   resp.Total,
		"list":    resp.List,
	}
	return formatJSONResult(out)
}

// callRunOnSelectFuzzy 执行 OnSelectFuzzy 回调（工作台测试用）；仅支持按关键词或空关键词，不支持 by_value/by_values
func (r *ToolRegistry) callRunOnSelectFuzzy(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if fullCodePath == "" {
		fullCodePath = currentFullCodePath
	}
	if fullCodePath != "" && !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}
	if fullCodePath == "" {
		return "run_on_select_fuzzy 需传 full_code_path（配置了 OnSelectFuzzy 的 Form/Table 路径，如 .../cashier_desk.form）。", true
	}
	code := strings.TrimSpace(GetStringArg(args, "code"))
	if code == "" {
		return "run_on_select_fuzzy 需传 code（字段 code，与 OnSelectFuzzyMap 的键一致）。", true
	}

	keyword := strings.TrimSpace(GetStringArg(args, "keyword"))
	body := map[string]interface{}{
		"code":  code,
		"type":  "by_keyword",
		"value": keyword,
	}
	if s := GetStringArg(args, "request"); s != "" {
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
		return "run_on_select_fuzzy 调用失败: " + err.Error(), true
	}
	return formatJSONResult(result)
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

// toInt64 从 interface{} 转 int64（支持 float64/int/int64）
func toInt64(v interface{}) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

// toFloat64 从 interface{} 解析为 float64（JSON 数字多为 float64）
func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
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

// isFunctionPath 判断路径是否为函数路径：路径最后一段含 "."（如 xxx.table、xxx.form、xxx.chart 等）即视为函数节点，目录节点的 code 不含点号
func isFunctionPath(path string) bool {
	path = strings.TrimSuffix(path, "/")
	lastSlash := strings.LastIndex(path, "/")
	lastSegment := path
	if lastSlash >= 0 {
		lastSegment = path[lastSlash+1:]
	}
	return strings.Contains(lastSegment, ".")
}

// getParentPath 返回路径的父目录路径；如果已是根级或无法提取则返回空字符串
func getParentPath(path string) string {
	path = strings.TrimSuffix(path, "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash <= 0 {
		return ""
	}
	return path[:lastSlash]
}
