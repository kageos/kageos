package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/kageos/kageos/dto"
)

type SummarizeTaskStateTool struct{}

type summarizeTaskStateArgs struct {
	RoleID              string   `json:"role_id" schema_desc:"当前角色 ID，例如 product_manager/app_developer/app_operator/maintenance_engineer/qa_engineer" schema_required:"true"`
	Directory           string   `json:"directory" schema_desc:"当前工作目录"`
	Outcome             string   `json:"outcome" schema_desc:"当前阶段结果，保留结论和关键状态" schema_required:"true"`
	UserGoal            string   `json:"user_goal" schema_desc:"用户最终目标或业务意图"`
	ConfirmedScope      []string `json:"confirmed_scope" schema_desc:"已确认范围、已完成能力或产物"`
	KeyDecisions        []string `json:"key_decisions" schema_desc:"已定下来的关键决策、取舍和默认处理"`
	Constraints         []string `json:"constraints" schema_desc:"必须遵守的限制、权限、字段、平台或实现约束"`
	NonGoals            []string `json:"non_goals" schema_desc:"明确不做、暂不做或不能做的内容"`
	OpenQuestions       []string `json:"open_questions" schema_desc:"仍未解决但需要后续角色知道的问题"`
	ChangedFiles        []string `json:"changed_files" schema_desc:"本阶段改动文件路径"`
	Verified            []string `json:"verified" schema_desc:"已验证事项或命令"`
	Blockers            []string `json:"blockers" schema_desc:"未解决问题"`
	ImplementationNotes []string `json:"implementation_notes" schema_desc:"给开发/维护/构建角色的实现注意事项"`
	VerificationNotes   []string `json:"verification_notes" schema_desc:"给测试/操作角色的验证或运行注意事项"`
	ArtifactRefs        []string `json:"artifact_refs" schema_desc:"相关产物、函数路径、版本、文件或消息引用"`
	ReferenceDocs       []string `json:"reference_docs" schema_desc:"下一角色应优先读取或已作为依据的参考文档、案例、SDK、平台边界、PRD、外部文档路径或 URL"`
	ReferenceFiles      []string `json:"reference_files" schema_desc:"下一角色应优先查看的源码、日志、配置、生成文件或工作区路径"`
	NextRoleID          string   `json:"next_role_id" schema_desc:"建议下一角色 ID，例如 qa_engineer/build_engineer"`
	NextGoal            string   `json:"next_goal" schema_desc:"下一阶段目标"`
}

type taskStateSummaryData struct {
	RoleID              string          `json:"role_id" schema_desc:"当前角色 ID" schema_required:"true"`
	Directory           string          `json:"directory,omitempty" schema_desc:"工作目录"`
	Summary             string          `json:"summary" schema_desc:"高密度阶段摘要；优先使用 handoff 四块传给 change_role" schema_required:"true"`
	Handoff             roleHandoffData `json:"handoff" schema_desc:"可直接传给 change_role 的四块标准交接信息" schema_required:"true"`
	UserGoal            string          `json:"user_goal,omitempty" schema_desc:"用户目标"`
	ConfirmedScope      []string        `json:"confirmed_scope,omitempty" schema_desc:"已确认范围"`
	KeyDecisions        []string        `json:"key_decisions,omitempty" schema_desc:"关键决策"`
	Constraints         []string        `json:"constraints,omitempty" schema_desc:"约束"`
	NonGoals            []string        `json:"non_goals,omitempty" schema_desc:"非目标"`
	OpenQuestions       []string        `json:"open_questions,omitempty" schema_desc:"待确认问题"`
	ChangedFiles        []string        `json:"changed_files,omitempty" schema_desc:"改动文件"`
	Verified            []string        `json:"verified,omitempty" schema_desc:"已验证事项"`
	Blockers            []string        `json:"blockers,omitempty" schema_desc:"未解决问题"`
	ImplementationNotes []string        `json:"implementation_notes,omitempty" schema_desc:"实现注意事项"`
	VerificationNotes   []string        `json:"verification_notes,omitempty" schema_desc:"验证注意事项"`
	ArtifactRefs        []string        `json:"artifact_refs,omitempty" schema_desc:"相关产物引用"`
	ReferenceDocs       []string        `json:"reference_docs,omitempty" schema_desc:"参考文档"`
	ReferenceFiles      []string        `json:"reference_files,omitempty" schema_desc:"参考文件"`
	NextRoleID          string          `json:"next_role_id,omitempty" schema_desc:"建议下一角色"`
	NextGoal            string          `json:"next_goal,omitempty" schema_desc:"下一阶段目标"`
}

var summarizeTaskStateToolDef = toolDefinitionWithOutput[summarizeTaskStateArgs, structuredToolResultSchema[taskStateSummaryData]](
	"summarize_task_state",
	"把当前任务阶段压缩成可传给 change_role 的四块标准交接信息。适用于所有角色切换：开发、构建、测试、维护、业务操作、平台集成、数据处理和 review。只读、无副作用；用于避免旧 PRD、旧错误和长上下文污染下一身份，同时保留执行目录、任务上下文、关键信息、参考资料和下一步目标。",
)

func (t *SummarizeTaskStateTool) Definition() dto.ToolDef {
	return summarizeTaskStateToolDef
}

func (t *SummarizeTaskStateTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	args, err := decodeToolArgs[summarizeTaskStateArgs](call.Args)
	if err != nil {
		return toolResult("summarize_task_state 参数解析失败: "+err.Error(), true)
	}
	data := buildTaskStateSummary(args)
	return toolResultWithStructuredData(data, false)
}

func buildTaskStateSummary(args summarizeTaskStateArgs) taskStateSummaryData {
	roleID := normalizeTaskStateRole(args.RoleID, WorkspaceRoleReviewer)
	nextRoleID := normalizeTaskStateRole(args.NextRoleID, "")
	confirmedScope := trimStringSlice(args.ConfirmedScope)
	keyDecisions := trimStringSlice(args.KeyDecisions)
	constraints := trimStringSlice(args.Constraints)
	nonGoals := trimStringSlice(args.NonGoals)
	openQuestions := trimStringSlice(args.OpenQuestions)
	changedFiles := trimStringSlice(args.ChangedFiles)
	verified := trimStringSlice(args.Verified)
	blockers := trimStringSlice(args.Blockers)
	implementationNotes := trimStringSlice(args.ImplementationNotes)
	verificationNotes := trimStringSlice(args.VerificationNotes)
	artifactRefs := trimStringSlice(args.ArtifactRefs)
	referenceDocs := trimStringSlice(args.ReferenceDocs)
	referenceFiles := trimStringSlice(args.ReferenceFiles)
	parts := []string{
		"角色=" + roleID,
		"结果=" + compactText(args.Outcome, 260),
	}
	if strings.TrimSpace(args.UserGoal) != "" {
		parts = append(parts, "用户目标="+compactText(args.UserGoal, 220))
	}
	if len(confirmedScope) > 0 {
		parts = append(parts, "已确认="+compactText(strings.Join(confirmedScope, "；"), 260))
	}
	if len(keyDecisions) > 0 {
		parts = append(parts, "关键决策="+compactText(strings.Join(keyDecisions, "；"), 260))
	}
	if len(constraints) > 0 {
		parts = append(parts, "约束="+compactText(strings.Join(constraints, "；"), 260))
	}
	if len(nonGoals) > 0 {
		parts = append(parts, "不做="+compactText(strings.Join(nonGoals, "；"), 220))
	}
	if len(changedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("改动=%d 个文件:%s", len(changedFiles), compactText(strings.Join(changedFiles, "，"), 220)))
	}
	if len(verified) > 0 {
		parts = append(parts, "验证="+compactText(strings.Join(verified, "；"), 220))
	}
	if len(blockers) > 0 {
		parts = append(parts, "阻塞="+compactText(strings.Join(blockers, "；"), 220))
	}
	if len(implementationNotes) > 0 {
		parts = append(parts, "实现注意="+compactText(strings.Join(implementationNotes, "；"), 260))
	}
	if len(verificationNotes) > 0 {
		parts = append(parts, "验证注意="+compactText(strings.Join(verificationNotes, "；"), 220))
	}
	if len(openQuestions) > 0 {
		parts = append(parts, "未决="+compactText(strings.Join(openQuestions, "；"), 220))
	}
	if len(artifactRefs) > 0 {
		parts = append(parts, "引用="+compactText(strings.Join(artifactRefs, "；"), 220))
	}
	if len(referenceDocs) > 0 {
		parts = append(parts, "参考文档="+compactText(strings.Join(referenceDocs, "；"), 260))
	}
	if len(referenceFiles) > 0 {
		parts = append(parts, "参考文件="+compactText(strings.Join(referenceFiles, "；"), 260))
	}
	if nextRoleID != "" {
		parts = append(parts, "下一角色="+nextRoleID)
	}
	if strings.TrimSpace(args.NextGoal) != "" {
		parts = append(parts, "下一目标="+compactText(args.NextGoal, 260))
	}
	handoff := buildTaskStateHandoff(args, confirmedScope, keyDecisions, constraints, nonGoals, openQuestions, changedFiles, verified, blockers, implementationNotes, verificationNotes, artifactRefs, referenceDocs, referenceFiles)
	return taskStateSummaryData{
		RoleID:              roleID,
		Directory:           handoff.ExecuteDirectory,
		Summary:             strings.Join(parts, " | "),
		Handoff:             handoff,
		UserGoal:            strings.TrimSpace(args.UserGoal),
		ConfirmedScope:      confirmedScope,
		KeyDecisions:        keyDecisions,
		Constraints:         constraints,
		NonGoals:            nonGoals,
		OpenQuestions:       openQuestions,
		ChangedFiles:        changedFiles,
		Verified:            verified,
		Blockers:            blockers,
		ImplementationNotes: implementationNotes,
		VerificationNotes:   verificationNotes,
		ArtifactRefs:        artifactRefs,
		ReferenceDocs:       referenceDocs,
		ReferenceFiles:      referenceFiles,
		NextRoleID:          nextRoleID,
		NextGoal:            strings.TrimSpace(args.NextGoal),
	}
}

func buildTaskStateHandoff(args summarizeTaskStateArgs, confirmedScope, keyDecisions, constraints, nonGoals, openQuestions, changedFiles, verified, blockers, implementationNotes, verificationNotes, artifactRefs, referenceDocs, referenceFiles []string) roleHandoffData {
	executeDirectory := deriveTaskStateExecuteDirectory(args.Directory, changedFiles, artifactRefs, referenceFiles)
	taskContext := []string{}
	if outcome := compactText(args.Outcome, 260); outcome != "" {
		taskContext = append(taskContext, "阶段结果："+outcome)
	}
	if goal := compactText(args.UserGoal, 220); goal != "" {
		taskContext = append(taskContext, "用户目标："+goal)
	}
	if len(confirmedScope) > 0 {
		taskContext = append(taskContext, "已确认范围："+compactText(strings.Join(confirmedScope, "；"), 260))
	}
	if len(constraints) > 0 {
		taskContext = append(taskContext, "必须满足："+compactText(strings.Join(constraints, "；"), 260))
	}
	if len(nonGoals) > 0 {
		taskContext = append(taskContext, "不做："+compactText(strings.Join(nonGoals, "；"), 220))
	}
	if len(openQuestions) > 0 {
		taskContext = append(taskContext, "未决/特殊 case："+compactText(strings.Join(openQuestions, "；"), 220))
	}
	if len(blockers) > 0 {
		taskContext = append(taskContext, "阻塞："+compactText(strings.Join(blockers, "；"), 220))
	}

	keyInfo := []string{}
	keyInfo = append(keyInfo, keyDecisions...)
	if originalDirectory := normalizeRoleHandoffDirectory(args.Directory); originalDirectory != "" && executeDirectory != "" && originalDirectory != executeDirectory {
		keyInfo = append(keyInfo, "工作区构建/来源目录："+originalDirectory)
		keyInfo = append(keyInfo, "下一阶段目标应用目录："+executeDirectory)
	}
	if len(changedFiles) > 0 {
		keyInfo = append(keyInfo, "改动文件："+compactText(strings.Join(changedFiles, "，"), 260))
	}
	if len(verified) > 0 {
		keyInfo = append(keyInfo, "已验证："+compactText(strings.Join(verified, "；"), 260))
	}
	keyInfo = append(keyInfo, implementationNotes...)
	keyInfo = append(keyInfo, verificationNotes...)
	keyInfo = append(keyInfo, artifactRefs...)

	references := append([]string{}, referenceDocs...)
	references = appendUniqueRoleHandoffStrings(references, referenceFiles...)
	return roleHandoffData{
		ExecuteDirectory: executeDirectory,
		TaskContext:      trimRoleHandoffStrings(taskContext, 8),
		KeyInformation:   trimRoleHandoffStrings(keyInfo, 12),
		References:       trimRoleHandoffStrings(references, 16),
	}
}

func deriveTaskStateExecuteDirectory(directory string, changedFiles, artifactRefs, referenceFiles []string) string {
	directory = normalizeRoleHandoffDirectory(directory)
	candidates := append([]string{}, changedFiles...)
	candidates = append(candidates, artifactRefs...)
	candidates = append(candidates, referenceFiles...)
	if narrowed := workspaceTargetDirectoryFromCandidates(directory, candidates); narrowed != "" {
		return narrowed
	}
	return directory
}

func normalizeTaskStateRole(role string, fallback string) string {
	role = strings.TrimSpace(role)
	if role == "" {
		return fallback
	}
	normalized := normalizeWorkspaceRole(role)
	if isKnownWorkspaceRole(normalized) {
		return normalized
	}
	return role
}

func trimStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func compactText(s string, maxRunes int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if maxRunes <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
