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
	CurrentRole  string `json:"current_role" schema_desc:"当前身份 ID；没有则留空"`
	TargetRole   string `json:"target_role" schema_desc:"目标身份 ID，例如 product_manager/app_developer/app_operator/qa_engineer。为空时仅沿用 current_role；没有 current_role 则进入 reviewer"`
	UserInput    string `json:"user_input" schema_desc:"用户本轮最新需求；仅作上下文记录，change_role 不按关键词推断身份"`
	TaskSummary  string `json:"task_summary" schema_desc:"切换前要携带的极简任务摘要"`
	Directory    string `json:"directory" schema_desc:"当前工作目录"`
	ResetContext bool   `json:"reset_context" schema_desc:"是否建议丢弃旧细节，只保留摘要进入新身份"`
}

type changeRoleData struct {
	PreviousRole     string              `json:"previous_role,omitempty" schema_desc:"切换前身份"`
	PreviousRoleName string              `json:"previous_role_name,omitempty" schema_desc:"切换前展示名称"`
	RoleID           string              `json:"role_id" schema_desc:"当前角色 ID" schema_required:"true"`
	DisplayName      string              `json:"display_name" schema_desc:"当前角色展示名称" schema_required:"true"`
	CurrentRole      string              `json:"current_role" schema_desc:"当前身份" schema_required:"true"`
	Switched         bool                `json:"switched" schema_desc:"是否发生身份切换" schema_required:"true"`
	Reason           string              `json:"reason" schema_desc:"选择或切换原因" schema_required:"true"`
	Directory        string              `json:"directory,omitempty" schema_desc:"工作目录"`
	ContextPolicy    string              `json:"context_policy" schema_desc:"上下文携带策略" schema_required:"true"`
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

var changeRoleToolDef = toolDefinitionWithOutput[changeRoleArgs, structuredToolResultSchema[changeRoleData]](
	"change_role",
	"根据用户最新需求选择或切换工作台身份，并返回该身份需要的文档包正文。每轮开始执行前必须先用它明确当前身份；只读、无副作用。",
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
	data := buildChangeRole(ctx, args)
	return toolResultWithStructuredData(data, false)
}

func buildChangeRole(ctx context.Context, args changeRoleArgs) changeRoleData {
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
	contextPolicy := buildRoleContextPolicy(switched, args.ResetContext, args.TaskSummary)

	return changeRoleData{
		PreviousRole:     previous,
		PreviousRoleName: workspaceRoleDisplayName(previous),
		RoleID:           target,
		DisplayName:      roleSpec.DisplayName,
		CurrentRole:      target,
		Switched:         switched,
		Reason:           reason,
		Directory:        strings.TrimSpace(args.Directory),
		ContextPolicy:    contextPolicy,
		RequiredDocs:     requiredDocs,
		LoadedDocs:       loadedDocs,
		MissingDocs:      missingDocs,
		AllowedNextTools: workspaceRoleAllowedTools(target),
		NextAction:       roleSpec.Action,
		NextRoles:        roleSpec.NextRoles,
	}
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

func buildRoleContextPolicy(switched bool, reset bool, summary string) string {
	summary = strings.TrimSpace(summary)
	if reset {
		if summary == "" {
			return "已进入新身份；丢弃旧细节，只保留用户最新目标、当前目录和必要文件路径。"
		}
		return "已进入新身份；丢弃旧细节，只保留本摘要、用户最新目标、当前目录和必要文件路径。摘要：" + compactText(summary, 220)
	}
	if switched {
		if summary == "" {
			return "已切换身份；旧上下文只作背景，决策以当前身份文档包、用户最新目标和当前源码为准。"
		}
		return "已切换身份；旧上下文只作背景，优先携带摘要并按当前身份文档包执行。摘要：" + compactText(summary, 220)
	}
	return "沿用当前身份；继续以当前身份文档包、用户最新目标和当前源码为准。"
}
