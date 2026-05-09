package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	workspaceprd "github.com/ai-agent-os/ai-agent-os/core/agent-server/workspace/prd"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type WritePRDTool struct{}

// PRD v2 只保留大模型稳定可输出、前端稳定可预览的轻结构。
// 业务资源直接拆成 tables/forms/charts，再由 workflow 表达用户展示和操作顺序；字段只写简单 widget 类型，
// 选项、默认值、范围、数据来源、计算规则都放进自然语言 desc，不再兼容旧 models/functions/widget tag。
const (
	maxWritePRDTableExamples = workspaceprd.MaxTableExamples
	maxWritePRDChartExamples = workspaceprd.MaxChartExamples
)

type writePRDArgs struct {
	Project  writePRDProject        `json:"project" schema_desc:"项目和目录信息" schema_required:"true"`
	Tables   []writePRDTable        `json:"tables,omitempty" schema_desc:"业务数据表和表格页语义；每个 table 直接包含 fields、search_fields、handlers、examples；纯处理型工具可不填"`
	Forms    []writePRDForm         `json:"forms,omitempty" schema_desc:"独立提交入口；只描述 target_table、request_fields、response_fields 和 example"`
	Charts   []writePRDChart        `json:"charts,omitempty" schema_desc:"统计图表；只描述 source_table、chart_type、dimension、metrics、filters、examples"`
	Workflow []writePRDWorkflowStep `json:"workflow" schema_desc:"用户展示和操作顺序；只写 type/ref，不承载业务细节" schema_required:"true"`
	Rules    []string               `json:"rules,omitempty" schema_desc:"业务规则、计算口径、状态流转等自然语言规则"`
}

type writePRDProject struct {
	Name    string `json:"name" schema_desc:"项目显示名称，例如：NPS 净推荐值系统" schema_required:"true"`
	Code    string `json:"code" schema_desc:"目录 code，小写下划线，例如：nps" schema_required:"true"`
	Summary string `json:"summary" schema_desc:"业务目标一句话说明" schema_required:"true"`
}

type writePRDField struct {
	Name     string `json:"name" schema_desc:"用户可见字段名，例如：问卷标题" schema_required:"true"`
	Widget   string `json:"widget" schema_desc:"简单组件类型，例如 input、text_area、number、select、datetime；不要写 widget tag" schema_required:"true"`
	Required bool   `json:"required" schema_desc:"是否必填" schema_required:"true"`
	Desc     string `json:"desc" schema_desc:"用户能看懂的字段说明；选项、默认值、范围、数据来源、计算规则都写在这里" schema_required:"true"`
	Hide     string `json:"hide,omitempty" schema_desc:"展示范围，例如 create,update；没有就不填"`
}

type writePRDTable struct {
	Name         string                   `json:"name" schema_desc:"业务数据表名，例如：NPS评分记录" schema_required:"true"`
	Title        string                   `json:"title" schema_desc:"表格页标题，例如：NPS评分记录" schema_required:"true"`
	Desc         string                   `json:"desc" schema_desc:"表格业务说明" schema_required:"true"`
	Fields       []writePRDField          `json:"fields" schema_desc:"表字段；每项单行写 name/widget/required/desc/hide" schema_required:"true"`
	SearchFields []writePRDField          `json:"search_fields,omitempty" schema_desc:"搜索参数；只描述字段，不需要 handlers；大多数业务表默认加 创建开始时间/创建结束时间 两个 datetime 用于按记录创建时间范围查询，并加一个用户筛选字段：优先用提交人/处理人/评分人/申请人等业务用户，没有明确业务用户时用创建人"`
	Handlers     []string                 `json:"handlers,omitempty" schema_desc:"表格行操作能力，例如 OnTableAddRow、OnTableUpdateRow、OnTableDeleteRow；只查询时填空数组"`
	Examples     []map[string]interface{} `json:"examples" schema_desc:"1-3 条用户可见示例记录；key 必须是 fields 中的业务字段名" schema_required:"true"`
}

type writePRDForm struct {
	Name           string              `json:"name" schema_desc:"表单名称，例如：NPS评分提交" schema_required:"true"`
	Desc           string              `json:"desc" schema_desc:"表单业务说明和边界" schema_required:"true"`
	TargetTable    string              `json:"target_table,omitempty" schema_desc:"提交后写入或影响的业务表名；纯文件处理、转换、计算类表单可不填"`
	RequestFields  []writePRDField     `json:"request_fields" schema_desc:"请求字段；每项单行写 name/widget/required/desc/hide" schema_required:"true"`
	ResponseFields []writePRDField     `json:"response_fields,omitempty" schema_desc:"提交后展示字段；没有就填空数组"`
	Example        writePRDFormExample `json:"example" schema_desc:"一次提交和返回示例" schema_required:"true"`
}

type writePRDFormExample struct {
	Request  map[string]interface{} `json:"request" schema_desc:"表单提交示例；key 必须是 request_fields 的业务字段名" schema_required:"true"`
	Response map[string]interface{} `json:"response" schema_desc:"表单返回示例；key 必须是 response_fields 的业务字段名；没有响应字段时可为空对象" schema_required:"true"`
}

type writePRDChart struct {
	Name        string                   `json:"name" schema_desc:"图表名称，例如：NPS趋势分析" schema_required:"true"`
	Desc        string                   `json:"desc" schema_desc:"图表业务说明" schema_required:"true"`
	SourceTable string                   `json:"source_table" schema_desc:"统计来源业务表名" schema_required:"true"`
	ChartType   string                   `json:"chart_type" schema_desc:"图表类型；只允许 line、bar、pie" schema_enum:"line,bar,pie" schema_required:"true"`
	Dimension   string                   `json:"dimension" schema_desc:"维度字段名，例如：日期、评分类型；如果写成 日期（按天/周/月），工具会归一为 日期" schema_required:"true"`
	Metrics     []string                 `json:"metrics" schema_desc:"指标，例如：NPS分数、评分数量" schema_required:"true"`
	Filters     []writePRDField          `json:"filters,omitempty" schema_desc:"图表筛选条件；每项单行写 name/widget/required/desc/hide"`
	Examples    []map[string]interface{} `json:"examples" schema_desc:"1-12 条图表示例数据；推荐写 {dimension: 维度值, metrics: {指标名: 指标值}}，工具会归一为前端预览行" schema_required:"true"`
}

type writePRDWorkflowStep struct {
	Type string `json:"type" schema_desc:"功能类型，只允许 table、form、chart" schema_enum:"table,form,chart" schema_required:"true"`
	Ref  string `json:"ref" schema_desc:"引用 tables/forms/charts 中的 name" schema_required:"true"`
}

type writePRDResultData struct {
	Kind          string                 `json:"kind" schema_desc:"固定为 agent_app_prd" schema_required:"true"`
	SchemaVersion string                 `json:"schema_version" schema_desc:"固定为 prd.v2" schema_required:"true"`
	Project       writePRDProject        `json:"project" schema_desc:"项目和目录信息" schema_required:"true"`
	Tables        []writePRDTable        `json:"tables,omitempty" schema_desc:"业务数据表和表格页语义"`
	Forms         []writePRDForm         `json:"forms,omitempty" schema_desc:"独立提交入口"`
	Charts        []writePRDChart        `json:"charts,omitempty" schema_desc:"统计图表"`
	Workflow      []writePRDWorkflowStep `json:"workflow" schema_desc:"用户展示和操作顺序" schema_required:"true"`
	Rules         []string               `json:"rules,omitempty" schema_desc:"业务规则"`
	Interaction   *writePRDInteraction   `json:"interaction,omitempty" schema_desc:"PRD 交互状态和允许动作；用于前端固定展示确认入口"`
	Issues        []string               `json:"issues,omitempty" schema_desc:"PRD 结构问题；非空时本次工具返回错误"`
}

var writePRDToolDef = toolDefinitionWithOutput[writePRDArgs, structuredToolResultSchema[writePRDResultData]](
	"write_prd",
	workspaceprd.ToolDescription(),
)

func (t *WritePRDTool) Definition() dto.ToolDef {
	return writePRDToolDef
}

func (t *WritePRDTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	rawIssues := validateWritePRDRawShape(call.Args)
	args, err := decodeToolArgs[writePRDArgs](call.Args)
	if err != nil {
		return toolResult("write_prd 参数解析失败: "+err.Error(), true)
	}
	result := buildWritePRDResult(args)
	result.Issues = append(rawIssues, result.Issues...)
	if len(result.Issues) > 0 {
		return toolResultWithStructuredData(result, true, writePRDIssueContent(result.Issues))
	}
	result.Interaction = pendingPRDInteraction()
	return toolResultWithStructuredData(result, false, "PRD 已生成，请确认后我再进入开发。看不到按钮也可以直接回复：确认 PRD / 修改 PRD：xxx / 取消 PRD。")
}

func validateWritePRDRawShape(args map[string]interface{}) []string {
	raw := normalizeWritePRDRawArgs(args)
	if raw == nil {
		return nil
	}
	var issues []string
	issues = append(issues, validateWritePRDRawObjectKeys("$", raw, writePRDAllowedTopLevelKeys())...)
	issues = append(issues, validateWritePRDRawObjectKeys("project", raw["project"], writePRDAllowedProjectKeys())...)
	issues = append(issues, validateWritePRDRawObjectArrayKeys("tables", raw["tables"], writePRDAllowedTableKeys())...)
	issues = append(issues, validateWritePRDRawObjectArrayKeys("forms", raw["forms"], writePRDAllowedFormKeys())...)
	issues = append(issues, validateWritePRDRawObjectArrayKeys("charts", raw["charts"], writePRDAllowedChartKeys())...)
	issues = append(issues, validateWritePRDRawObjectArrayKeys("workflow", raw["workflow"], writePRDAllowedWorkflowKeys())...)
	issues = append(issues, validateWritePRDRawFieldArrays("tables", raw["tables"], []string{"fields", "search_fields"})...)
	issues = append(issues, validateWritePRDRawFieldArrays("forms", raw["forms"], []string{"request_fields", "response_fields"})...)
	issues = append(issues, validateWritePRDRawFieldArrays("charts", raw["charts"], []string{"filters"})...)
	return issues
}

func buildWritePRDResult(args writePRDArgs) writePRDResultData {
	result := writePRDResultData{
		Kind:          workspaceprd.Kind,
		SchemaVersion: workspaceprd.SchemaVersion,
		Project:       normalizeWritePRDProject(args.Project),
		Tables:        normalizeWritePRDTables(args.Tables),
		Forms:         normalizeWritePRDForms(args.Forms),
		Charts:        normalizeWritePRDCharts(args.Charts),
		Workflow:      normalizeWritePRDWorkflow(args.Workflow),
		Rules:         trimStringSlice(args.Rules),
	}
	result.Issues = validateWritePRDResult(result)
	return result
}

func writePRDIssueContent(issues []string) string {
	if len(issues) == 0 {
		return "write_prd 参数不完整，先修正 PRD 结构后再继续。"
	}
	limit := len(issues)
	if limit > 8 {
		limit = 8
	}
	var out strings.Builder
	out.WriteString("write_prd 参数不完整，先修正 PRD 结构后再继续：")
	for i := 0; i < limit; i++ {
		out.WriteString("\n- ")
		out.WriteString(issues[i])
	}
	if len(issues) > limit {
		out.WriteString(fmt.Sprintf("\n- 还有 %d 个问题见 structured data issues", len(issues)-limit))
	}
	return out.String()
}

func normalizeWritePRDProject(project writePRDProject) writePRDProject {
	project.Name = strings.TrimSpace(project.Name)
	project.Code = strings.TrimSpace(project.Code)
	project.Summary = strings.TrimSpace(project.Summary)
	return project
}

func normalizeWritePRDTables(tables []writePRDTable) []writePRDTable {
	out := make([]writePRDTable, 0, len(tables))
	for _, table := range tables {
		table.Name = strings.TrimSpace(table.Name)
		table.Title = strings.TrimSpace(table.Title)
		if table.Title == "" {
			table.Title = table.Name
		}
		table.Desc = strings.TrimSpace(table.Desc)
		normalizeWritePRDFields(table.Fields)
		normalizeWritePRDFields(table.SearchFields)
		table.Handlers = trimStringSlice(table.Handlers)
		table.Examples = normalizeWritePRDExamples(table.Examples)
		out = append(out, table)
	}
	return out
}

func normalizeWritePRDForms(forms []writePRDForm) []writePRDForm {
	out := make([]writePRDForm, 0, len(forms))
	for _, form := range forms {
		form.Name = strings.TrimSpace(form.Name)
		form.Desc = strings.TrimSpace(form.Desc)
		form.TargetTable = strings.TrimSpace(form.TargetTable)
		normalizeWritePRDFields(form.RequestFields)
		normalizeWritePRDFields(form.ResponseFields)
		form.Example.Request = normalizeWritePRDExample(form.Example.Request)
		form.Example.Response = normalizeWritePRDExample(form.Example.Response)
		out = append(out, form)
	}
	return out
}

func normalizeWritePRDCharts(charts []writePRDChart) []writePRDChart {
	out := make([]writePRDChart, 0, len(charts))
	for _, chart := range charts {
		chart.Name = strings.TrimSpace(chart.Name)
		chart.Desc = strings.TrimSpace(chart.Desc)
		chart.SourceTable = strings.TrimSpace(chart.SourceTable)
		chart.ChartType = strings.ToLower(strings.TrimSpace(chart.ChartType))
		chart.Dimension = workspaceprd.NormalizeChartDimension(chart.Dimension)
		chart.Metrics = trimStringSlice(chart.Metrics)
		normalizeWritePRDFields(chart.Filters)
		chart.Examples = normalizeWritePRDChartExamples(chart.Dimension, chart.Examples)
		out = append(out, chart)
	}
	return out
}

func normalizeWritePRDWorkflow(workflow []writePRDWorkflowStep) []writePRDWorkflowStep {
	out := make([]writePRDWorkflowStep, 0, len(workflow))
	for _, step := range workflow {
		step.Type = strings.ToLower(strings.TrimSpace(step.Type))
		step.Ref = strings.TrimSpace(step.Ref)
		out = append(out, step)
	}
	return out
}

func normalizeWritePRDFields(fields []writePRDField) {
	for i := range fields {
		fields[i].Name = strings.TrimSpace(fields[i].Name)
		fields[i].Widget = strings.ToLower(strings.TrimSpace(fields[i].Widget))
		fields[i].Desc = strings.TrimSpace(fields[i].Desc)
		fields[i].Hide = strings.TrimSpace(fields[i].Hide)
	}
}

func normalizeWritePRDExamples(examples []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(examples))
	for _, example := range examples {
		out = append(out, normalizeWritePRDExample(example))
	}
	return out
}

func normalizeWritePRDChartExamples(dimension string, examples []map[string]interface{}) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(examples))
	for _, example := range examples {
		out = append(out, normalizeWritePRDChartExample(dimension, example))
	}
	return out
}

func normalizeWritePRDChartExample(dimension string, example map[string]interface{}) map[string]interface{} {
	normalized := normalizeWritePRDExample(example)
	if len(normalized) == 0 {
		return normalized
	}
	dimensionValue, hasDimension := normalized["dimension"]
	metricsValue, hasMetrics := normalized["metrics"]
	if !hasDimension || !hasMetrics {
		return normalized
	}

	out := make(map[string]interface{}, len(normalized)+4)
	if dimension != "" {
		out[dimension] = normalizeWritePRDScalar(dimensionValue)
	}
	if metrics, ok := metricsValue.(map[string]interface{}); ok {
		for _, key := range sortedWritePRDMapKeys(metrics) {
			out[key] = normalizeWritePRDScalar(metrics[key])
		}
	}
	for key, value := range normalized {
		if key == "dimension" || key == "metrics" {
			continue
		}
		out[key] = normalizeWritePRDScalar(value)
	}
	return out
}

func normalizeWritePRDExample(example map[string]interface{}) map[string]interface{} {
	if example == nil {
		return nil
	}
	out := make(map[string]interface{}, len(example))
	for key, value := range example {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = normalizeWritePRDScalar(value)
	}
	return out
}

func normalizeWritePRDScalar(value interface{}) interface{} {
	if str, ok := value.(string); ok {
		return strings.TrimSpace(str)
	}
	return value
}

func normalizeWritePRDRawArgs(args map[string]interface{}) map[string]interface{} {
	if len(args) == 0 {
		return nil
	}
	data, err := json.Marshal(args)
	if err != nil {
		return args
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return args
	}
	return out
}

func validateWritePRDRawObjectKeys(prefix string, value interface{}, allowed map[string]struct{}) []string {
	object, ok := value.(map[string]interface{})
	if !ok {
		return nil
	}
	var issues []string
	for _, key := range sortedWritePRDMapKeys(object) {
		if _, ok := allowed[key]; !ok {
			issues = append(issues, prefix+"."+key+" 不是 PRD v2 字段")
		}
	}
	return issues
}

func validateWritePRDRawObjectArrayKeys(prefix string, value interface{}, allowed map[string]struct{}) []string {
	items, ok := value.([]interface{})
	if !ok {
		return nil
	}
	var issues []string
	for idx, item := range items {
		issues = append(issues, validateWritePRDRawObjectKeys(fmt.Sprintf("%s[%d]", prefix, idx), item, allowed)...)
	}
	return issues
}

func validateWritePRDRawFieldArrays(parentPrefix string, parentValue interface{}, fieldKeys []string) []string {
	items, ok := parentValue.([]interface{})
	if !ok {
		return nil
	}
	var issues []string
	for idx, item := range items {
		object, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for _, fieldKey := range fieldKeys {
			issues = append(issues, validateWritePRDRawObjectArrayKeys(fmt.Sprintf("%s[%d].%s", parentPrefix, idx, fieldKey), object[fieldKey], writePRDAllowedFieldKeys())...)
		}
	}
	return issues
}

func validateWritePRDResult(result writePRDResultData) []string {
	var issues []string
	if result.Project.Name == "" {
		issues = append(issues, "project.name 不能为空")
	}
	if result.Project.Code == "" {
		issues = append(issues, "project.code 不能为空")
	}
	if result.Project.Summary == "" {
		issues = append(issues, "project.summary 不能为空")
	}
	if len(result.Tables)+len(result.Forms)+len(result.Charts) == 0 {
		issues = append(issues, "tables/forms/charts 至少需要 1 个业务资源")
	}
	if len(result.Workflow) == 0 {
		issues = append(issues, "workflow 至少需要 1 个功能引用")
	}

	tableNames := map[string]struct{}{}
	formNames := map[string]struct{}{}
	chartNames := map[string]struct{}{}

	for idx, table := range result.Tables {
		prefix := fmt.Sprintf("tables[%d]", idx)
		issues = append(issues, validateWritePRDTable(prefix, table)...)
		if table.Name != "" {
			if _, ok := tableNames[table.Name]; ok {
				issues = append(issues, prefix+".name 重复："+table.Name)
			}
			tableNames[table.Name] = struct{}{}
		}
	}
	for idx, form := range result.Forms {
		prefix := fmt.Sprintf("forms[%d]", idx)
		issues = append(issues, validateWritePRDForm(prefix, form, tableNames)...)
		if form.Name != "" {
			if _, ok := formNames[form.Name]; ok {
				issues = append(issues, prefix+".name 重复："+form.Name)
			}
			formNames[form.Name] = struct{}{}
		}
	}
	for idx, chart := range result.Charts {
		prefix := fmt.Sprintf("charts[%d]", idx)
		issues = append(issues, validateWritePRDChart(prefix, chart, tableNames)...)
		if chart.Name != "" {
			if _, ok := chartNames[chart.Name]; ok {
				issues = append(issues, prefix+".name 重复："+chart.Name)
			}
			chartNames[chart.Name] = struct{}{}
		}
	}
	for idx, step := range result.Workflow {
		issues = append(issues, validateWritePRDWorkflowStep(fmt.Sprintf("workflow[%d]", idx), step, tableNames, formNames, chartNames)...)
	}
	issues = append(issues, validateWritePRDWorkflowOrder(result.Workflow, result.Tables, result.Forms, result.Charts)...)
	return issues
}

func validateWritePRDTable(prefix string, table writePRDTable) []string {
	var issues []string
	if table.Name == "" {
		issues = append(issues, prefix+".name 不能为空")
	}
	if table.Title == "" {
		issues = append(issues, prefix+".title 不能为空")
	}
	if table.Desc == "" {
		issues = append(issues, prefix+".desc 不能为空")
	}
	if len(table.Fields) == 0 {
		issues = append(issues, prefix+".fields 至少需要 1 个字段")
	}
	issues = append(issues, validateWritePRDFields(prefix+".fields", table.Fields)...)
	issues = append(issues, validateWritePRDFields(prefix+".search_fields", table.SearchFields)...)
	issues = append(issues, validateWritePRDSearchFieldsAlign(prefix+".search_fields", table.Fields, table.SearchFields)...)
	issues = append(issues, validateWritePRDHandlers(prefix+".handlers", table.Handlers)...)
	if len(table.Examples) == 0 {
		issues = append(issues, prefix+".examples 至少需要 1 条示例数据")
	}
	if len(table.Examples) > maxWritePRDTableExamples {
		issues = append(issues, fmt.Sprintf("%s.examples 最多写 %d 条示例数据", prefix, maxWritePRDTableExamples))
	}
	fieldNames := writePRDFieldNameSet(table.Fields)
	for idx, example := range table.Examples {
		issues = append(issues, validateWritePRDExampleKeys(fmt.Sprintf("%s.examples[%d]", prefix, idx), example, fieldNames)...)
	}
	return issues
}

func validateWritePRDForm(prefix string, form writePRDForm, tableNames map[string]struct{}) []string {
	var issues []string
	if form.Name == "" {
		issues = append(issues, prefix+".name 不能为空")
	}
	if form.Desc == "" {
		issues = append(issues, prefix+".desc 不能为空")
	}
	if form.TargetTable != "" {
		if _, ok := tableNames[form.TargetTable]; !ok {
			issues = append(issues, prefix+".target_table 必须引用 tables 中已定义的 name")
		}
	}
	if len(form.RequestFields) == 0 {
		issues = append(issues, prefix+".request_fields 至少需要 1 个请求字段")
	}
	issues = append(issues, validateWritePRDFields(prefix+".request_fields", form.RequestFields)...)
	issues = append(issues, validateWritePRDFields(prefix+".response_fields", form.ResponseFields)...)
	issues = append(issues, validateWritePRDExampleKeys(prefix+".example.request", form.Example.Request, writePRDFieldNameSet(form.RequestFields))...)
	if len(form.ResponseFields) == 0 {
		if len(form.Example.Response) > 0 {
			issues = append(issues, prefix+".example.response 有返回示例时必须先声明 response_fields")
		}
	} else {
		issues = append(issues, validateWritePRDExampleKeys(prefix+".example.response", form.Example.Response, writePRDFieldNameSet(form.ResponseFields))...)
	}
	return issues
}

func validateWritePRDChart(prefix string, chart writePRDChart, tableNames map[string]struct{}) []string {
	var issues []string
	if chart.Name == "" {
		issues = append(issues, prefix+".name 不能为空")
	}
	if chart.Desc == "" {
		issues = append(issues, prefix+".desc 不能为空")
	}
	if chart.SourceTable == "" {
		issues = append(issues, prefix+".source_table 不能为空")
	} else if _, ok := tableNames[chart.SourceTable]; !ok {
		issues = append(issues, prefix+".source_table 必须引用 tables 中已定义的 name")
	}
	if !isSupportedWritePRDChartType(chart.ChartType) {
		issues = append(issues, prefix+".chart_type 只允许 line、bar、pie")
	}
	if chart.Dimension == "" {
		issues = append(issues, prefix+".dimension 不能为空")
	}
	if len(chart.Metrics) == 0 {
		issues = append(issues, prefix+".metrics 至少需要 1 个指标")
	}
	issues = append(issues, validateWritePRDFields(prefix+".filters", chart.Filters)...)
	if len(chart.Examples) == 0 {
		issues = append(issues, prefix+".examples 至少需要 1 条示例数据")
	}
	if len(chart.Examples) > maxWritePRDChartExamples {
		issues = append(issues, fmt.Sprintf("%s.examples 最多写 %d 条示例数据", prefix, maxWritePRDChartExamples))
	}
	expected := append([]string{chart.Dimension}, chart.Metrics...)
	for idx, example := range chart.Examples {
		issues = append(issues, validateWritePRDExampleExactKeys(fmt.Sprintf("%s.examples[%d]", prefix, idx), example, expected)...)
	}
	return issues
}

func validateWritePRDWorkflowStep(prefix string, step writePRDWorkflowStep, tableNames, formNames, chartNames map[string]struct{}) []string {
	var issues []string
	if step.Type == "" {
		issues = append(issues, prefix+".type 不能为空")
	}
	if step.Ref == "" {
		issues = append(issues, prefix+".ref 不能为空")
	}
	var refs map[string]struct{}
	switch step.Type {
	case "table":
		refs = tableNames
	case "form":
		refs = formNames
	case "chart":
		refs = chartNames
	default:
		issues = append(issues, prefix+".type 只允许 table、form、chart")
		return issues
	}
	if step.Ref != "" {
		if _, ok := refs[step.Ref]; !ok {
			issues = append(issues, prefix+".ref 必须引用对应 tables/forms/charts 中已定义的 name")
		}
	}
	return issues
}

func validateWritePRDWorkflowOrder(workflow []writePRDWorkflowStep, tables []writePRDTable, forms []writePRDForm, charts []writePRDChart) []string {
	var issues []string
	workflowIndexes := map[string]int{}
	for idx, step := range workflow {
		if step.Type == "" || step.Ref == "" {
			continue
		}
		key := workspaceprd.WorkflowStepKey(step.Type, step.Ref)
		if first, ok := workflowIndexes[key]; ok {
			issues = append(issues, fmt.Sprintf("workflow[%d] 与 workflow[%d] 重复：%s/%s", idx, first, step.Type, step.Ref))
			continue
		}
		workflowIndexes[key] = idx
	}

	tableByName := map[string]writePRDTable{}
	for _, table := range tables {
		if table.Name != "" {
			tableByName[table.Name] = table
		}
	}
	for _, table := range tables {
		tableIndex, hasTableStep := workflowIndexes[workspaceprd.WorkflowStepKey("table", table.Name)]
		if !hasTableStep || len(table.Handlers) == 0 {
			continue
		}
		for _, form := range forms {
			formIndex, hasFormStep := workflowIndexes[workspaceprd.WorkflowStepKey("form", form.Name)]
			if hasFormStep && formIndex < tableIndex {
				issues = append(issues, fmt.Sprintf("workflow 顺序不合理：基础/配置表「%s」包含行操作，应排在表单「%s」前面", table.Name, form.Name))
			}
		}
	}

	for _, form := range forms {
		if form.TargetTable == "" {
			continue
		}
		targetTable, ok := tableByName[form.TargetTable]
		if !ok || len(targetTable.Handlers) > 0 {
			continue
		}
		formIndex, hasFormStep := workflowIndexes[workspaceprd.WorkflowStepKey("form", form.Name)]
		tableIndex, hasTableStep := workflowIndexes[workspaceprd.WorkflowStepKey("table", form.TargetTable)]
		if hasFormStep && hasTableStep && tableIndex < formIndex {
			issues = append(issues, fmt.Sprintf("workflow 顺序不合理：表单「%s」写入目标记录表「%s」，应先排表单，再排记录表", form.Name, form.TargetTable))
		}
	}

	for _, chart := range charts {
		chartIndex, hasChartStep := workflowIndexes[workspaceprd.WorkflowStepKey("chart", chart.Name)]
		tableIndex, hasTableStep := workflowIndexes[workspaceprd.WorkflowStepKey("table", chart.SourceTable)]
		if hasChartStep && hasTableStep && chartIndex < tableIndex {
			issues = append(issues, fmt.Sprintf("workflow 顺序不合理：图表「%s」统计来源表「%s」，应排在来源表后面", chart.Name, chart.SourceTable))
		}
	}
	return issues
}

func validateWritePRDFields(prefix string, fields []writePRDField) []string {
	var issues []string
	names := map[string]int{}
	for idx, field := range fields {
		fieldPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
		if field.Name == "" {
			issues = append(issues, fieldPrefix+".name 不能为空")
		} else if first, ok := names[field.Name]; ok {
			issues = append(issues, fmt.Sprintf("%s.name 与 %s[%d].name 重复：%s", fieldPrefix, prefix, first, field.Name))
		} else {
			names[field.Name] = idx
		}
		if field.Widget == "" {
			issues = append(issues, fieldPrefix+".widget 不能为空")
		} else if strings.Contains(field.Widget, ";") || strings.Contains(field.Widget, ":") {
			issues = append(issues, fieldPrefix+".widget 只能写简单组件类型，不要写 widget tag")
		} else if !isSupportedWritePRDWidget(field.Widget) {
			issues = append(issues, fieldPrefix+".widget 不支持："+field.Widget)
		} else if isWritePRDUserSemanticField(field.Name) && normalizeWritePRDWidget(field.Widget) != "user" {
			issues = append(issues, fieldPrefix+".widget 用户语义字段必须使用 user 组件："+field.Name)
		}
		if field.Desc == "" {
			issues = append(issues, fieldPrefix+".desc 不能为空；选项、默认值、范围、数据来源、计算规则都写进 desc")
		}
	}
	return issues
}

func validateWritePRDSearchFieldsAlign(prefix string, fields []writePRDField, searchFields []writePRDField) []string {
	var issues []string
	fieldWidgets := make(map[string]string, len(fields))
	for _, field := range fields {
		if field.Name == "" {
			continue
		}
		fieldWidgets[field.Name] = normalizeWritePRDWidget(field.Widget)
	}
	for idx, field := range searchFields {
		if field.Name == "" {
			continue
		}
		fieldName, fieldWidget, ok := writePRDSearchFieldBase(field.Name, fieldWidgets)
		if !ok {
			issues = append(issues, fmt.Sprintf("%s[%d].name 必须对齐 fields 中的字段；同名筛选直接用字段名，时间范围用 xxx开始时间/xxx结束时间 对齐 fields 中的 xxx时间 字段：%s", prefix, idx, field.Name))
			continue
		}
		searchWidget := normalizeWritePRDWidget(field.Widget)
		if searchWidget != "" && fieldWidget != "" && searchWidget != fieldWidget {
			issues = append(issues, fmt.Sprintf("%s[%d].widget 必须与 fields 中「%s」字段的 widget 对齐：got %s, want %s", prefix, idx, fieldName, searchWidget, fieldWidget))
		}
	}
	return issues
}

func writePRDSearchFieldBase(searchName string, fieldWidgets map[string]string) (string, string, bool) {
	searchName = strings.TrimSpace(searchName)
	if widget, ok := fieldWidgets[searchName]; ok {
		return searchName, widget, true
	}
	for _, suffix := range []string{"开始时间", "结束时间"} {
		if !strings.HasSuffix(searchName, suffix) {
			continue
		}
		base := strings.TrimSuffix(searchName, suffix)
		if base == "" {
			continue
		}
		fieldName := base + "时间"
		if widget, ok := fieldWidgets[fieldName]; ok {
			return fieldName, widget, true
		}
	}
	return "", "", false
}

func normalizeWritePRDWidget(widget string) string {
	widget = strings.ToLower(strings.TrimSpace(widget))
	if widget == "textarea" {
		return "text_area"
	}
	return widget
}

func isWritePRDUserSemanticField(name string) bool {
	switch strings.TrimSpace(name) {
	case "创建人", "提交人", "处理人", "评分人", "申请人", "审核人", "负责人", "预约人", "投票人", "发布人", "操作人", "办理人":
		return true
	default:
		return false
	}
}

func validateWritePRDHandlers(prefix string, handlers []string) []string {
	var issues []string
	seen := map[string]int{}
	for idx, handler := range handlers {
		itemPrefix := fmt.Sprintf("%s[%d]", prefix, idx)
		if handler == "" {
			issues = append(issues, itemPrefix+" 不能为空")
			continue
		}
		if !isSupportedWritePRDHandler(handler) {
			issues = append(issues, itemPrefix+" 只允许 OnTableAddRow、OnTableUpdateRow、OnTableDeleteRow")
		}
		if first, ok := seen[handler]; ok {
			issues = append(issues, fmt.Sprintf("%s 与 %s[%d] 重复：%s", itemPrefix, prefix, first, handler))
		} else {
			seen[handler] = idx
		}
	}
	return issues
}

func validateWritePRDExampleKeys(prefix string, example map[string]interface{}, allowed map[string]struct{}) []string {
	if len(example) == 0 {
		return []string{prefix + " 不能为空"}
	}
	var issues []string
	for _, key := range sortedWritePRDMapKeys(example) {
		if _, ok := allowed[key]; !ok {
			issues = append(issues, prefix+" 包含非业务字段 key："+key)
		}
	}
	return issues
}

func validateWritePRDExampleExactKeys(prefix string, example map[string]interface{}, expected []string) []string {
	allowed := make(map[string]struct{}, len(expected))
	for _, key := range expected {
		if key != "" {
			allowed[key] = struct{}{}
		}
	}
	var issues []string
	issues = append(issues, validateWritePRDExampleKeys(prefix, example, allowed)...)
	for _, key := range expected {
		if key == "" {
			continue
		}
		if _, ok := example[key]; !ok {
			issues = append(issues, prefix+" 缺少示例字段："+key)
		}
	}
	return issues
}

func writePRDFieldNameSet(fields []writePRDField) map[string]struct{} {
	out := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		if field.Name != "" {
			out[field.Name] = struct{}{}
		}
	}
	return out
}

func sortedWritePRDMapKeys(values map[string]interface{}) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func writePRDAllowedTopLevelKeys() map[string]struct{} {
	return workspaceprd.AllowedTopLevelKeys()
}

func writePRDAllowedProjectKeys() map[string]struct{} {
	return workspaceprd.AllowedProjectKeys()
}

func writePRDAllowedFieldKeys() map[string]struct{} {
	return workspaceprd.AllowedFieldKeys()
}

func writePRDAllowedTableKeys() map[string]struct{} {
	return workspaceprd.AllowedTableKeys()
}

func writePRDAllowedFormKeys() map[string]struct{} {
	return workspaceprd.AllowedFormKeys()
}

func writePRDAllowedChartKeys() map[string]struct{} {
	return workspaceprd.AllowedChartKeys()
}

func writePRDAllowedWorkflowKeys() map[string]struct{} {
	return workspaceprd.AllowedWorkflowKeys()
}

func isSupportedWritePRDWidget(widget string) bool {
	return workspaceprd.IsSupportedWidget(widget)
}

func isSupportedWritePRDHandler(handler string) bool {
	return workspaceprd.IsSupportedHandler(handler)
}

func isSupportedWritePRDChartType(chartType string) bool {
	return workspaceprd.IsSupportedChartType(chartType)
}
