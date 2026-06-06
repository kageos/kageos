package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
	"github.com/kageos/kageos/pkg/logger"
)

type BuildWorkspaceTool struct{}

type buildWorkspaceArgs struct{}

type buildWorkspaceResultData struct {
	Kind             string                     `json:"kind" schema_desc:"阶段产物类型" schema_required:"true"`
	Status           string                     `json:"status,omitempty" schema_desc:"构建状态：ok/error"`
	WorkspacePath    string                     `json:"workspace_path" schema_desc:"编译部署的工作空间路径" schema_required:"true"`
	User             string                     `json:"user" schema_desc:"工作空间用户" schema_required:"true"`
	App              string                     `json:"app" schema_desc:"应用 code" schema_required:"true"`
	OldVersion       string                     `json:"old_version,omitempty" schema_desc:"编译前版本"`
	NewVersion       string                     `json:"new_version,omitempty" schema_desc:"编译后版本"`
	GitCommitHash    string                     `json:"git_commit_hash,omitempty" schema_desc:"构建对应的 Git commit hash"`
	Warnings         []string                   `json:"warnings,omitempty" schema_desc:"非阻断构建告警"`
	Error            string                     `json:"error,omitempty" schema_desc:"构建失败摘要"`
	BuildDiagnostics *workspaceBuildDiagnostics `json:"build_diagnostics,omitempty" schema_desc:"构建失败诊断和修复策略"`
	Interaction      *workspaceStageInteraction `json:"interaction,omitempty" schema_desc:"构建成功后的测试交互状态"`
}

type workspaceBuildDiagnostics struct {
	Status        string                     `json:"status" schema_desc:"诊断状态：error/empty" schema_required:"true"`
	WorkspacePath string                     `json:"workspace_path,omitempty" schema_desc:"构建范围"`
	ErrorSummary  string                     `json:"error_summary,omitempty" schema_desc:"压缩后的错误摘要"`
	Categories    []string                   `json:"categories,omitempty" schema_desc:"错误类别"`
	Routers       []string                   `json:"routers,omitempty" schema_desc:"错误涉及的相对 router"`
	Files         []string                   `json:"files,omitempty" schema_desc:"错误涉及的源码位置"`
	FieldIssues   []workspaceBuildFieldIssue `json:"field_issues,omitempty" schema_desc:"字段级 schema/tag 问题"`
	SDKSymbols    []string                   `json:"sdk_symbols,omitempty" schema_desc:"未确认或未定义的 SDK 符号"`
	Hints         []string                   `json:"hints,omitempty" schema_desc:"直接修复提示"`
	RequiredDocs  []string                   `json:"required_docs,omitempty" schema_desc:"修复前建议优先阅读的文档"`
	RepairPolicy  []string                   `json:"repair_policy,omitempty" schema_desc:"修复策略"`
	RetryPolicy   string                     `json:"retry_policy,omitempty" schema_desc:"重试策略"`
}

type workspaceBuildFieldIssue struct {
	Field    string `json:"field,omitempty" schema_desc:"Go 字段名"`
	JSONName string `json:"json_name,omitempty" schema_desc:"JSON 字段名"`
	Message  string `json:"message,omitempty" schema_desc:"字段问题摘要"`
}

const workspaceBuildFailureKind = "agent_app_build_failure"

var undefinedSDKSelectorRe = regexp.MustCompile(`undefined:\s*((?:app|types|chart|response|callback|statistics)\.[A-Z][A-Za-z0-9_]*)`)
var workspaceBuildRouterRe = regexp.MustCompile(`router\s+(/[A-Za-z0-9_./-]+)`)
var workspaceBuildFileRefRe = regexp.MustCompile(`(?:^|\s)((?:namespace/)?[A-Za-z0-9_./-]+\.go:\d+(?::\d+)?)`)
var workspaceBuildFieldIssueRe = regexp.MustCompile(`field\s+([A-Za-z0-9_]+)\s+\(([^)]+)\):`)

var buildWorkspaceToolDef = toolDefinitionWithOutput[buildWorkspaceArgs, structuredToolResultSchema[buildWorkspaceResultData]](
	"build_workspace",
	"编译当前工作空间（Go 应用）。不写文件，仅基于当前已落盘的代码触发一次编译并部署。无需传参。连续写多个文件后可调用一次 build_workspace 再编译。构建成功后返回 agent_app_build 阶段产物和 pending_test 交互状态，前端应提示用户确认是否交接给 qa_engineer 测试工程师；构建失败时不要交接测试，也不要凭直觉反复重写。先完整阅读错误，按 router/字段/文件定位同类问题；不清楚 SDK schema、widget、callback、审计字段或 API 写法时，先 read_doc /system/prompt/sdk/reference/build-validation、SDK 主文档或匹配案例，再批量修复后重新 build。",
)

func (t *BuildWorkspaceTool) Definition() dto.ToolDef {
	return buildWorkspaceToolDef
}

func (t *BuildWorkspaceTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	result, content, isError := runBuildWorkspaceTool(ctx, call.FullCodePath)
	if isError {
		return toolResultWithData(content, true, result)
	}
	return toolResultWithStructuredData(result, false, content)
}

// runBuildWorkspaceTool 编译当前工作空间（不写文件，仅触发编译并部署）；从当前工作目录解析 user/app，无需参数
func runBuildWorkspaceTool(ctx context.Context, currentFullCodePath string) (buildWorkspaceResultData, string, bool) {
	dir := strings.Trim(strings.TrimSpace(currentFullCodePath), "/")
	if dir == "" {
		content := "build_workspace 无法获取当前工作目录，请确保在有效的工作台会话中操作"
		return buildWorkspaceFailureResult("", content), content, true
	}
	parts := strings.Split(dir, "/")
	if len(parts) < 2 {
		content := "build_workspace 当前目录格式应为 /user/app 或更长路径（如 /luobei/demo）"
		return buildWorkspaceFailureResult("", content), content, true
	}
	user, app := parts[0], parts[1]
	workspacePath := fmt.Sprintf("/%s/%s", user, app)
	resp, err := apicall.UpdateAppBuild(ctx, user, app)
	if err != nil {
		logger.Errorf(ctx, "[WorkspaceBuild] UpdateAppBuild 失败: %v", err)
		return buildWorkspaceFailureResult(workspacePath, err.Error()), "build_workspace 调用失败: " + enrichWorkspaceBuildError(err.Error(), workspacePath), true
	}
	result := buildWorkspaceSuccessResult(workspacePath, resp)
	content := fmt.Sprintf("工作空间已编译并部署: workspace=%s, app=%s, 旧版本=%s, 新版本=%s。请确认是否进入测试；看不到按钮也可以直接回复：开始测试 / 测试 / 暂不测试。", workspacePath, resp.App, resp.OldVersion, resp.NewVersion)
	return result, content, false
}

func buildWorkspaceSuccessResult(workspacePath string, resp *dto.UpdateAppResp) buildWorkspaceResultData {
	user, app := splitWorkspacePath(workspacePath)
	result := buildWorkspaceResultData{
		Kind:          workspaceBuildArtifactKind,
		Status:        "ok",
		WorkspacePath: strings.TrimSpace(workspacePath),
		User:          user,
		App:           app,
		Interaction:   pendingBuildTestInteraction(),
	}
	if resp == nil {
		return result
	}
	if strings.TrimSpace(resp.User) != "" {
		result.User = resp.User
	}
	if strings.TrimSpace(resp.App) != "" {
		result.App = resp.App
	}
	result.OldVersion = resp.OldVersion
	result.NewVersion = resp.NewVersion
	result.GitCommitHash = resp.GitCommitHash
	result.Warnings = append([]string(nil), resp.Warnings...)
	return result
}

func buildWorkspaceFailureResult(workspacePath string, errText string) buildWorkspaceResultData {
	workspacePath = normalizeWorkspacePath(workspacePath)
	user, app := splitWorkspacePath(workspacePath)
	return buildWorkspaceResultData{
		Kind:             workspaceBuildFailureKind,
		Status:           "error",
		WorkspacePath:    workspacePath,
		User:             user,
		App:              app,
		Error:            compactText(errText, 700),
		BuildDiagnostics: buildWorkspaceDiagnostics(errText, workspacePath),
	}
}

func buildWorkspaceDiagnostics(errText string, workspacePath string) *workspaceBuildDiagnostics {
	errText = strings.TrimSpace(errText)
	workspacePath = normalizeWorkspacePath(workspacePath)
	diagnostics := &workspaceBuildDiagnostics{
		Status:        "error",
		WorkspacePath: workspacePath,
		ErrorSummary:  compactText(errText, 420),
		RetryPolicy:   "同类错误第二次出现前，必须补读 required_docs、匹配案例或相关 SDK 源码；不要继续同一方案重试或整文件重写。",
	}
	if errText == "" {
		diagnostics.Status = "empty"
		diagnostics.ErrorSummary = "未提供构建错误正文；进入构建修复前需要先读取 build_workspace 完整失败输出。"
	}
	diagnostics.Routers = extractWorkspaceBuildRouters(errText)
	diagnostics.Files = extractWorkspaceBuildFiles(errText)
	diagnostics.FieldIssues = extractWorkspaceBuildFieldIssues(errText)
	diagnostics.SDKSymbols = extractWorkspaceBuildSDKSymbols(errText)
	diagnostics.Categories = workspaceBuildErrorCategories(errText)
	diagnostics.Hints = trimRoleHandoffStrings(workspaceBuildErrorHints(errText), 8)
	diagnostics.RequiredDocs = workspaceBuildRequiredDocsForCategories(diagnostics.Categories)
	diagnostics.RepairPolicy = workspaceBuildRepairPolicyForCategories(diagnostics.Categories)
	return diagnostics
}

func extractWorkspaceBuildRouters(errText string) []string {
	out := []string{}
	for _, match := range workspaceBuildRouterRe.FindAllStringSubmatch(errText, -1) {
		if len(match) > 1 {
			out = appendUniqueRoleHandoffStrings(out, match[1])
		}
	}
	return trimRoleHandoffStrings(out, 12)
}

func extractWorkspaceBuildFiles(errText string) []string {
	out := []string{}
	for _, match := range workspaceBuildFileRefRe.FindAllStringSubmatch(errText, -1) {
		if len(match) > 1 {
			out = appendUniqueRoleHandoffStrings(out, match[1])
		}
	}
	return trimRoleHandoffStrings(out, 12)
}

func extractWorkspaceBuildSDKSymbols(errText string) []string {
	out := []string{}
	for _, match := range undefinedSDKSelectorRe.FindAllStringSubmatch(errText, -1) {
		if len(match) > 1 {
			out = appendUniqueRoleHandoffStrings(out, match[1])
		}
	}
	return trimRoleHandoffStrings(out, 12)
}

func extractWorkspaceBuildFieldIssues(errText string) []workspaceBuildFieldIssue {
	matches := workspaceBuildFieldIssueRe.FindAllStringSubmatchIndex(errText, -1)
	if len(matches) == 0 {
		return nil
	}
	out := []workspaceBuildFieldIssue{}
	seen := map[string]bool{}
	for i, match := range matches {
		if len(match) < 6 {
			continue
		}
		start := match[1]
		end := len(errText)
		if i+1 < len(matches) {
			end = matches[i+1][0]
		}
		if newline := strings.IndexByte(errText[start:end], '\n'); newline >= 0 {
			end = start + newline
		}
		field := strings.TrimSpace(errText[match[2]:match[3]])
		jsonName := strings.TrimSpace(errText[match[4]:match[5]])
		message := compactText(errText[start:end], 180)
		key := field + "/" + jsonName + "/" + message
		if field == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, workspaceBuildFieldIssue{
			Field:    field,
			JSONName: jsonName,
			Message:  message,
		})
		if len(out) >= 12 {
			break
		}
	}
	return out
}

func workspaceBuildErrorCategories(errText string) []string {
	categories := []string{}
	add := func(category string) {
		categories = appendUniqueRoleHandoffStrings(categories, category)
	}
	if strings.TrimSpace(errText) == "" {
		add("missing_error_context")
		return categories
	}
	if strings.Contains(errText, "SDK schema compile failed") || strings.Contains(errText, "schema decode failed") || strings.Contains(errText, "failed to validate request model") {
		add("schema_validation")
	}
	if strings.Contains(errText, "audit field") {
		add("audit_field")
	}
	if strings.Contains(errText, "requires options or OnSelectFuzzyMap entry") || strings.Contains(errText, "OnSelectFuzzyMap") {
		add("select_options")
	}
	if strings.Contains(errText, "unsupported widget") || strings.Contains(errText, "options_colors") || strings.Contains(errText, "invalid color") || strings.Contains(errText, "number widget requires") {
		add("widget")
	}
	if strings.Contains(errText, "table request field") || strings.Contains(errText, "Req has no field or method") {
		add("table_request")
	}
	if strings.Contains(errText, "GetPage undefined") || strings.Contains(errText, "GetPageSize undefined") || strings.Contains(errText, "unknown field Total") || strings.Contains(errText, "unknown field DataList") {
		add("pagination_response")
	}
	if len(extractWorkspaceBuildSDKSymbols(errText)) > 0 {
		add("sdk_api_or_go_symbol")
	}
	if strings.Contains(errText, "does not implement chart.Charter") || strings.Contains(errText, "resp.Charts undefined") {
		add("chart_response")
	}
	if strings.Contains(errText, "redeclared in this block") {
		add("duplicate_definition")
	}
	if len(categories) == 0 {
		add("build_failure")
	}
	return categories
}

func workspaceBuildRequiredDocsForCategories(categories []string) []string {
	docs := []string{
		"/system/prompt/sdk/reference/build-validation",
		"/system/prompt/sdk/agent-app-sdk-readme",
	}
	for _, category := range categories {
		switch category {
		case "schema_validation", "audit_field", "select_options", "widget", "table_request", "pagination_response", "chart_response":
			docs = appendUniqueRoleHandoffStrings(docs, "/system/prompt/case_catalog")
		}
	}
	return trimRoleHandoffStrings(docs, 6)
}

func workspaceBuildRepairPolicyForCategories(categories []string) []string {
	policy := []string{
		"先按 router/字段/文件分组同类错误，小范围批量修复；不要直接整文件重写。",
		"不确定 SDK schema、widget、callback、审计字段或 API 写法时，先读取 required_docs 和匹配案例。",
		"修复后只在 execute_directory 对应工作空间重新 build_workspace；不要扩大到整个空间测试。",
	}
	for _, category := range categories {
		switch category {
		case "audit_field":
			policy = appendUniqueRoleHandoffStrings(policy, "审计字段必须按标准 tag 修复：created_by/updated_by 的 hide 和 gorm column 要与 SDK 规范一致，不要删除系统字段绕过校验。")
		case "select_options":
			policy = appendUniqueRoleHandoffStrings(policy, "select/multiselect 必须有静态 options 或 OnSelectFuzzyMap；纯展示名称改为 input 或补真实回调。")
		case "widget":
			policy = appendUniqueRoleHandoffStrings(policy, "widget 的 type、tag 和 options_colors 先对照白名单；颜色用不带 # 的 RRGGBB。")
		case "table_request":
			policy = appendUniqueRoleHandoffStrings(policy, "Table Request 与 Model 冲突或 req 字段不存在时，同步检查 Request 字段、PageSortReq 和 Handler 筛选逻辑。")
		case "pagination_response":
			policy = appendUniqueRoleHandoffStrings(policy, "分页返回按 SDK response.TableResult 构造，不要猜 GetPage/GetPageSize/Total/DataList 等不存在 API。")
		case "sdk_api_or_go_symbol":
			policy = appendUniqueRoleHandoffStrings(policy, "undefined SDK 符号必须先查文档、案例或源码确认真实导出符号，再改调用方式。")
		case "chart_response":
			policy = appendUniqueRoleHandoffStrings(policy, "Chart 路由只返回 SDK chart 对象；多个图拆多个 .chart 路由，指标放 Metadata。")
		case "duplicate_definition":
			policy = appendUniqueRoleHandoffStrings(policy, "重复定义先定位同名模型/函数/方法，保留一个共享定义，其他文件直接复用。")
		}
	}
	return trimRoleHandoffStrings(policy, 10)
}

func splitWorkspacePath(workspacePath string) (string, string) {
	parts := strings.Split(strings.Trim(strings.TrimSpace(workspacePath), "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func enrichWorkspaceBuildError(errText string, workspacePath ...string) string {
	prefix := workspaceBuildScopeMessage(workspacePath...)
	hints := workspaceBuildErrorHints(errText)
	guidance := workspaceBuildFailureWorkflowHint()
	if len(hints) == 0 {
		return prefix + errText + guidance
	}
	return prefix + errText + guidance + "\n\n常见修复建议:\n- " + strings.Join(hints, "\n- ")
}

func workspaceBuildFailureWorkflowHint() string {
	return "\n\n下一步要求:\n- 先完整阅读错误，按 router/字段/文件定位同类问题并批量修复；不确定 SDK schema、widget、callback、审计字段或 API 写法时，先 read_doc(\"/system/prompt/sdk/reference/build-validation\")、SDK 主文档或匹配案例，不要想当然改 tag 或反复整文件重写。"
}

func workspaceBuildScopeMessage(workspacePath ...string) string {
	if len(workspacePath) == 0 {
		return ""
	}
	scope := strings.TrimSpace(workspacePath[0])
	if scope == "" {
		return ""
	}
	return fmt.Sprintf("本次编译范围: %s。启动期校验只覆盖该工作空间内已注册路由；错误中的 router /xxx 是该工作空间内相对路由。\n", scope)
}

func workspaceBuildErrorHints(errText string) []string {
	var hints []string
	add := func(hint string) {
		for _, existing := range hints {
			if existing == hint {
				return
			}
		}
		hints = append(hints, hint)
	}

	if strings.Contains(errText, "options_colors") || strings.Contains(errText, "invalid color") || strings.Contains(errText, "contains invalid color") {
		add("options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，不能写 primary/success/default/rgb(...)，数量也必须和 options 一致。")
	}
	if strings.Contains(errText, "unsupported widget type") || strings.Contains(errText, "unsupported widget tag") || strings.Contains(errText, "invalid tag format") {
		add("widget 的 type 和配置 key 必须来自 SDK 主文档组件速查和运行时白名单；不要写 file/readonly/multiple 等未支持类型或参数。文件上传用 type:files + string，多文件用 max_count；图片/视频列表预览用 thumbnail:true;list_preview:true；只读展示用 hide:\"create,update\" 或 widget:\"-\" 控制场景。")
	}
	if strings.Contains(errText, "number widget requires integer Go type") {
		add("数值组件的字段类型要匹配：整数用 type:number，小数、金额、评分、均值、比例用 type:float。")
	}
	if strings.Contains(errText, "as *int64 value in argument") && strings.Contains(errText, ".Count") {
		add("GORM Count 的参数必须是 *int64；声明 var total int64，再 Count(&total)。需要业务 int 时在计算处显式 int(total)。")
	}
	if strings.Contains(errText, "DateTimeBucketExpr returns 2 values") || (strings.Contains(errText, "too many arguments in call") && strings.Contains(errText, ".Group")) {
		add("app.DateTimeBucketExpr 返回 selectExpr 和 groupExpr 两个值：dateExpr, groupExpr := app.DateTimeBucketExpr(...); Select 用 dateExpr，Group 只传 groupExpr 一个参数。")
	}
	if strings.Contains(errText, "table request field") && strings.Contains(errText, "conflicts with table model field") {
		add("Table 列表使用 query.PageSortReq：Request 显式声明筛选字段，并在 Handler 里手写 Where。")
	}
	if strings.Contains(errText, "undefined") && strings.Contains(errText, "Req has no field or method") {
		add("调整 Table Request 字段后，要同步更新 Handler 里对 req.<字段> 的手写筛选。新 Table 默认把筛选字段写在 Request，分页嵌入 query.PageSortReq。")
	}
	if strings.Contains(errText, "OnSelectFuzzyMap field") || strings.Contains(errText, "must use select or multiselect widget") {
		add("OnSelectFuzzyMap 的 key 必须对应 schema 中 type:select 或 type:multiselect 的字段；Table 新增/编辑要回调选择时，把外键字段定义在 Model 上并注册回调。")
	}
	if strings.Contains(errText, "requires options or OnSelectFuzzyMap entry") {
		add("select/multiselect 字段必须有选项来源：静态 options，或字段 callback:\"OnSelectFuzzy\" 并在对应 Template.OnSelectFuzzyMap 注册；纯展示名称不要写成 select，改用 input 或补真实回调。")
	}
	if strings.Contains(errText, "GetPage undefined") || strings.Contains(errText, "GetPageSize undefined") ||
		strings.Contains(errText, "unknown field Total") || strings.Contains(errText, "unknown field DataList") {
		add("Table 列表先用 req.PageSortReq.GetOrder/GetOffset/GetLimit 显式排序分页并查询 total，再用 resp.Table(response.TableResult{Items: rows, TotalCount: total, PageInfo: &req.PageSortReq}).Build() 返回；不要调用 req.GetPageSize()，也不要手工构造 Total/DataList。")
	}
	if strings.Contains(errText, "Time.Format undefined") || strings.Contains(errText, "has no field or method Format") ||
		strings.Contains(errText, "type func() time.Time") || strings.Contains(errText, ".Time.Format") {
		add("types.Time 做格式化或比较时先调用 Time() 方法，例如 t.Time().Format(...)、t.Time().After(...)，不要写 t.Format(...) 或 t.Time.Format(...)。")
	}
	for _, match := range undefinedSDKSelectorRe.FindAllStringSubmatch(errText, -1) {
		add(fmt.Sprintf("代码使用了未确认的 SDK API %s；只允许使用已读文档、案例或源码中真实导出的符号，先读对应知识点或 SDK 源码，不要按命名猜 API。", match[1]))
	}
	if strings.Contains(errText, "does not implement chart.Charter") || strings.Contains(errText, "as chart.Charter value in argument to resp.Chart") {
		add("Chart Handler 必须把 SDK chart 包里的具体图表对象传给 resp.Chart；不要把自定义业务响应结构体传给 resp.Chart，附加指标放到图表 Metadata，多个图拆多个 .chart 路由。")
	}
	if strings.Contains(errText, "resp.Charts undefined") {
		add("SDK 没有 resp.Charts；一个 Chart 路由只返回一张图，用 resp.Chart(chart).Build()。多张图拆成多个 .chart 路由，汇总指标放 Metadata。")
	}
	if strings.Contains(errText, "redeclared in this block") {
		add("同一个目录里的模型、函数和方法只能定义一次；共享模型应放在一个文件中，其他文件直接复用。")
	}
	if strings.Contains(errText, "audit field") {
		add("系统审计字段必须按 SDK 规范写完整 tag：created_by/updated_by 用 widget type:user、hide:\"create,update\"，且 gorm column 必须与 json 名一致；CreatedBy 示例为 `gorm:\"column:created_by\" widget:\"name:创建人;type:user\" hide:\"create,update\"`。")
	}
	return hints
}
