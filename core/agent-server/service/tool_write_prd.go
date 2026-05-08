package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

type WritePRDTool struct{}

type writePRDArgs struct {
	Project         writePRDProject          `json:"project" schema_desc:"目录和项目确认信息" schema_required:"true"`
	Models          []writePRDModel          `json:"models" schema_desc:"本次 PRD 的业务模型；只描述用户可见字段、widget、validate、hide，不输出字段 code 或 Go 源码" schema_required:"true"`
	Functions       []writePRDFunction       `json:"functions" schema_desc:"本次要创建的功能列表；只允许 table、form、chart 三类" schema_required:"true"`
	AcceptanceCases []writePRDAcceptanceCase `json:"acceptance_cases" schema_desc:"验收用例：用户确认后用于实现和测试"`
	Confirmation    writePRDConfirmation     `json:"confirmation" schema_desc:"给用户确认的简短问题"`
}

func (args *writePRDArgs) UnmarshalJSON(data []byte) error {
	type rawWritePRDArgs struct {
		Project         json.RawMessage `json:"project"`
		Models          json.RawMessage `json:"models"`
		Functions       json.RawMessage `json:"functions"`
		AcceptanceCases json.RawMessage `json:"acceptance_cases"`
		Confirmation    json.RawMessage `json:"confirmation"`
	}

	var raw rawWritePRDArgs
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if len(raw.Project) > 0 {
		if err := unmarshalWritePRDMaybeJSONString(raw.Project, &args.Project); err != nil {
			return fmt.Errorf("project: %w", err)
		}
	}
	if len(raw.Models) > 0 {
		if err := unmarshalWritePRDMaybeJSONString(raw.Models, &args.Models); err != nil {
			return fmt.Errorf("models: %w", err)
		}
	}
	if len(raw.Functions) > 0 {
		if err := unmarshalWritePRDMaybeJSONStringWithRepair(raw.Functions, &args.Functions, repairWritePRDFunctionsJSONString); err != nil {
			return fmt.Errorf("functions: %w", err)
		}
	}
	if len(raw.AcceptanceCases) > 0 {
		if err := unmarshalWritePRDMaybeJSONString(raw.AcceptanceCases, &args.AcceptanceCases); err != nil {
			return fmt.Errorf("acceptance_cases: %w", err)
		}
	}
	if len(raw.Confirmation) > 0 {
		if err := unmarshalWritePRDMaybeJSONString(raw.Confirmation, &args.Confirmation); err != nil {
			return fmt.Errorf("confirmation: %w", err)
		}
	}
	return nil
}

func unmarshalWritePRDMaybeJSONString[T any](data []byte, out *T) error {
	return unmarshalWritePRDMaybeJSONStringWithRepair(data, out, nil)
}

func unmarshalWritePRDMaybeJSONStringWithRepair[T any](data []byte, out *T, repair func(string) string) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		return nil
	}
	if strings.HasPrefix(raw, `"`) {
		var encoded string
		if err := json.Unmarshal(data, &encoded); err != nil {
			return formatWritePRDJSONError(raw, err)
		}
		encoded = strings.TrimSpace(encoded)
		if encoded == "" {
			return nil
		}
		if err := json.Unmarshal([]byte(encoded), out); err != nil {
			if repair != nil {
				repaired := repair(encoded)
				if repaired != encoded {
					if repairedErr := json.Unmarshal([]byte(repaired), out); repairedErr == nil {
						return nil
					}
				}
			}
			return formatWritePRDJSONError(encoded, err)
		}
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		if repair != nil {
			repaired := repair(raw)
			if repaired != raw {
				if repairedErr := json.Unmarshal([]byte(repaired), out); repairedErr == nil {
					return nil
				}
			}
		}
		return formatWritePRDJSONError(raw, err)
	}
	return nil
}

func repairWritePRDFunctionsJSONString(raw string) string {
	var out strings.Builder
	out.Grow(len(raw))
	inString := false
	escaped := false
	objectDepth := 0
	for i := 0; i < len(raw); i++ {
		ch := raw[i]
		if inString {
			out.WriteByte(ch)
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inString = false
			}
			continue
		}
		switch ch {
		case '"':
			inString = true
			out.WriteByte(ch)
		case '{':
			objectDepth++
			out.WriteByte(ch)
		case '}':
			if objectDepth == 1 && nextJSONKeyAfterComma(raw, i+1) != "" {
				// 常见模型错误：{"model":"X","table":{...}}, "title":"..."
				// 此时 table/form/chart 所在的 function 对象被提前关闭，跳过这个多余的 }。
				continue
			}
			if objectDepth > 0 {
				objectDepth--
			}
			out.WriteByte(ch)
		default:
			out.WriteByte(ch)
		}
	}
	return out.String()
}

func nextJSONKeyAfterComma(raw string, start int) string {
	i := skipJSONWhitespace(raw, start)
	if i >= len(raw) || raw[i] != ',' {
		return ""
	}
	i = skipJSONWhitespace(raw, i+1)
	if i >= len(raw) || raw[i] != '"' {
		return ""
	}
	i++
	keyStart := i
	escaped := false
	for i < len(raw) {
		ch := raw[i]
		if escaped {
			escaped = false
			i++
			continue
		}
		if ch == '\\' {
			escaped = true
			i++
			continue
		}
		if ch == '"' {
			key := raw[keyStart:i]
			if key == "title" || key == "type" || key == "route" || key == "method" || key == "model" || key == "description" {
				return key
			}
			return ""
		}
		i++
	}
	return ""
}

func skipJSONWhitespace(raw string, start int) int {
	for start < len(raw) {
		switch raw[start] {
		case ' ', '\n', '\r', '\t':
			start++
		default:
			return start
		}
	}
	return start
}

func formatWritePRDJSONError(raw string, err error) error {
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("%w near %q", err, writePRDJSONErrorSnippet(raw, syntaxErr.Offset))
	}
	return err
}

func writePRDJSONErrorSnippet(raw string, offset int64) string {
	if offset <= 0 {
		return ""
	}
	index := int(offset) - 1
	if index < 0 {
		index = 0
	}
	if index > len(raw) {
		index = len(raw)
	}
	start := index - 80
	if start < 0 {
		start = 0
	}
	end := index + 80
	if end > len(raw) {
		end = len(raw)
	}
	return strings.TrimSpace(raw[start:end])
}

type writePRDProject struct {
	Name               string `json:"name" schema_desc:"目录显示名称，例如：会员收银系统" schema_required:"true"`
	Code               string `json:"code" schema_desc:"目录 code，小写下划线，例如：cashier" schema_required:"true"`
	CreateNewDirectory bool   `json:"create_new_directory" schema_desc:"是否创建新目录；新应用通常为 true" schema_required:"true"`
	ParentDirectory    string `json:"parent_directory" schema_desc:"创建新目录时的父目录；不传则使用当前目录"`
	TargetDirectory    string `json:"target_directory" schema_desc:"不创建新目录时，要放入的现有目录完整路径"`
	Summary            string `json:"summary" schema_desc:"业务目标一句话说明"`
	Reason             string `json:"reason" schema_desc:"为什么创建新目录或为什么放入现有目录"`
}

type writePRDFunction struct {
	Title       string         `json:"title" schema_desc:"业务功能名称，例如：支付记录表" schema_required:"true"`
	Order       int            `json:"order,omitempty" schema_desc:"功能展示和实现顺序，从 1 开始；必须按业务流程排序，例如基础管理表→提交 Form→记录查询表→统计 Chart"`
	Type        string         `json:"type" schema_desc:"函数类型，只允许 table、form、chart" schema_enum:"table,form,chart" schema_required:"true"`
	Route       string         `json:"route" schema_desc:"函数路由，必须带类型后缀，例如 cashier_desk.form" schema_required:"true"`
	Method      string         `json:"method" schema_desc:"请求方法；form 通常 POST，table/chart 通常 GET" schema_enum:"GET,POST"`
	Model       string         `json:"model" schema_desc:"该功能主要依赖的数据模型名；table/form 必填，chart 可填统计来源模型"`
	Description string         `json:"description" schema_desc:"业务说明和能力边界"`
	Table       *writePRDTable `json:"table,omitempty" schema_desc:"type=table 时填写；Table 是业务列表预览，不写 Go struct、字段 code、widget 或数据库细节"`
	Form        *writePRDForm  `json:"form,omitempty" schema_desc:"type=form 时填写；Form 请求字段固定为 字段/类型/必填/默认值/说明"`
	Chart       *writePRDChart `json:"chart,omitempty" schema_desc:"type=chart 时填写；一个 chart 函数只对应一张图"`
}

type writePRDModel struct {
	Name        string               `json:"name" schema_desc:"业务模型名称，例如 客户工单；开发阶段自动生成 Go 结构体名、json 字段名、gorm column 和表名" schema_required:"true"`
	DisplayName string               `json:"display_name,omitempty" schema_desc:"兼容旧字段：业务显示名；新 PRD 可不填" schema_ignore:"true"`
	TableName   string               `json:"table_name,omitempty" schema_desc:"兼容旧字段：数据库表名；新 PRD 可不填，由开发阶段自动生成" schema_ignore:"true"`
	Description string               `json:"description,omitempty" schema_desc:"模型说明"`
	Fields      []writePRDModelField `json:"fields" schema_desc:"业务字段列表；每个字段只需要 name、widget、validate、hide、description，开发阶段自动生成代码字段" schema_required:"true"`
}

type writePRDModelField struct {
	Name          string   `json:"name" schema_desc:"字段显示名，例如 客户姓名；不要写字段 code" schema_required:"true"`
	Widget        string   `json:"widget" schema_desc:"widget tag 内容，不含反引号，例如 name:客户姓名;type:input；选项、颜色、默认值、format 等个性化参数都写在这里" schema_required:"true"`
	Validate      string   `json:"validate,omitempty" schema_desc:"validate tag 内容，不含反引号；是否必填、长度、范围等由这里决定"`
	Hide          string   `json:"hide,omitempty" schema_desc:"hide tag 内容，不含反引号，例如 create,update、list"`
	Description   string   `json:"description,omitempty" schema_desc:"复杂字段的业务说明；简单字段可不填"`
	GoName        string   `json:"go_name,omitempty" schema_ignore:"true"`
	JSONName      string   `json:"json_name,omitempty" schema_ignore:"true"`
	GoType        string   `json:"go_type,omitempty" schema_ignore:"true"`
	DisplayName   string   `json:"display_name,omitempty" schema_ignore:"true"`
	Gorm          string   `json:"gorm,omitempty" schema_ignore:"true"`
	Options       []string `json:"options,omitempty" schema_ignore:"true"`
	OptionColors  []string `json:"option_colors,omitempty" schema_ignore:"true"`
	RenderDefault string   `json:"render_default,omitempty" schema_ignore:"true"`
}

type writePRDPreviewField struct {
	Name        string `json:"name" schema_desc:"字段显示名，例如：开始时间" schema_required:"true"`
	Type        string `json:"type" schema_desc:"组件类型，例如：input、select、datetime、number" schema_required:"true"`
	Required    bool   `json:"required" schema_desc:"是否必填" schema_required:"true"`
	Desc        string `json:"desc,omitempty" schema_desc:"字段说明；复杂信息、可选项、默认值都写这里"`
	Description string `json:"description,omitempty" schema_ignore:"true"`
}

type writePRDPreviewFields []writePRDPreviewField

func (fields *writePRDPreviewFields) UnmarshalJSON(data []byte) error {
	raw := strings.TrimSpace(string(data))
	if raw == "" || raw == "null" {
		*fields = nil
		return nil
	}

	var structured []writePRDPreviewField
	structuredErr := json.Unmarshal(data, &structured)
	if structuredErr == nil {
		*fields = structured
		return nil
	}

	var legacy []string
	if err := json.Unmarshal(data, &legacy); err == nil {
		out := make(writePRDPreviewFields, 0, len(legacy))
		for _, item := range legacy {
			name := strings.TrimSpace(item)
			if name == "" {
				continue
			}
			out = append(out, writePRDPreviewField{
				Name:     name,
				Type:     inferWritePRDPreviewFieldType(name),
				Required: false,
				Desc:     "筛选条件",
			})
		}
		*fields = out
		return nil
	}

	return structuredErr
}

type writePRDTable struct {
	Capability    string                   `json:"capability" schema_desc:"能力边界，例如：仅列表查询，记录由收银台 Form 产生"`
	ReadOnly      bool                     `json:"readonly" schema_desc:"是否只读；只读时不允许手工新增、编辑、删除"`
	Operations    []string                 `json:"operations" schema_desc:"允许的业务操作，例如：列表查询、新增、编辑、删除"`
	RequestFields []writePRDFormField      `json:"request_fields" schema_desc:"Table Request 搜索/筛选字段，固定五列：字段、类型、必填、默认值、说明；没有搜索字段填空数组" schema_required:"true"`
	Filters       []string                 `json:"filters" schema_desc:"兼容旧字段：业务筛选条件，例如：状态、创建时间；新 PRD 优先使用 request_fields"`
	Columns       []string                 `json:"columns" schema_desc:"列表模式表头，必须是用户将看到的业务列" schema_required:"true"`
	SampleRows    []map[string]interface{} `json:"sample_rows" schema_desc:"列表模式示例行；标准写法是 key 对应 columns 的用户可见列名；工具和前端会兼容旧字段名和数字值，至少 1 行" schema_required:"true"`
}

type writePRDForm struct {
	RequestFields  []writePRDFormField     `json:"request_fields" schema_desc:"请求字段；每项只写 name、type、required、desc，复杂细节写 desc" schema_required:"true"`
	ResponseFields []writePRDResponseField `json:"response_fields" schema_desc:"Form Response 需要展示给用户看的响应字段；每项优先写 name、type、example、desc；没有就填空数组" schema_required:"true"`
}

type writePRDFormField struct {
	Name        string `json:"name" schema_desc:"字段显示名，例如：商品清单" schema_required:"true"`
	Type        string `json:"type" schema_desc:"组件/业务字段类型，例如：input、select、datetime、number、table、text_area、files" schema_required:"true"`
	Required    bool   `json:"required" schema_desc:"是否必填" schema_required:"true"`
	Desc        string `json:"desc,omitempty" schema_desc:"字段说明；可选项、默认值、OnSelectFuzzy、type:table 等复杂信息都写这里"`
	Field       string `json:"field,omitempty" schema_ignore:"true"`
	Default     string `json:"default,omitempty" schema_ignore:"true"`
	Description string `json:"description,omitempty" schema_ignore:"true"`
}

type writePRDResponseField struct {
	Name        string `json:"name" schema_desc:"响应字段显示名" schema_required:"true"`
	Type        string `json:"type" schema_desc:"响应字段类型" schema_required:"true"`
	Example     string `json:"example,omitempty" schema_desc:"示例值"`
	Desc        string `json:"desc,omitempty" schema_desc:"响应说明"`
	Field       string `json:"field,omitempty" schema_ignore:"true"`
	Description string `json:"description,omitempty" schema_ignore:"true"`
}

type writePRDChart struct {
	ChartType      string                   `json:"chart_type" schema_desc:"图表类型；推荐使用 line、bar、pie，对应折线图、柱状图、饼图；SDK 额外支持 gauge" schema_enum:"line,bar,pie,gauge,LineChart,BarChart,PieChart,GaugeChart" schema_required:"true"`
	Dimension      string                   `json:"dimension" schema_desc:"维度，例如：日期、状态、负责人" schema_required:"true"`
	Metrics        []string                 `json:"metrics" schema_desc:"指标，例如：新增数量、完成数量" schema_required:"true"`
	Filters        writePRDPreviewFields    `json:"filters" schema_desc:"图表查询条件；每个字段只写 name/type/required/desc；时间区间必须拆成开始时间、结束时间两个 datetime 字段"`
	PreviewData    []map[string]interface{} `json:"preview_data" schema_desc:"图表预览数据；折线/柱状一般为日期或分类行，饼图一般为 name/value 行" schema_required:"true"`
	Summary        []writePRDChartSummary   `json:"summary" schema_desc:"图表摘要指标卡，例如总数、占比、平均值、NPS 分数"`
	RequestFields  []writePRDFormField      `json:"request_fields,omitempty" schema_desc:"兼容旧字段：Chart Request 查询/筛选字段；新 PRD 优先使用 filters"`
	ResponseFields []writePRDResponseField  `json:"response_fields,omitempty" schema_desc:"兼容旧字段：Chart Response/Metadata 字段；新 PRD 优先使用 preview_data 和 summary"`
}

type writePRDChartSummary struct {
	Name        string      `json:"name" schema_desc:"指标名称，例如：总销售额" schema_required:"true"`
	Value       interface{} `json:"value" schema_desc:"指标展示值，例如：1280.50 或 52%" schema_required:"true"`
	Desc        string      `json:"desc,omitempty" schema_desc:"指标口径说明"`
	Description string      `json:"description,omitempty" schema_ignore:"true"`
}

type writePRDAcceptanceCase struct {
	Name     string `json:"name" schema_desc:"验收项，例如：查询列表" schema_required:"true"`
	Action   string `json:"action" schema_desc:"用户操作" schema_required:"true"`
	Expected string `json:"expected" schema_desc:"预期结果" schema_required:"true"`
}

type writePRDConfirmation struct {
	Question string `json:"question" schema_desc:"确认问题，例如：请确认是否按以上 PRD 创建目录和生成代码"`
}

type writePRDResultData struct {
	Kind            string                   `json:"kind" schema_desc:"固定为 agent_app_prd" schema_required:"true"`
	Project         writePRDProject          `json:"project" schema_desc:"目录和项目确认信息" schema_required:"true"`
	Models          []writePRDModel          `json:"models" schema_desc:"规范数据模型" schema_required:"true"`
	Functions       []writePRDFunction       `json:"functions" schema_desc:"功能列表" schema_required:"true"`
	AcceptanceCases []writePRDAcceptanceCase `json:"acceptance_cases" schema_desc:"验收用例"`
	Confirmation    writePRDConfirmation     `json:"confirmation" schema_desc:"确认问题"`
	Issues          []string                 `json:"issues,omitempty" schema_desc:"PRD 结构问题；非空时本次工具返回错误"`
}

var writePRDToolDef = toolDefinitionWithOutput[writePRDArgs, structuredToolResultSchema[writePRDResultData]](
	"write_prd",
	"在 app.plan 阶段输出结构化 PRD 预览供前端渲染和用户确认。无副作用：不创建目录、不写文件、不 build。必填 project、models 与 functions；models 只写业务字段 name、widget、validate、hide、description，不写 go_source、Go 源码、字段 code、json_name、go_name、go_type、gorm 或 example。字段个性化参数统一写进 widget，是否必填和校验写进 validate，展示范围写进 hide。functions 必须直接传 JSON 数组，首层值是数组本身，不要把思考过程写进字段。functions 必须按业务流程排序并填写 order，从 1 开始；通常是基础资料/管理表 → 业务提交 Form → 产生的记录查询表 → 统计 Chart。Table 自带列表查询、新增、编辑、删除能力；如果同一个 model 已有可新增/编辑的 Table，普通提交/新增/创建不要再单独设计 Form，除非 Form 在 description 明确说明差异（外部/匿名/客户自助、批量导入、文件解析、计算生成、支付结算、审批流、跨多表事务、只提交不编辑等）。functions 只允许 table、form、chart 三类，并且 type=table 只填 table，type=form 只填 form，type=chart 只填 chart。Table 必须区分 request_fields 搜索请求、columns 列表列和 sample_rows 示例行；request_fields 每项只写 name/type/required/desc，columns 只是列表表头，sample_rows 标准 key 使用 columns 的用户可见列名。Form 必须区分 request_fields 与 response_fields；字段同样优先使用 name/type/required/desc，响应字段可额外写 example。Chart 使用自己的轻量结构：chart_type 推荐 line/bar/pie，dimension 写维度，metrics 写指标，filters 写查询字段且每个字段只用 name/type/required/desc，时间范围必须拆成开始时间和结束时间两个 datetime 字段，preview_data 写可视化样例数据，summary 写摘要指标；兼容旧 field/default/description 和旧 chart.request_fields/response_fields，但新 PRD 优先用轻量字段。一个 chart 路由只表示一张图。write_prd 成功后助手正文最多 1 句话提示用户确认，不要再复述 PRD 表格、字段、功能清单或确认问题。前端会展示确认按钮和备注输入，确认后会把完整 PRD 作为新会话第一条消息交给 app.create 生成阶段；收到确认前不要继续 create_directory、write_go_file 或 build_workspace。",
)

func (t *WritePRDTool) Definition() dto.ToolDef {
	return writePRDToolDef
}

func (t *WritePRDTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	args, err := decodeToolArgs[writePRDArgs](normalizeWritePRDRawArgs(call.Args))
	if err != nil {
		return toolResult("write_prd 参数解析失败: "+err.Error(), true)
	}
	result := buildWritePRDResult(args)
	if len(result.Issues) > 0 {
		return toolResultWithStructuredData(result, true, "write_prd 参数不完整，先修正 PRD 结构后再继续。")
	}
	return toolResultWithStructuredData(result, false, "PRD 预览已生成；请等待用户点击确认 PRD 或回复确认。不要再复述 PRD 细节。")
}

func normalizeWritePRDRawArgs(args map[string]interface{}) map[string]interface{} {
	if len(args) == 0 {
		return args
	}
	out := make(map[string]interface{}, len(args))
	for key, value := range args {
		out[key] = value
	}
	for _, key := range []string{"models", "functions", "acceptance_cases"} {
		out[key] = decodeJSONEncodedToolArg(out[key])
	}
	for _, key := range []string{"project", "confirmation"} {
		out[key] = decodeJSONEncodedToolArg(out[key])
	}
	return out
}

func decodeJSONEncodedToolArg(value interface{}) interface{} {
	raw, ok := value.(string)
	if !ok {
		return value
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return value
	}
	var decoded interface{}
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return value
	}
	return decoded
}

func buildWritePRDResult(args writePRDArgs) writePRDResultData {
	result := writePRDResultData{
		Kind:            "agent_app_prd",
		Project:         normalizeWritePRDProject(args.Project),
		Models:          normalizeWritePRDModels(args.Models),
		Functions:       normalizeWritePRDFunctions(args.Functions),
		AcceptanceCases: normalizeWritePRDAcceptanceCases(args.AcceptanceCases),
		Confirmation:    normalizeWritePRDConfirmation(args.Confirmation),
	}
	result.Issues = validateWritePRDResult(result)
	return result
}

func normalizeWritePRDProject(project writePRDProject) writePRDProject {
	project.Name = strings.TrimSpace(project.Name)
	project.Code = strings.TrimSpace(project.Code)
	project.ParentDirectory = strings.TrimSpace(project.ParentDirectory)
	project.TargetDirectory = strings.TrimSpace(project.TargetDirectory)
	project.Summary = strings.TrimSpace(project.Summary)
	project.Reason = strings.TrimSpace(project.Reason)
	return project
}

func normalizeWritePRDFunctions(functions []writePRDFunction) []writePRDFunction {
	out := make([]writePRDFunction, 0, len(functions))
	for idx, fn := range functions {
		fn.Title = strings.TrimSpace(fn.Title)
		if fn.Order <= 0 {
			fn.Order = idx + 1
		}
		fn.Type = strings.ToLower(strings.TrimSpace(fn.Type))
		fn.Route = strings.TrimSpace(fn.Route)
		fn.Method = strings.ToUpper(strings.TrimSpace(fn.Method))
		fn.Model = strings.TrimSpace(fn.Model)
		fn.Description = strings.TrimSpace(fn.Description)
		if fn.Method == "" {
			if fn.Type == "form" {
				fn.Method = "POST"
			} else if fn.Type == "table" || fn.Type == "chart" {
				fn.Method = "GET"
			}
		}
		if fn.Table != nil {
			normalizeWritePRDTable(fn.Table)
		}
		if fn.Form != nil {
			normalizeWritePRDForm(fn.Form)
		}
		if fn.Chart != nil {
			normalizeWritePRDChart(fn.Chart)
		}
		out = append(out, fn)
	}
	return out
}

func normalizeWritePRDModels(models []writePRDModel) []writePRDModel {
	out := make([]writePRDModel, 0, len(models))
	for _, model := range models {
		model.Name = strings.TrimSpace(model.Name)
		model.DisplayName = strings.TrimSpace(model.DisplayName)
		model.TableName = strings.TrimSpace(model.TableName)
		model.Description = strings.TrimSpace(model.Description)
		for i := range model.Fields {
			field := &model.Fields[i]
			field.Name = strings.TrimSpace(field.Name)
			field.GoName = strings.TrimSpace(field.GoName)
			field.JSONName = strings.TrimSpace(field.JSONName)
			field.GoType = strings.TrimSpace(field.GoType)
			field.DisplayName = strings.TrimSpace(field.DisplayName)
			if field.Name == "" {
				field.Name = field.DisplayName
			}
			if field.Name == "" {
				field.Name = parseWritePRDWidgetValue(field.Widget, "name")
			}
			field.Gorm = strings.TrimSpace(field.Gorm)
			field.Widget = strings.TrimSpace(field.Widget)
			if field.Widget == "" && field.Name != "" {
				field.Widget = "name:" + field.Name + ";type:input"
			}
			field.Validate = strings.TrimSpace(field.Validate)
			field.Hide = strings.TrimSpace(field.Hide)
			field.Description = strings.TrimSpace(field.Description)
			field.Options = trimStringSlice(field.Options)
			field.OptionColors = trimStringSlice(field.OptionColors)
			field.RenderDefault = strings.TrimSpace(field.RenderDefault)
		}
		out = append(out, model)
	}
	return out
}

func parseWritePRDWidgetValue(raw string, key string) string {
	for _, part := range strings.Split(raw, ";") {
		idx := strings.Index(part, ":")
		if idx <= 0 {
			continue
		}
		if strings.TrimSpace(part[:idx]) == key {
			return strings.TrimSpace(part[idx+1:])
		}
	}
	return ""
}

func normalizeWritePRDTable(table *writePRDTable) {
	table.Capability = strings.TrimSpace(table.Capability)
	table.Operations = trimStringSlice(table.Operations)
	table.Filters = trimStringSlice(table.Filters)
	normalizeWritePRDFormFields(table.RequestFields)
	table.Columns = trimStringSlice(table.Columns)
	for i := range table.SampleRows {
		if table.SampleRows[i] == nil {
			continue
		}
		normalized := make(map[string]interface{}, len(table.SampleRows[i]))
		for key, value := range table.SampleRows[i] {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey != "" {
				normalized[trimmedKey] = normalizeWritePRDScalar(value)
			}
		}
		table.SampleRows[i] = normalized
	}
}

func normalizeWritePRDScalar(value interface{}) interface{} {
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return value
}

func normalizeWritePRDForm(form *writePRDForm) {
	normalizeWritePRDFormFields(form.RequestFields)
	for i := range form.ResponseFields {
		form.ResponseFields[i].Name = strings.TrimSpace(form.ResponseFields[i].Name)
		form.ResponseFields[i].Field = strings.TrimSpace(form.ResponseFields[i].Field)
		if form.ResponseFields[i].Name == "" {
			form.ResponseFields[i].Name = form.ResponseFields[i].Field
		}
		form.ResponseFields[i].Field = ""
		form.ResponseFields[i].Type = strings.TrimSpace(form.ResponseFields[i].Type)
		form.ResponseFields[i].Example = strings.TrimSpace(form.ResponseFields[i].Example)
		form.ResponseFields[i].Desc = strings.TrimSpace(form.ResponseFields[i].Desc)
		form.ResponseFields[i].Description = strings.TrimSpace(form.ResponseFields[i].Description)
		if form.ResponseFields[i].Desc == "" {
			form.ResponseFields[i].Desc = form.ResponseFields[i].Description
		}
		form.ResponseFields[i].Description = ""
	}
}

func normalizeWritePRDChart(chart *writePRDChart) {
	chart.ChartType = strings.TrimSpace(chart.ChartType)
	chart.Dimension = strings.TrimSpace(chart.Dimension)
	chart.Metrics = trimStringSlice(chart.Metrics)
	normalizeWritePRDPreviewFields(chart.Filters)
	for i := range chart.PreviewData {
		if chart.PreviewData[i] == nil {
			continue
		}
		normalized := make(map[string]interface{}, len(chart.PreviewData[i]))
		for key, value := range chart.PreviewData[i] {
			trimmedKey := strings.TrimSpace(key)
			if trimmedKey != "" {
				normalized[trimmedKey] = normalizeWritePRDScalar(value)
			}
		}
		chart.PreviewData[i] = normalized
	}
	for i := range chart.Summary {
		chart.Summary[i].Name = strings.TrimSpace(chart.Summary[i].Name)
		chart.Summary[i].Desc = strings.TrimSpace(chart.Summary[i].Desc)
		chart.Summary[i].Description = strings.TrimSpace(chart.Summary[i].Description)
		if chart.Summary[i].Desc == "" {
			chart.Summary[i].Desc = chart.Summary[i].Description
		}
		chart.Summary[i].Value = normalizeWritePRDScalar(chart.Summary[i].Value)
	}
	normalizeWritePRDFormFields(chart.RequestFields)
	for i := range chart.ResponseFields {
		chart.ResponseFields[i].Name = strings.TrimSpace(chart.ResponseFields[i].Name)
		chart.ResponseFields[i].Field = strings.TrimSpace(chart.ResponseFields[i].Field)
		if chart.ResponseFields[i].Name == "" {
			chart.ResponseFields[i].Name = chart.ResponseFields[i].Field
		}
		chart.ResponseFields[i].Field = ""
		chart.ResponseFields[i].Type = strings.TrimSpace(chart.ResponseFields[i].Type)
		chart.ResponseFields[i].Example = strings.TrimSpace(chart.ResponseFields[i].Example)
		chart.ResponseFields[i].Desc = strings.TrimSpace(chart.ResponseFields[i].Desc)
		chart.ResponseFields[i].Description = strings.TrimSpace(chart.ResponseFields[i].Description)
		if chart.ResponseFields[i].Desc == "" {
			chart.ResponseFields[i].Desc = chart.ResponseFields[i].Description
		}
		chart.ResponseFields[i].Description = ""
	}
}

func normalizeWritePRDPreviewFields(fields writePRDPreviewFields) {
	for i := range fields {
		fields[i].Name = strings.TrimSpace(fields[i].Name)
		fields[i].Type = strings.TrimSpace(fields[i].Type)
		fields[i].Desc = strings.TrimSpace(fields[i].Desc)
		fields[i].Description = strings.TrimSpace(fields[i].Description)
		if fields[i].Desc == "" {
			fields[i].Desc = fields[i].Description
		}
		if fields[i].Type == "" {
			fields[i].Type = inferWritePRDPreviewFieldType(fields[i].Name + " " + fields[i].Desc)
		}
	}
}

func inferWritePRDPreviewFieldType(text string) string {
	lower := strings.ToLower(text)
	if strings.Contains(text, "时间") || strings.Contains(text, "日期") || strings.Contains(lower, "date") || strings.Contains(lower, "time") {
		return "datetime"
	}
	if strings.Contains(text, "数量") || strings.Contains(text, "金额") || strings.Contains(text, "价格") || strings.Contains(text, "分数") || strings.Contains(text, "比例") || strings.Contains(text, "占比") || strings.Contains(text, "率") || strings.Contains(lower, "number") {
		return "number"
	}
	if strings.Contains(text, "状态") || strings.Contains(text, "分类") || strings.Contains(text, "类型") || strings.Contains(text, "选择") || strings.Contains(text, "下拉") || strings.Contains(text, "人员") || strings.Contains(text, "部门") || strings.Contains(lower, "select") {
		return "select"
	}
	return "input"
}

func normalizeWritePRDFormFields(fields []writePRDFormField) {
	for i := range fields {
		fields[i].Name = strings.TrimSpace(fields[i].Name)
		fields[i].Field = strings.TrimSpace(fields[i].Field)
		if fields[i].Name == "" {
			fields[i].Name = fields[i].Field
		}
		fields[i].Field = ""
		fields[i].Type = strings.TrimSpace(fields[i].Type)
		fields[i].Default = strings.TrimSpace(fields[i].Default)
		fields[i].Desc = strings.TrimSpace(fields[i].Desc)
		fields[i].Description = strings.TrimSpace(fields[i].Description)
		if fields[i].Desc == "" {
			fields[i].Desc = fields[i].Description
		}
		fields[i].Description = ""
		fields[i].Default = ""
	}
}

func normalizeWritePRDAcceptanceCases(cases []writePRDAcceptanceCase) []writePRDAcceptanceCase {
	out := make([]writePRDAcceptanceCase, 0, len(cases))
	for _, item := range cases {
		item.Name = strings.TrimSpace(item.Name)
		item.Action = strings.TrimSpace(item.Action)
		item.Expected = strings.TrimSpace(item.Expected)
		out = append(out, item)
	}
	return out
}

func normalizeWritePRDConfirmation(confirmation writePRDConfirmation) writePRDConfirmation {
	confirmation.Question = strings.TrimSpace(confirmation.Question)
	if confirmation.Question == "" {
		confirmation.Question = "请确认是否按以上 PRD 创建目录和生成代码。确认后我再开始创建目录、写 Go 文件并 build。"
	}
	return confirmation
}

func validateWritePRDResult(result writePRDResultData) []string {
	var issues []string
	if result.Project.Name == "" {
		issues = append(issues, "project.name 不能为空")
	}
	if result.Project.Code == "" {
		issues = append(issues, "project.code 不能为空")
	}
	if result.Project.CreateNewDirectory && result.Project.Reason == "" {
		issues = append(issues, "project.reason 不能为空，需要说明为什么创建新目录")
	}
	if !result.Project.CreateNewDirectory && result.Project.TargetDirectory == "" {
		issues = append(issues, "create_new_directory=false 时 project.target_directory 不能为空")
	}
	if len(result.Functions) == 0 {
		issues = append(issues, "functions 至少需要 1 个功能")
	}
	modelsByName := map[string]writePRDModel{}
	if len(result.Models) == 0 {
		issues = append(issues, "models 至少需要 1 个数据模型，PRD 预览和后续实现必须以模型字段为准")
	}
	for idx, model := range result.Models {
		prefix := fmt.Sprintf("models[%d]", idx)
		issues = append(issues, validateWritePRDModel(prefix, model)...)
		if model.Name != "" {
			modelsByName[model.Name] = model
		}
		if model.DisplayName != "" {
			modelsByName[model.DisplayName] = model
		}
		if model.TableName != "" {
			modelsByName[model.TableName] = model
		}
	}
	for idx, fn := range result.Functions {
		prefix := fmt.Sprintf("functions[%d]", idx)
		issues = append(issues, validateWritePRDFunction(prefix, fn)...)
		if fn.Type == "table" || fn.Type == "form" {
			if fn.Model == "" {
				issues = append(issues, prefix+".model 不能为空，table/form 必须绑定一个 PRD model")
			} else if _, ok := modelsByName[fn.Model]; !ok {
				issues = append(issues, prefix+".model 必须引用 models 中已定义的模型")
			}
		}
		if fn.Type == "chart" && fn.Model != "" {
			if _, ok := modelsByName[fn.Model]; !ok {
				issues = append(issues, prefix+".model 必须引用 models 中已定义的模型")
			}
		}
	}
	issues = append(issues, validateWritePRDNoRedundantCRUDForms(result.Functions)...)
	for idx, item := range result.AcceptanceCases {
		prefix := fmt.Sprintf("acceptance_cases[%d]", idx)
		if item.Name == "" {
			issues = append(issues, prefix+".name 不能为空")
		}
		if item.Action == "" {
			issues = append(issues, prefix+".action 不能为空")
		}
		if item.Expected == "" {
			issues = append(issues, prefix+".expected 不能为空")
		}
	}
	return issues
}

func validateWritePRDNoRedundantCRUDForms(functions []writePRDFunction) []string {
	var issues []string
	writableTableByModel := map[string]writePRDFunction{}
	for _, fn := range functions {
		if fn.Type != "table" || fn.Model == "" || !writePRDTableHasCRUD(fn.Table) {
			continue
		}
		writableTableByModel[fn.Model] = fn
	}
	if len(writableTableByModel) == 0 {
		return nil
	}
	for idx, fn := range functions {
		if fn.Type != "form" || fn.Model == "" {
			continue
		}
		tableFn, ok := writableTableByModel[fn.Model]
		if !ok {
			continue
		}
		text := strings.ToLower(strings.Join([]string{fn.Title, fn.Route, fn.Description}, " "))
		if !writePRDLooksLikeCreateSubmitForm(text) {
			continue
		}
		if writePRDFormHasExplicitDifference(text) {
			continue
		}
		issues = append(issues, fmt.Sprintf(
			"functions[%d] 看起来是重复的提交/新增 Form：同一 model %q 已有可新增/编辑的 Table %q。Table 自带 CRUD；只有外部/匿名/客户自助、批量导入、文件解析、计算生成、支付结算、审批流、跨多表事务、只提交不编辑等明确差异时才保留 Form，并必须在 description 说明差异。",
			idx, fn.Model, tableFn.Route,
		))
	}
	return issues
}

func writePRDTableHasCRUD(table *writePRDTable) bool {
	if table == nil || table.ReadOnly {
		return false
	}
	text := strings.ToLower(strings.Join(append([]string{table.Capability}, table.Operations...), " "))
	return writePRDContainsAny(text, []string{"新增", "新建", "创建", "编辑", "修改", "删除", "add", "create", "edit", "update", "delete", "crud"})
}

func writePRDLooksLikeCreateSubmitForm(text string) bool {
	return writePRDContainsAny(text, []string{"提交", "新增", "新建", "创建", "录入", "填报", "submit", "create", "add"})
}

func writePRDFormHasExplicitDifference(text string) bool {
	return writePRDContainsAny(text, []string{
		"外部", "匿名", "客户自助", "公开", "游客", "移动端", "只提交", "禁止编辑", "不允许编辑",
		"批量", "导入", "文件", "解析", "计算", "生成", "支付", "结算", "审批", "跨表", "多表", "事务",
		"问卷", "投票", "收集", "回调", "专门结果", "返回结果",
		"external", "anonymous", "import", "parse", "calculate", "generate", "payment", "approval",
	})
}

func writePRDContainsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func validateWritePRDModel(prefix string, model writePRDModel) []string {
	var issues []string
	if model.Name == "" {
		issues = append(issues, prefix+".name 不能为空")
	}
	if len(model.Fields) == 0 {
		issues = append(issues, prefix+".fields 至少需要 1 个字段")
	}
	for idx, field := range model.Fields {
		fieldPrefix := fmt.Sprintf("%s.fields[%d]", prefix, idx)
		if field.Name == "" {
			issues = append(issues, fieldPrefix+".name 不能为空")
		}
		if field.Widget == "" {
			issues = append(issues, fieldPrefix+".widget 不能为空；字段必须说明控件类型，完全隐藏字段才可写 widget:\"-\" 对应值")
		}
	}
	return issues
}

func validateWritePRDFunction(prefix string, fn writePRDFunction) []string {
	var issues []string
	if fn.Title == "" {
		issues = append(issues, prefix+".title 不能为空")
	}
	if fn.Route == "" {
		issues = append(issues, prefix+".route 不能为空")
	}
	switch fn.Type {
	case "table":
		if fn.Form != nil || fn.Chart != nil {
			issues = append(issues, prefix+" type=table 时只能提供 table 结构，不要混入 form/chart")
		}
		if !strings.HasSuffix(fn.Route, ".table") {
			issues = append(issues, prefix+".route 必须以 .table 结尾")
		}
		issues = append(issues, validateWritePRDTable(prefix+".table", fn.Table)...)
	case "form":
		if fn.Table != nil || fn.Chart != nil {
			issues = append(issues, prefix+" type=form 时只能提供 form 结构，不要混入 table/chart")
		}
		if !strings.HasSuffix(fn.Route, ".form") {
			issues = append(issues, prefix+".route 必须以 .form 结尾")
		}
		if fn.Method != "" && fn.Method != "POST" {
			issues = append(issues, prefix+".method 建议为 POST")
		}
		issues = append(issues, validateWritePRDForm(prefix+".form", fn.Form)...)
	case "chart":
		if fn.Table != nil || fn.Form != nil {
			issues = append(issues, prefix+" type=chart 时只能提供 chart 结构，不要混入 table/form")
		}
		if !strings.HasSuffix(fn.Route, ".chart") {
			issues = append(issues, prefix+".route 必须以 .chart 结尾")
		}
		issues = append(issues, validateWritePRDChart(prefix+".chart", fn.Chart)...)
	default:
		issues = append(issues, prefix+".type 只允许 table、form、chart")
	}
	return issues
}

func validateWritePRDTable(prefix string, table *writePRDTable) []string {
	if table == nil {
		return []string{prefix + " 不能为空，type=table 必须提供 table 结构"}
	}
	var issues []string
	if len(table.Columns) == 0 {
		issues = append(issues, prefix+".columns 至少需要 1 个业务列")
	}
	if len(table.SampleRows) == 0 {
		issues = append(issues, prefix+".sample_rows 至少需要 1 行示例数据")
	}
	issues = append(issues, validateWritePRDFormFields(prefix+".request_fields", table.RequestFields)...)
	return issues
}

func validateWritePRDForm(prefix string, form *writePRDForm) []string {
	if form == nil {
		return []string{prefix + " 不能为空，type=form 必须提供 form 结构"}
	}
	var issues []string
	if len(form.RequestFields) == 0 {
		issues = append(issues, prefix+".request_fields 至少需要 1 个请求字段")
	}
	issues = append(issues, validateWritePRDFormFields(prefix+".request_fields", form.RequestFields)...)
	for idx, field := range form.ResponseFields {
		fieldPrefix := fmt.Sprintf("%s.response_fields[%d]", prefix, idx)
		if field.Name == "" {
			issues = append(issues, fieldPrefix+".name 不能为空")
		}
		if field.Type == "" {
			issues = append(issues, fieldPrefix+".type 不能为空")
		}
	}
	return issues
}

func validateWritePRDFormFields(prefix string, fields []writePRDFormField) []string {
	var issues []string
	for idx, field := range fields {
		fieldPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
		if field.Name == "" {
			issues = append(issues, fieldPrefix+".name 不能为空")
		}
		if field.Type == "" {
			issues = append(issues, fieldPrefix+".type 不能为空")
		}
	}
	return issues
}

func validateWritePRDPreviewFields(prefix string, fields writePRDPreviewFields) []string {
	var issues []string
	for idx, field := range fields {
		fieldPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
		if field.Name == "" {
			issues = append(issues, fieldPrefix+".name 不能为空")
		}
		if field.Type == "" {
			issues = append(issues, fieldPrefix+".type 不能为空")
		}
	}
	return issues
}

func validateWritePRDChart(prefix string, chart *writePRDChart) []string {
	if chart == nil {
		return []string{prefix + " 不能为空，type=chart 必须提供 chart 结构"}
	}
	var issues []string
	if chart.ChartType == "" {
		issues = append(issues, prefix+".chart_type 不能为空")
	} else if !isSupportedWritePRDChartType(chart.ChartType) {
		issues = append(issues, prefix+".chart_type 只允许 line、bar、pie、gauge 或对应 SDK 类型 LineChart/BarChart/PieChart/GaugeChart")
	}
	if chart.Dimension == "" {
		issues = append(issues, prefix+".dimension 不能为空")
	}
	if len(chart.Metrics) == 0 {
		issues = append(issues, prefix+".metrics 至少需要 1 个指标")
	}
	if len(chart.PreviewData) == 0 && len(chart.ResponseFields) == 0 {
		issues = append(issues, prefix+".preview_data 至少需要 1 行图表预览数据；兼容旧 PRD 时可用 response_fields 说明图表数据")
	}
	issues = append(issues, validateWritePRDPreviewFields(prefix+".filters", chart.Filters)...)
	issues = append(issues, validateWritePRDFormFields(prefix+".request_fields", chart.RequestFields)...)
	for idx, field := range chart.ResponseFields {
		fieldPrefix := fmt.Sprintf("%s.response_fields[%d]", prefix, idx)
		if field.Name == "" {
			issues = append(issues, fieldPrefix+".name 不能为空")
		}
		if field.Type == "" {
			issues = append(issues, fieldPrefix+".type 不能为空")
		}
	}
	for idx, item := range chart.Summary {
		itemPrefix := fmt.Sprintf("%s.summary[%d]", prefix, idx)
		if item.Name == "" {
			issues = append(issues, itemPrefix+".name 不能为空")
		}
		if item.Value == nil {
			issues = append(issues, itemPrefix+".value 不能为空")
		}
	}
	return issues
}

func isSupportedWritePRDChartType(chartType string) bool {
	switch strings.ToLower(strings.TrimSpace(chartType)) {
	case "line", "linechart", "bar", "barchart", "pie", "piechart", "gauge", "gaugechart":
		return true
	default:
		return false
	}
}
