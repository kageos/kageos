package prd

import "strings"

const (
	Kind          = "agent_app_prd"
	SchemaVersion = "prd.v2"

	MaxTableExamples = 3
	MaxChartExamples = 12

	ContractMarker = "{{WORKSPACE_PRD_CONTRACT}}"
)

var (
	topLevelKeys = []string{"kind", "schema_version", "project", "tables", "forms", "charts", "rules"}
	projectKeys  = []string{"name", "code", "summary"}
	fieldKeys    = []string{"name", "widget", "required", "desc", "hide"}
	tableKeys    = []string{"name", "title", "desc", "fields", "search_fields", "handlers", "examples"}
	formKeys     = []string{"name", "desc", "target_table", "request_fields", "response_fields", "example"}
	chartKeys    = []string{"name", "desc", "source_table", "chart_type", "dimension", "metrics", "filters", "examples"}

	supportedWidgets    = []string{"input", "text_area", "textarea", "richtext", "integer", "float", "select", "datetime", "date", "files", "user", "table", "rate", "radio", "checkbox", "switch"}
	supportedHandlers   = []string{"OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRow"}
	supportedChartTypes = []string{"line", "bar", "pie"}
)

func ToolDescription() string {
	return "在 product_manager 角色下输出轻量结构化 PRD，供前端预览和用户确认。无副作用：不创建目录、不写文件、不 build。只允许 project、tables、forms、charts、rules 这 5 类业务结构，不输出 models/functions/workflow/route/method/order/columns/sample_rows/confirmation/widget tag。字段只写 name/widget/required/desc/hide；widget 只能是简单组件类型，例如 input、text_area、richtext、integer、float、select、datetime、date、files、user、table、rate、radio、checkbox、switch；整数数量/次数/0-10 整数评分用 integer，金额/比例/均值/可小数评分用 float，禁止使用 number；普通长文本用 text_area，图文、富文本或可插图片内容用 richtext；选项、默认值、范围、数据来源、计算规则都写进 desc，使用用户能看懂的自然语言。tables 直接包含 fields、search_fields、handlers、examples；search_fields 只描述搜索参数，不需要 handlers。除纯配置、小字典或无时间/用户概念的表外，大多数业务表默认带 创建开始时间/创建结束时间 两个 datetime 搜索条件，用于按记录创建时间范围查询；还要带一个用户筛选字段，优先用提交人、处理人、评分人、申请人等业务用户，没有明确业务用户时用创建人。handlers 只写 OnTableAddRow、OnTableUpdateRow、OnTableDeleteRow，只查询表填空数组。forms 只描述独立提交入口、target_table、request_fields、response_fields、example；纯文件处理、转换、计算类 Form 可不填 target_table。charts 只描述 source_table、chart_type、dimension、metrics、filters、examples；时间趋势图 filters 写清默认时间范围和粒度，波动型数据默认最近1天、自动粒度，并允许按分钟、按5分钟、按小时、按天、按月；dimension 推荐写字段名如 日期，写成 日期（按天/周/月）时会归一为 日期；chart examples 推荐写 {\"dimension\":\"2026-05-01\",\"metrics\":{\"NPS分数\":45,\"评分人数\":80}}，工具会归一为前端预览行。前端和生成链路按业务资源派生展示顺序：可维护 Table、Form、只读记录 Table、Chart。tables.examples 的每条记录只能使用同一个 table.fields[].name 里已经声明的用户可见业务字段名，key 必须逐字匹配；例如字段 name 是 工单标题 时示例必须写 \"工单标题\"，不要写 Title、Content、Attachment、Priority、Status、CreateTime 等结构体/json/db/系统字段。forms.example.request/response 同理分别匹配 request_fields/response_fields 的 name。tables.examples 最多 3 条；charts.examples 用 dimension/metrics 自然结构，建议 3-6 条、最多 12 条；不写 json/code/db 字段名。非 CRUD 逻辑如果会影响数据结构、不可逆副作用、权限边界或跨表事务，且无法从用户数据、file_profile 或常见默认值推断，才先追问；可合理默认的状态、评分、基础统计和筛选直接写入 rules 或 desc，不要因可选增强项阻塞 PRD。write_prd 成功后助手正文最多 1 句话提示用户确认，不复述 PRD 细节；收到确认前不要继续 create_directory、write_go_file 或 build_workspace。"
}

func ContractMarkdown() string {
	return strings.TrimSpace(contractMarkdown)
}

func ApplyContractMarkdown(content string) string {
	if !strings.Contains(content, ContractMarker) {
		return content
	}
	return strings.ReplaceAll(content, ContractMarker, ContractMarkdown())
}

func AllowedTopLevelKeys() map[string]struct{} {
	return keySet(topLevelKeys...)
}

func AllowedProjectKeys() map[string]struct{} {
	return keySet(projectKeys...)
}

func AllowedFieldKeys() map[string]struct{} {
	return keySet(fieldKeys...)
}

func AllowedTableKeys() map[string]struct{} {
	return keySet(tableKeys...)
}

func AllowedFormKeys() map[string]struct{} {
	return keySet(formKeys...)
}

func AllowedChartKeys() map[string]struct{} {
	return keySet(chartKeys...)
}

func IsSupportedWidget(widget string) bool {
	return containsFold(supportedWidgets, widget)
}

func IsSupportedHandler(handler string) bool {
	handler = strings.TrimSpace(handler)
	for _, supported := range supportedHandlers {
		if handler == supported {
			return true
		}
	}
	return false
}

func IsSupportedChartType(chartType string) bool {
	return containsFold(supportedChartTypes, chartType)
}

func NormalizeChartDimension(raw string) string {
	dimension := strings.TrimSpace(raw)
	for _, sep := range []string{"（", "(", "／", "/", "按"} {
		if idx := strings.Index(dimension, sep); idx > 0 {
			dimension = strings.TrimSpace(dimension[:idx])
		}
	}
	return dimension
}

func keySet(keys ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		out[key] = struct{}{}
	}
	return out
}

func containsFold(values []string, target string) bool {
	target = strings.ToLower(strings.TrimSpace(target))
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == target {
			return true
		}
	}
	return false
}

const contractMarkdown = `
## PRD 规则

- 只允许输出 ` + "`project/tables/forms/charts/rules`" + ` 这 5 类业务结构，不新增其他顶层字段。
- ` + "`project`" + ` 只写 ` + "`name/code/summary`" + `；新应用默认创建独立目录。
- ` + "`tables`" + ` 表示业务数据表和表格页语义；每个 table 直接写 ` + "`name/title/desc/fields/search_fields/handlers/examples`" + `；纯文件处理、转换、计算工具可以没有 table。
- 字段只写 ` + "`name/widget/required/desc/hide`" + `；` + "`widget`" + ` 只写简单组件类型，例如 ` + "`input`" + `、` + "`text_area`" + `、` + "`richtext`" + `、` + "`integer`" + `、` + "`float`" + `、` + "`select`" + `、` + "`datetime`" + `。整数数量/次数/0-10 整数评分用 ` + "`integer`" + `；金额、比例、均值、可小数评分用 ` + "`float`" + `；禁止使用 ` + "`number`" + `。
- 普通长文本字段用 ` + "`text_area`" + `；图文、富文本或可插图片内容用 ` + "`richtext`" + `。
- 不输出 widget tag；不要写 ` + "`name:状态;type:select;options:...`" + `。选项、默认值、范围、数据来源、计算规则全部写进 ` + "`desc`" + `，用用户能看懂的自然语言。
- ` + "`search_fields`" + ` 只描述搜索参数，不需要 ` + "`handlers`" + `。除纯配置、小字典或无时间/用户概念的表外，大多数业务表都要带常用搜索组合：` + "`创建开始时间`" + `、` + "`创建结束时间`" + ` 两个 ` + "`datetime`" + `，用于按记录创建时间范围查询；再加一个用户筛选字段。用户筛选优先用业务语义，例如 ` + "`提交人`" + `、` + "`处理人`" + `、` + "`评分人`" + `、` + "`申请人`" + `；没有明确业务用户时用 ` + "`创建人`" + `，表示系统记录的创建用户。例如 ` + "`{\"name\":\"创建开始时间\",\"widget\":\"datetime\",\"required\":false,\"desc\":\"按记录创建时间范围查询的开始时间。\"}`" + `、` + "`{\"name\":\"创建人\",\"widget\":\"user\",\"required\":false,\"desc\":\"按系统记录的创建人筛选。\"}`" + `。
- ` + "`handlers`" + ` 只表达表格行操作能力：` + "`OnTableAddRow`" + `、` + "`OnTableUpdateRow`" + `、` + "`OnTableDeleteRow`" + `；只查询表填空数组。
- ` + "`forms`" + ` 只描述独立提交入口、` + "`target_table`" + `、` + "`request_fields`" + `、` + "`response_fields`" + ` 和 ` + "`example`" + `；如果是纯处理型 Form，不写 ` + "`target_table`" + `。
- ` + "`charts`" + ` 只描述 ` + "`source_table`" + `、` + "`chart_type`" + `、` + "`dimension`" + `、` + "`metrics`" + `、` + "`filters`" + ` 和 ` + "`examples`" + `；图表类型只用 ` + "`line/bar/pie`" + `。时间趋势图的 ` + "`filters`" + ` 写清默认时间范围和粒度：波动型数据默认最近1天、自动粒度，并允许用户选择按分钟、按5分钟、按小时、按天、按月；长时间范围作为可选筛选，不要让默认图表一打开就查超长时间跨度。
- ` + "`charts[].dimension`" + ` 推荐写一个字段名，例如 ` + "`日期`" + `、` + "`评分类型`" + `、` + "`问卷名称`" + `；写成 ` + "`日期（按天/周/月）`" + ` 时工具会归一为 ` + "`日期`" + `。
- ` + "`charts[].examples`" + ` 推荐写模型自然结构：` + "`{\"dimension\":\"2026-05-01\",\"metrics\":{\"NPS分数\":45,\"评分人数\":80}}`" + `；工具会归一成前端预览行。
- ` + "`tables[].examples`" + ` 是示例业务记录；每条记录的 key 必须逐字等于同一个 table 的 ` + "`fields[].name`" + `，先对照 ` + "`fields`" + ` 再写示例。例如 ` + "`fields`" + ` 里是 ` + "`工单标题/详细内容/附件/优先级/工单状态`" + `，示例就写这些中文 key；不要写 ` + "`Title/Content/Attachment/Priority/Status/CreateTime`" + ` 等结构体、JSON、数据库或系统字段。` + "`forms[].example.request`" + ` 的 key 必须来自 ` + "`request_fields[].name`" + `，` + "`forms[].example.response`" + ` 的 key 必须来自 ` + "`response_fields[].name`" + `。表格示例最多 3 条；` + "`charts[].examples`" + ` 用上面的 ` + "`dimension/metrics`" + ` 自然结构，建议 3-6 条，最多 12 条。不写 json/code/db 字段名。
- ` + "`rules`" + ` 写业务规则、计算口径、状态流转、只读边界等自然语言规则。
- 非 CRUD 逻辑如果会影响数据结构、不可逆副作用、权限边界或跨表事务，且无法从用户数据、` + "`file_profile`" + ` 或常见默认值推断，才先追问并写清楚。可合理默认的状态、评分、基础统计和筛选直接写入 ` + "`rules`" + ` 或 ` + "`desc`" + `，不要因可选增强项阻塞 PRD。
- 禁止输出旧结构：` + "`models/functions/workflow/route/method/order/columns/sample_rows/preview_data/acceptance_cases/confirmation`" + `。
- ` + "`write_prd`" + ` 成功后，助手正文最多 1 句话提示用户确认，不复述 PRD 表格、字段清单或功能清单。

## 代表性输出示例

输出时按这个结构替换业务内容，不要新增顶层字段。注意：表格示例字段必须逐字来自同表 ` + "`fields[].name`" + `，不能使用代码、JSON、数据库或系统字段名；大多数表格搜索条件都包含按记录创建时间查询的 ` + "`创建开始时间`" + `、` + "`创建结束时间`" + `，以及按用户查询的 ` + "`创建人`" + ` 或业务用户字段。

` + "```json" + `
{
  "project": {"name": "NPS 客户满意度调研系统", "code": "nps_survey", "summary": "收集客户 0-10 分推荐意愿评分，计算 NPS 分数并查看趋势。"},
  "tables": [
    {
      "name": "NPS问卷",
      "title": "NPS问卷管理",
      "desc": "维护 NPS 调研问卷。",
      "fields": [
        {"name": "问卷标题", "widget": "input", "required": true, "desc": "调研问卷标题。"},
        {"name": "截止时间", "widget": "datetime", "required": true, "desc": "问卷停止收集评分的时间。"},
        {"name": "状态", "widget": "select", "required": false, "hide": "create,update", "desc": "可选：草稿、进行中、已结束。"}
      ],
      "search_fields": [
        {"name": "问卷标题", "widget": "input", "required": false, "desc": "按标题搜索。"},
        {"name": "创建人", "widget": "user", "required": false, "desc": "按系统记录的创建人筛选。"},
        {"name": "创建开始时间", "widget": "datetime", "required": false, "desc": "按记录创建时间范围查询的开始时间。"},
        {"name": "创建结束时间", "widget": "datetime", "required": false, "desc": "按记录创建时间范围查询的结束时间。"}
      ],
      "handlers": ["OnTableAddRow", "OnTableUpdateRow", "OnTableDeleteRow"],
      "examples": [
        {"问卷标题": "产品体验调研", "截止时间": "2026-05-31 23:59", "状态": "进行中"}
      ]
    },
    {
      "name": "NPS评分记录",
      "title": "NPS评分记录",
      "desc": "只读查看客户提交的评分。",
      "fields": [
        {"name": "提交时间", "widget": "datetime", "required": false, "hide": "create,update", "desc": "评分提交时间。"},
        {"name": "问卷标题", "widget": "input", "required": false, "hide": "create,update", "desc": "关联的问卷。"},
        {"name": "评分", "widget": "integer", "required": false, "hide": "create,update", "desc": "0-10 的整数评分。"},
        {"name": "评分类型", "widget": "select", "required": false, "hide": "create,update", "desc": "9-10 推荐者，7-8 被动者，0-6 贬低者。"}
      ],
      "search_fields": [
        {"name": "问卷标题", "widget": "input", "required": false, "desc": "按问卷搜索。"},
        {"name": "评分类型", "widget": "select", "required": false, "desc": "按推荐者、被动者、贬低者筛选。"},
        {"name": "创建开始时间", "widget": "datetime", "required": false, "desc": "按记录创建时间范围查询的开始时间。"},
        {"name": "创建结束时间", "widget": "datetime", "required": false, "desc": "按记录创建时间范围查询的结束时间。"}
      ],
      "handlers": [],
      "examples": [
        {"提交时间": "2026-05-09 14:00", "问卷标题": "产品体验调研", "评分": 9, "评分类型": "推荐者"}
      ]
    }
  ],
  "forms": [
    {
      "name": "提交NPS评分",
      "desc": "客户选择问卷后提交 0-10 分评分。",
      "target_table": "NPS评分记录",
      "request_fields": [
        {"name": "问卷标题", "widget": "select", "required": true, "desc": "选择进行中的问卷。"},
        {"name": "评分", "widget": "integer", "required": true, "desc": "0-10 整数评分。"}
      ],
      "response_fields": [
        {"name": "提交结果", "widget": "input", "required": false, "desc": "提交成功或失败信息。"}
      ],
      "example": {"request": {"问卷标题": "产品体验调研", "评分": 9}, "response": {"提交结果": "评分成功"}}
    }
  ],
  "charts": [
    {
      "name": "NPS趋势分析",
      "desc": "按日期查看 NPS 分数变化。",
      "source_table": "NPS评分记录",
      "chart_type": "line",
      "dimension": "日期",
      "metrics": ["NPS分数"],
      "filters": ["时间范围默认最近1天，可选最近7天、最近30天、自定义", "聚合粒度默认自动，可选按分钟、按5分钟、按小时、按天、按月"],
      "examples": [
        {"dimension": "2026-05-01", "metrics": {"NPS分数": 35}}
      ]
    }
  ],
  "rules": ["NPS = 推荐者占比 - 贬低者占比。"]
}
` + "```" + `
`
