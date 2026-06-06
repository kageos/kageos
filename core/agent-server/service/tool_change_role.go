package service

import (
	"context"
	"slices"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/dto"
)

type ChangeRoleTool struct{}

type changeRoleArgs struct {
	CurrentRole      string   `json:"current_role" schema_desc:"当前身份 ID；没有则留空"`
	TargetRole       string   `json:"target_role" schema_desc:"目标身份 ID，例如 product_manager/app_developer/app_operator/qa_engineer；沿用身份时也明确传当前身份" schema_required:"true"`
	ExecuteDirectory string   `json:"execute_directory" schema_desc:"下一身份所有读取、构建、测试、运行都必须限定的工作台目录完整路径；新建应用开发阶段传已存在父目录，例如 /user/app，目标新目录放入 key_information；测试/维护/操作阶段传目标应用目录；不能写“当前目录”" schema_required:"true"`
	TaskContext      []string `json:"task_context" schema_desc:"交接上下文：上一阶段做了什么、用户原始目标/需求、必须满足的要求、特殊 case 或未决问题；3-6 条短句" schema_required:"true"`
	KeyInformation   []string `json:"key_information" schema_desc:"下一身份必须知道的关键信息：PRD 摘要、构建版本、函数/表单/表格/图表路径、测试重点、失败现象等"`
	References       []string `json:"references" schema_desc:"参考资料：PRD/构建产物、示例案例、系统文档、SDK 文档、源码文件、日志或外部 URL；只放真正要看的资料"`
	ResetContext     bool     `json:"reset_context" schema_desc:"是否建议丢弃旧细节，只保留本次四块交接信息进入新身份"`

	UserInput      string   `json:"user_input" schema_ignore:"true"`
	TaskSummary    string   `json:"task_summary" schema_ignore:"true"`
	ReferenceDocs  []string `json:"reference_docs" schema_ignore:"true"`
	ReferenceFiles []string `json:"reference_files" schema_ignore:"true"`
	Directory      string   `json:"directory" schema_ignore:"true"`
}

type changeRoleData struct {
	PreviousRole     string              `json:"previous_role,omitempty" schema_desc:"切换前身份"`
	PreviousRoleName string              `json:"previous_role_name,omitempty" schema_desc:"切换前展示名称"`
	RoleID           string              `json:"role_id" schema_desc:"当前角色 ID" schema_required:"true"`
	DisplayName      string              `json:"display_name" schema_desc:"当前角色展示名称" schema_required:"true"`
	CurrentRole      string              `json:"current_role" schema_desc:"当前身份" schema_required:"true"`
	Switched         bool                `json:"switched" schema_desc:"是否发生身份切换" schema_required:"true"`
	Reason           string              `json:"reason" schema_desc:"选择或切换原因" schema_required:"true"`
	ExecuteDirectory string              `json:"execute_directory" schema_desc:"下一身份执行目录" schema_required:"true"`
	Directory        string              `json:"directory,omitempty" schema_desc:"工作目录"`
	Handoff          roleHandoffData     `json:"handoff" schema_desc:"标准四块角色交接信息" schema_required:"true"`
	ContextPolicy    string              `json:"context_policy" schema_desc:"上下文携带策略" schema_required:"true"`
	ReferenceDocs    []string            `json:"reference_docs,omitempty" schema_desc:"建议优先读取的参考文档"`
	ReferenceFiles   []string            `json:"reference_files,omitempty" schema_desc:"建议优先查看的参考文件"`
	RequiredDocs     []string            `json:"required_docs" schema_desc:"当前身份文档路径" schema_required:"true"`
	LoadedDocs       []changeRoleDoc     `json:"loaded_docs" schema_desc:"已返回的文档正文" schema_required:"true"`
	MissingDocs      []string            `json:"missing_docs,omitempty" schema_desc:"未能读取到正文的文档路径"`
	AllowedNextTools []string            `json:"allowed_next_tools,omitempty" schema_desc:"当前身份常用下一步工具"`
	NextAction       string              `json:"next_action" schema_desc:"当前身份下一步动作" schema_required:"true"`
	NextRoles        []nextWorkspaceRole `json:"next_roles,omitempty" schema_desc:"完成后的推荐后续角色"`
}

type changeRoleDoc struct {
	Path    string `json:"path" schema_desc:"文档路径" schema_required:"true"`
	Name    string `json:"name" schema_desc:"文档名称" schema_required:"true"`
	Content string `json:"content" schema_desc:"文档正文" schema_required:"true"`
}

type roleHandoffData struct {
	ExecuteDirectory string   `json:"execute_directory" schema_desc:"执行目录" schema_required:"true"`
	TaskContext      []string `json:"task_context" schema_desc:"上一阶段和用户需求摘要" schema_required:"true"`
	KeyInformation   []string `json:"key_information,omitempty" schema_desc:"关键信息"`
	References       []string `json:"references,omitempty" schema_desc:"参考资料"`
}

var changeRoleToolDef = toolDefinitionWithOutput[changeRoleArgs, structuredToolResultSchema[changeRoleData]](
	"change_role",
	"根据用户最新需求选择或切换工作台身份，并返回该身份需要的文档包正文。每轮开始执行前必须先用它明确当前身份。必须使用四块标准交接：execute_directory、task_context、key_information、references；下一身份只能在 execute_directory 指定目录内读取、构建、测试或运行。",
)

func (t *ChangeRoleTool) Definition() dto.ToolDef {
	return changeRoleToolDef
}

func (t *ChangeRoleTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[changeRoleArgs](call.Args)
	if err != nil {
		return toolResult("change_role 参数解析失败: "+err.Error(), true)
	}
	if strings.TrimSpace(args.TargetRole) != "" && !isKnownWorkspaceRole(args.TargetRole) {
		return toolResult("target_role 不支持: "+strings.TrimSpace(args.TargetRole)+"。请使用标准角色 ID：product_manager、app_developer、maintenance_engineer、app_operator、qa_engineer、build_engineer、data_operator、platform_engineer、reviewer。", true)
	}
	data := buildChangeRole(ctx, args, call.FullCodePath)
	return toolResultWithStructuredData(data, false)
}

func buildChangeRole(ctx context.Context, args changeRoleArgs, fallbackDirectory ...string) changeRoleData {
	previous := normalizeWorkspaceRole(args.CurrentRole)
	if !isKnownWorkspaceRole(previous) {
		previous = ""
	}
	target := normalizeWorkspaceRole(args.TargetRole)
	reason := ""
	switch {
	case target != "" && isKnownWorkspaceRole(target):
		reason = "按 target_role 选择身份"
	case target != "":
		target = WorkspaceRoleReviewer
		reason = "target_role 不是已知身份，进入只读分析身份"
	case previous != "":
		target = previous
		reason = "未指定 target_role，沿用 current_role"
	default:
		target = WorkspaceRoleReviewer
		reason = "未指定 target_role 且没有 current_role，进入只读分析身份"
	}
	if previous != "" && target != "" && previous != target {
		if when, ok := workspaceRoleTransitionWhen(previous, target); ok && when != "" {
			reason += "；符合推荐流转：" + when
		}
	}

	roleSpec, _ := workspaceRoleSpecFor(target)
	requiredDocs := roleDocumentPackage(target, roleSpec)
	loadedDocs, missingDocs := loadRoleDocs(ctx, requiredDocs)
	switched := previous != "" && previous != target
	referenceDocs := trimStringSlice(args.ReferenceDocs)
	referenceFiles := trimStringSlice(args.ReferenceFiles)
	handoff := buildRoleHandoff(args, firstNonEmptyString(fallbackDirectory...))
	handoff = normalizeRoleHandoffForTargetRole(target, handoff, firstNonEmptyString(fallbackDirectory...))
	handoff = appendRoleHandoffAdvice(target, handoff)
	contextSummary := buildRoleHandoffSummary(handoff, args.TaskSummary)
	contextPolicy := buildRoleContextPolicy(switched, args.ResetContext, contextSummary, referenceDocs, referenceFiles, handoff)

	return changeRoleData{
		PreviousRole:     previous,
		PreviousRoleName: workspaceRoleDisplayName(previous),
		RoleID:           target,
		DisplayName:      roleSpec.DisplayName,
		CurrentRole:      target,
		Switched:         switched,
		Reason:           reason,
		ExecuteDirectory: handoff.ExecuteDirectory,
		Directory:        handoff.ExecuteDirectory,
		Handoff:          handoff,
		ContextPolicy:    contextPolicy,
		ReferenceDocs:    referenceDocs,
		ReferenceFiles:   referenceFiles,
		RequiredDocs:     requiredDocs,
		LoadedDocs:       loadedDocs,
		MissingDocs:      missingDocs,
		AllowedNextTools: workspaceRoleAllowedTools(target),
		NextAction:       roleSpec.Action,
		NextRoles:        roleSpec.NextRoles,
	}
}

func buildRoleHandoff(args changeRoleArgs, fallbackDirectory string) roleHandoffData {
	executeDirectory := firstNonEmptyString(args.ExecuteDirectory, args.Directory, fallbackDirectory)
	executeDirectory = normalizeRoleHandoffDirectory(executeDirectory)
	taskContext := trimStringSlice(args.TaskContext)
	if len(taskContext) == 0 {
		for _, item := range []string{args.TaskSummary, args.UserInput} {
			if s := compactText(item, 280); s != "" {
				taskContext = append(taskContext, s)
			}
		}
	}
	if len(taskContext) == 0 {
		taskContext = append(taskContext, "未提供上一阶段摘要；只按本轮用户需求、当前目录和角色文档推进。")
	}
	keyInfo := trimStringSlice(args.KeyInformation)
	references := trimStringSlice(args.References)
	references = appendUniqueRoleHandoffStrings(references, args.ReferenceDocs...)
	references = appendUniqueRoleHandoffStrings(references, args.ReferenceFiles...)
	return roleHandoffData{
		ExecuteDirectory: executeDirectory,
		TaskContext:      trimRoleHandoffStrings(taskContext, 8),
		KeyInformation:   trimRoleHandoffStrings(keyInfo, 12),
		References:       trimRoleHandoffStrings(references, 16),
	}
}

func appendRoleHandoffAdvice(targetRole string, handoff roleHandoffData) roleHandoffData {
	advice := roleHandoffAdvice(targetRole, handoff)
	if len(advice) == 0 {
		return handoff
	}
	handoff.KeyInformation = appendUniqueRoleHandoffStrings(advice, handoff.KeyInformation...)
	handoff.KeyInformation = trimRoleHandoffStrings(handoff.KeyInformation, 16)
	return handoff
}

func roleHandoffAdvice(targetRole string, handoff roleHandoffData) []string {
	targetRole = normalizeWorkspaceRole(targetRole)
	out := []string{}
	switch targetRole {
	case WorkspaceRoleProductManager:
		out = append(out, "下一步建议：先确认当前目录现有函数是否已能满足目标；能满足就交接给 app_operator，不能满足且用户要长期系统时再写 PRD。")
	case WorkspaceRoleAppOperator:
		out = append(out, "下一步建议：先用 search_tools/search_resources 限定 execute_directory 确认函数 schema、必填项、枚举和写入能力；不要编造关联 ID，必要时先查询或调用 run_on_select_fuzzy。")
	case WorkspaceRoleAppDeveloper:
		out = append(out, "下一步建议：先读 SDK 主文档和匹配案例，再按 PRD 写代码；生成或构建失败时不要反复整文件重写，先补读错误相关文档/案例/源码。")
	case WorkspaceRoleMaintenanceEngineer:
		out = append(out, "下一步建议：先读目标目录和相关源码，确认 schema/回调/业务逻辑后小范围修改；同类失败超过一次时先补读 SDK、案例或日志，不要盲目重写。")
	case WorkspaceRoleBuildEngineer:
		out = append(out, "下一步建议：先读完整构建/schema 错误和相关 SDK/案例/源码，按错误类型批量修复；不要猜不存在的 API 或反复用同一方案重试。")
	case WorkspaceRoleQAEngineer:
		out = append(out, "下一步建议：先用 search_tools(directory=execute_directory) 确认函数 schema；按主数据/配置表 -> Form 提交 -> 目标记录表 -> Chart/结果查询顺序测试，并把失败分类为参数、数据、schema、业务 bug 或构建问题。")
	}
	if handoffSuggestsFailedRetry(handoff) {
		out = append(out, "失败处理建议：当前上下文已有失败/阻塞信号，不要继续同一方案重试；先读取 references 中的 SDK、案例、源码或日志，再决定修复路径。")
	}
	return out
}

func handoffSuggestsFailedRetry(handoff roleHandoffData) bool {
	parts := make([]string, 0, len(handoff.TaskContext)+len(handoff.KeyInformation)+len(handoff.References))
	parts = append(parts, handoff.TaskContext...)
	parts = append(parts, handoff.KeyInformation...)
	parts = append(parts, handoff.References...)
	text := strings.ToLower(strings.Join(parts, "；"))
	for _, keyword := range []string{"失败", "报错", "错误", "阻塞", "重试", "反复", "多次", "仍然", "不通过", "failed", "error"} {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func normalizeRoleHandoffForTargetRole(targetRole string, handoff roleHandoffData, fallbackDirectory string) roleHandoffData {
	targetRole = normalizeWorkspaceRole(targetRole)
	if targetRole != WorkspaceRoleAppDeveloper {
		return handoff
	}
	fallback := normalizeWorkspacePath(fallbackDirectory)
	if fallback == "" {
		return handoff
	}
	workspaceRoot := workspaceRootPath(fallback)
	executeDirectory := normalizeWorkspacePath(handoff.ExecuteDirectory)
	if workspaceRoot == "" || executeDirectory == "" || executeDirectory == workspaceRoot {
		return handoff
	}
	if !workspacePathHasPrefix(executeDirectory, workspaceRoot) {
		return handoff
	}
	if !shouldNormalizeAppDeveloperNewAppHandoff(handoff) {
		return handoff
	}
	targetDirectory := executeDirectory
	handoff.ExecuteDirectory = workspaceRoot
	handoff.References = pruneAppDeveloperHandoffReferences(handoff.References, workspaceRoot, targetDirectory)
	handoff.KeyInformation = appendUniqueRoleHandoffStrings([]string{
		"新建应用目标目录：" + targetDirectory,
		"开发阶段先在已存在父目录 " + workspaceRoot + " 下创建目标目录，再在目标目录写代码。",
	}, handoff.KeyInformation...)
	return handoff
}

func shouldNormalizeAppDeveloperNewAppHandoff(handoff roleHandoffData) bool {
	parts := make([]string, 0, len(handoff.TaskContext)+len(handoff.KeyInformation)+len(handoff.References))
	parts = append(parts, handoff.TaskContext...)
	parts = append(parts, handoff.KeyInformation...)
	parts = append(parts, handoff.References...)
	text := strings.Join(parts, "；")
	for _, keyword := range []string{
		"PRD",
		"prd",
		"agent_app_prd",
		"开发阶段",
		"新建应用",
		"目标应用目录",
		"创建目录",
		"生成代码",
		"写 Go",
		"写代码",
		"build",
		"Build",
		"构建",
	} {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func pruneAppDeveloperHandoffReferences(references []string, workspaceRoot, targetDirectory string) []string {
	workspaceRoot = normalizeWorkspacePath(workspaceRoot)
	targetDirectory = normalizeWorkspacePath(targetDirectory)
	out := make([]string, 0, len(references))
	for _, ref := range references {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		pathRef := normalizeWorkspacePath(ref)
		if workspaceRoot != "" && targetDirectory != "" && workspacePathHasPrefix(pathRef, workspaceRoot) &&
			pathRef != workspaceRoot && !workspacePathHasPrefix(pathRef, targetDirectory) {
			continue
		}
		out = appendUniqueRoleHandoffStrings(out, ref)
	}
	return out
}

func normalizeRoleHandoffDirectory(directory string) string {
	directory = strings.TrimSpace(directory)
	if directory == "" {
		return ""
	}
	if !strings.HasPrefix(directory, "/") {
		directory = "/" + directory
	}
	return directory
}

func buildRoleHandoffSummary(handoff roleHandoffData, legacySummary string) string {
	parts := []string{}
	if handoff.ExecuteDirectory != "" {
		parts = append(parts, "执行目录="+handoff.ExecuteDirectory)
	}
	if len(handoff.TaskContext) > 0 {
		parts = append(parts, "任务上下文="+compactText(strings.Join(handoff.TaskContext, "；"), 500))
	}
	if len(handoff.KeyInformation) > 0 {
		parts = append(parts, "关键信息="+compactText(strings.Join(handoff.KeyInformation, "；"), 500))
	}
	if len(handoff.References) > 0 {
		parts = append(parts, "参考资料="+compactText(strings.Join(handoff.References, "；"), 500))
	}
	if len(parts) == 0 {
		return strings.TrimSpace(legacySummary)
	}
	if strings.TrimSpace(legacySummary) != "" && !strings.Contains(strings.Join(parts, "；"), strings.TrimSpace(legacySummary)) {
		parts = append(parts, "补充摘要="+compactText(legacySummary, 500))
	}
	return strings.Join(parts, " | ")
}

func appendUniqueRoleHandoffStrings(items []string, more ...string) []string {
	for _, item := range more {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !containsWorkspaceRoleString(items, item) {
			items = append(items, item)
		}
	}
	return items
}

func trimRoleHandoffStrings(items []string, limit int) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = compactText(item, 300)
		if item == "" || containsWorkspaceRoleString(out, item) {
			continue
		}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func roleDocumentPackage(role string, spec workspaceRoleSpec) []string {
	role = normalizeWorkspaceRole(role)
	docs := make([]string, 0, len(spec.Docs)+len(spec.Optional)+4)
	addDoc := func(path string) {
		path = prompt.NormalizePromptDocPath(path)
		if path == "" || slices.Contains(docs, path) {
			return
		}
		docs = append(docs, path)
	}
	for _, doc := range spec.Docs {
		addDoc(doc)
	}
	for _, doc := range spec.Optional {
		addDoc(doc)
	}
	switch role {
	case WorkspaceRoleMaintenanceEngineer:
		addDoc("/system/prompt/sdk/agent-app-sdk-readme")
	case WorkspaceRoleBuildEngineer:
		for _, doc := range []string{
			"/system/prompt/sdk/agent-app-sdk-readme",
			"/system/prompt/sdk/reference/build-validation",
		} {
			addDoc(doc)
		}
	case WorkspaceRolePlatformEngineer:
		for _, doc := range []string{
			"/system/prompt/platform-capability-boundaries",
		} {
			addDoc(doc)
		}
	}
	return docs
}

func loadRoleDocs(ctx context.Context, paths []string) ([]changeRoleDoc, []string) {
	loaded := make([]changeRoleDoc, 0, len(paths))
	var missing []string
	for _, docPath := range paths {
		name, content := prompt.GetPromptDocContent(ctx, docPath)
		if strings.TrimSpace(content) == "" {
			missing = append(missing, docPath)
			continue
		}
		if strings.TrimSpace(name) == "" {
			name = docPath
		}
		loaded = append(loaded, changeRoleDoc{
			Path:    docPath,
			Name:    name,
			Content: content,
		})
	}
	return loaded, missing
}

func buildRoleContextPolicy(switched bool, reset bool, summary string, referenceDocs []string, referenceFiles []string, handoff roleHandoffData) string {
	summary = strings.TrimSpace(summary)
	referenceText := buildRoleReferenceText(referenceDocs, referenceFiles)
	directoryText := ""
	if handoff.ExecuteDirectory != "" {
		directoryText = "执行目录固定为 " + handoff.ExecuteDirectory + "；所有读取、构建、测试、运行都必须限定在该目录或该目录下函数。"
	}
	if reset {
		if summary == "" {
			return strings.TrimSpace("已进入新身份；丢弃旧细节，只保留标准四块交接信息、用户最新目标和必要文件路径。" + directoryText + referenceText)
		}
		return "已进入新身份；丢弃旧细节，只保留标准四块交接信息、用户最新目标和必要文件路径。" + directoryText + "摘要：" + compactText(summary, 1200) + referenceText
	}
	if switched {
		if summary == "" {
			return strings.TrimSpace("已切换身份；旧上下文只作背景，决策以标准四块交接、当前身份文档包、用户最新目标和当前源码为准。" + directoryText + referenceText)
		}
		return "已切换身份；旧上下文只作背景，优先携带标准四块交接并按当前身份文档包执行。" + directoryText + "摘要：" + compactText(summary, 1200) + referenceText
	}
	return strings.TrimSpace("沿用当前身份；继续以当前身份文档包、用户最新目标和当前源码为准。" + directoryText + referenceText)
}

func buildRoleReferenceText(referenceDocs []string, referenceFiles []string) string {
	parts := []string{}
	if len(referenceDocs) > 0 {
		parts = append(parts, "参考文档="+compactText(strings.Join(referenceDocs, "；"), 500))
	}
	if len(referenceFiles) > 0 {
		parts = append(parts, "参考文件="+compactText(strings.Join(referenceFiles, "；"), 500))
	}
	if len(parts) == 0 {
		return ""
	}
	return " " + strings.Join(parts, "；") + "。"
}
