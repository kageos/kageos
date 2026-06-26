package service

import (
	"encoding/json"
	"strings"
)

const workspaceRoleHandoffPacketVersion = "role_handoff.v1"

type workspaceRoleHandoffPacket struct {
	Version            string                        `json:"version" schema_desc:"阶段交接版本" schema_required:"true"`
	SourceSessionID    string                        `json:"source_session_id,omitempty" schema_desc:"来源会话 ID"`
	SourceRole         string                        `json:"source_role,omitempty" schema_desc:"来源角色 ID"`
	TargetRole         string                        `json:"target_role" schema_desc:"目标角色 ID" schema_required:"true"`
	ArtifactKind       string                        `json:"artifact_kind,omitempty" schema_desc:"阶段产物类型"`
	ExecuteDirectory   string                        `json:"execute_directory" schema_desc:"目标角色主执行目录/绑定目录" schema_required:"true"`
	WorkspaceDirectory string                        `json:"workspace_directory,omitempty" schema_desc:"工作空间根目录"`
	TargetAppDirectory string                        `json:"target_app_directory,omitempty" schema_desc:"目标应用目录"`
	TaskContext        []string                      `json:"task_context" schema_desc:"上一阶段和用户需求摘要" schema_required:"true"`
	KeyInformation     []string                      `json:"key_information,omitempty" schema_desc:"下一角色必须知道的关键事实"`
	References         []string                      `json:"references,omitempty" schema_desc:"下一角色要优先读取的资料"`
	ContextPolicy      string                        `json:"context_policy,omitempty" schema_desc:"上下文携带策略"`
	Artifact           *workspaceRolePacketArtifact  `json:"artifact,omitempty" schema_desc:"完整产物引用，不内联大 JSON"`
	ArtifactDigest     *workspaceArtifactDigest      `json:"artifact_digest,omitempty" schema_desc:"产物摘要"`
	BuildDiagnostics   *workspaceBuildDiagnostics    `json:"build_diagnostics,omitempty" schema_desc:"构建失败诊断，仅构建修复阶段出现"`
	ExecutedHooks      []workspaceExecutedRoleHook   `json:"executed_hooks,omitempty" schema_desc:"已执行的角色生命周期 Hook"`
	Validation         workspaceRolePacketValidation `json:"validation" schema_desc:"阶段交接校验结果" schema_required:"true"`
}

type workspaceRolePacketArtifact struct {
	Kind     string `json:"kind,omitempty" schema_desc:"产物类型"`
	Included bool   `json:"included" schema_desc:"本消息是否包含完整产物 JSON" schema_required:"true"`
	Source   string `json:"source,omitempty" schema_desc:"完整产物所在块或来源"`
}

type workspaceRolePacketValidation struct {
	Status   string   `json:"status" schema_desc:"校验状态：ok/warning/error" schema_required:"true"`
	Errors   []string `json:"errors,omitempty" schema_desc:"阻断性字段错误"`
	Warnings []string `json:"warnings,omitempty" schema_desc:"非阻断风险"`
	Repaired []string `json:"repaired,omitempty" schema_desc:"已自动修正或裁剪的字段"`
}

func formatWorkspaceRoleHandoffPacketJSON(packet *workspaceRoleHandoffPacket) string {
	if packet == nil {
		return "{}"
	}
	raw, err := json.MarshalIndent(packet, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func buildWorkspaceRoleHandoffPacketFromChangeRole(previousRole, targetRole string, handoff roleHandoffData, contextPolicy string, hookOutput workspaceRoleHookOutput) workspaceRoleHandoffPacket {
	packet := workspaceRoleHandoffPacket{
		Version:          workspaceRoleHandoffPacketVersion,
		SourceRole:       normalizeWorkspaceRole(previousRole),
		TargetRole:       normalizeWorkspaceRole(targetRole),
		ExecuteDirectory: normalizeWorkspacePath(handoff.ExecuteDirectory),
		TaskContext:      trimRoleHandoffStrings(handoff.TaskContext, 8),
		KeyInformation:   trimRoleHandoffStrings(handoff.KeyInformation, 16),
		References:       trimRoleHandoffStrings(handoff.References, 20),
		ContextPolicy:    strings.TrimSpace(contextPolicy),
		ExecutedHooks:    hookOutput.ExecutedHooks,
	}
	if packet.ExecuteDirectory == "" {
		packet.ExecuteDirectory = strings.TrimSpace(handoff.ExecuteDirectory)
	}
	if len(packet.TaskContext) == 0 {
		packet.TaskContext = []string{"未提供上一阶段摘要；只按本轮用户需求、当前目录和角色文档推进。"}
	}
	if normalizeWorkspaceRole(targetRole) == WorkspaceRoleBuildEngineer && hookOutput.BuildDiagnostics != nil {
		packet.BuildDiagnostics = hookOutput.BuildDiagnostics
		packet.References = appendUniqueRoleHandoffStrings(packet.References, hookOutput.BuildDiagnostics.RequiredDocs...)
		packet.References = trimRoleHandoffStrings(packet.References, 20)
	}
	normalizeAndValidateWorkspaceRoleHandoffPacket(&packet)
	return packet
}

func buildWorkspaceRoleHandoffPacketFromContext(ctx workspaceHandoffContext) *workspaceRoleHandoffPacket {
	packet := &workspaceRoleHandoffPacket{
		Version:            workspaceRoleHandoffPacketVersion,
		SourceSessionID:    strings.TrimSpace(ctx.SourceSessionID),
		SourceRole:         normalizeWorkspaceRole(ctx.SourceRole),
		TargetRole:         normalizeWorkspaceRole(ctx.TargetRole),
		ArtifactKind:       strings.TrimSpace(ctx.ArtifactKind),
		ExecuteDirectory:   normalizeWorkspacePath(ctx.ExecuteDirectory),
		WorkspaceDirectory: normalizeWorkspacePath(ctx.WorkspaceDirectory),
		TargetAppDirectory: normalizeWorkspacePath(ctx.TargetAppDirectory),
		TaskContext:        workspaceRolePacketTaskContextFromContext(ctx),
		KeyInformation:     workspaceRolePacketKeyInformationFromContext(ctx),
		References:         workspaceRolePacketReferencesFromContext(ctx),
		ContextPolicy:      strings.TrimSpace(ctx.ContextPolicy),
		ArtifactDigest:     ctx.ArtifactDigest,
		ExecutedHooks:      ctx.ExecutedHooks,
	}
	if packet.ExecuteDirectory == "" {
		packet.ExecuteDirectory = strings.TrimSpace(ctx.ExecuteDirectory)
	}
	if ctx.ArtifactIncluded || strings.TrimSpace(ctx.ArtifactKind) != "" {
		packet.Artifact = &workspaceRolePacketArtifact{
			Kind:     strings.TrimSpace(ctx.ArtifactKind),
			Included: ctx.ArtifactIncluded,
			Source:   strings.ToUpper(firstNonEmptyString(ctx.ArtifactKind, "artifact")) + " JSON",
		}
	}
	if shouldIncludeBuildDiagnosticsInHandoffPacket(ctx) {
		packet.BuildDiagnostics = ctx.BuildDiagnostics
	}
	if len(packet.TaskContext) == 0 {
		packet.TaskContext = []string{firstNonEmptyString(ctx.StageSummary, "阶段交接已确认；按目标角色、执行目录和参考资料继续。")}
	}
	normalizeAndValidateWorkspaceRoleHandoffPacket(packet)
	return packet
}

func workspaceRolePacketTaskContextFromContext(ctx workspaceHandoffContext) []string {
	out := []string{}
	for _, item := range []string{
		ctx.StageSummary,
		prefixedPacketText("用户目标", ctx.UserGoal),
		prefixedPacketText("补充备注", ctx.Remark),
	} {
		if item != "" {
			out = appendUniqueRoleHandoffStrings(out, item)
		}
	}
	for _, note := range ctx.LatestUserNotes {
		if note = compactText(note, 220); note != "" {
			out = appendUniqueRoleHandoffStrings(out, "来源用户补充："+note)
		}
	}
	if len(ctx.OpenQuestions) > 0 {
		out = appendUniqueRoleHandoffStrings(out, "未决/需确认："+compactText(strings.Join(ctx.OpenQuestions, "；"), 260))
	}
	return trimRoleHandoffStrings(out, 8)
}

func workspaceRolePacketKeyInformationFromContext(ctx workspaceHandoffContext) []string {
	out := []string{}
	if ctx.ExecuteDirectory != "" {
		out = appendUniqueRoleHandoffStrings(out, "主执行目录/绑定目录："+ctx.ExecuteDirectory)
	}
	if ctx.WorkspaceDirectory != "" {
		out = appendUniqueRoleHandoffStrings(out, "工作空间根目录："+ctx.WorkspaceDirectory)
	}
	if ctx.TargetAppDirectory != "" {
		out = appendUniqueRoleHandoffStrings(out, "目标应用目录："+ctx.TargetAppDirectory)
	}
	if placementDecision := workspaceHandoffDirectoryPlacementDecision(ctx); placementDecision != "" {
		out = appendUniqueRoleHandoffStrings(out, placementDecision)
	}
	if digestSummary := workspaceRolePacketDigestSummary(ctx.ArtifactDigest); digestSummary != "" {
		out = appendUniqueRoleHandoffStrings(out, digestSummary)
	}
	out = appendPacketSection(out, "已确认范围", ctx.ConfirmedScope, 4)
	out = appendPacketSection(out, "关键决策", ctx.KeyDecisions, 5)
	out = appendPacketSection(out, "约束", ctx.Constraints, 5)
	out = appendPacketSection(out, "不做", ctx.NonGoals, 3)
	switch normalizeWorkspaceRole(ctx.TargetRole) {
	case WorkspaceRoleAppDeveloper, WorkspaceRoleMaintenanceEngineer, WorkspaceRoleBuildEngineer:
		out = appendPacketSection(out, "实现重点", ctx.ImplementationFocus, 6)
	case WorkspaceRoleQAEngineer:
		out = appendPacketSection(out, "测试重点", ctx.VerificationFocus, 8)
	case WorkspaceRoleAppOperator:
		out = appendPacketSection(out, "操作重点", ctx.WorkflowNotes, 6)
	}
	if shouldIncludeBuildDiagnosticsInHandoffPacket(ctx) {
		out = appendPacketSection(out, "构建错误类别", ctx.BuildDiagnostics.Categories, 4)
		out = appendPacketSection(out, "构建错误 router", ctx.BuildDiagnostics.Routers, 4)
		out = appendPacketSection(out, "构建修复策略", ctx.BuildDiagnostics.RepairPolicy, 4)
	}
	return trimRoleHandoffStrings(out, 16)
}

func workspaceRolePacketReferencesFromContext(ctx workspaceHandoffContext) []string {
	out := []string{}
	if ctx.ArtifactIncluded && ctx.ArtifactKind != "" {
		out = appendUniqueRoleHandoffStrings(out, strings.ToUpper(ctx.ArtifactKind)+" JSON（本消息完整产物块）")
	}
	out = appendUniqueRoleHandoffStrings(out, ctx.ReferenceDocs...)
	out = appendUniqueRoleHandoffStrings(out, ctx.ReferenceFiles...)
	if shouldIncludeBuildDiagnosticsInHandoffPacket(ctx) && ctx.BuildDiagnostics != nil {
		out = appendUniqueRoleHandoffStrings(out, ctx.BuildDiagnostics.RequiredDocs...)
	}
	return trimRoleHandoffStrings(out, 24)
}

func workspaceRolePacketDigestSummary(digest *workspaceArtifactDigest) string {
	if digest == nil {
		return ""
	}
	parts := []string{}
	if name := firstNonEmptyString(digest.ProjectName, digest.ProjectCode); name != "" {
		parts = append(parts, "项目="+name)
	}
	if len(digest.Tables) > 0 {
		parts = append(parts, "Table="+strings.Join(workspaceRolePacketResourceNames(digest.Tables, 4), "、"))
	}
	if len(digest.Forms) > 0 {
		parts = append(parts, "Form="+strings.Join(workspaceRolePacketResourceNames(digest.Forms, 4), "、"))
	}
	if len(digest.Charts) > 0 {
		parts = append(parts, "Chart="+strings.Join(workspaceRolePacketResourceNames(digest.Charts, 4), "、"))
	}
	if len(parts) == 0 {
		return ""
	}
	return compactText("产物摘要："+strings.Join(parts, "；"), 280)
}

func workspaceRolePacketResourceNames(items []workspaceResourceDigest, limit int) []string {
	out := []string{}
	for _, item := range items {
		out = appendUniqueRoleHandoffStrings(out, firstNonEmptyString(item.Name, item.Code))
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func appendPacketSection(out []string, label string, items []string, limit int) []string {
	items = trimRoleHandoffStrings(items, limit)
	if len(items) == 0 {
		return out
	}
	return appendUniqueRoleHandoffStrings(out, label+"："+compactText(strings.Join(items, "；"), 280))
}

func prefixedPacketText(label string, value string) string {
	value = compactText(value, 240)
	if value == "" {
		return ""
	}
	return label + "：" + value
}

func shouldIncludeBuildDiagnosticsInHandoffPacket(ctx workspaceHandoffContext) bool {
	return normalizeWorkspaceRole(ctx.TargetRole) == WorkspaceRoleBuildEngineer && ctx.BuildDiagnostics != nil
}

func normalizeAndValidateWorkspaceRoleHandoffPacket(packet *workspaceRoleHandoffPacket) {
	if packet == nil {
		return
	}
	validation := workspaceRolePacketValidation{Status: "ok"}
	if strings.TrimSpace(packet.Version) != workspaceRoleHandoffPacketVersion {
		packet.Version = workspaceRoleHandoffPacketVersion
		validation.Repaired = append(validation.Repaired, "version 已归一为 "+workspaceRoleHandoffPacketVersion)
	}
	packet.SourceSessionID = strings.TrimSpace(packet.SourceSessionID)
	packet.ArtifactKind = strings.TrimSpace(packet.ArtifactKind)
	packet.ContextPolicy = strings.TrimSpace(packet.ContextPolicy)

	sourceRole := normalizeWorkspaceRole(packet.SourceRole)
	if strings.TrimSpace(packet.SourceRole) != "" && sourceRole == "" {
		validation.Warnings = append(validation.Warnings, "source_role 不是标准角色 ID，已忽略")
		validation.Repaired = append(validation.Repaired, "source_role 已清空")
	}
	packet.SourceRole = sourceRole

	targetRole := normalizeWorkspaceRole(packet.TargetRole)
	if targetRole == "" || !isKnownWorkspaceRole(targetRole) {
		validation.Errors = append(validation.Errors, "target_role 必须是标准工作台角色 ID")
	} else if targetRole != packet.TargetRole {
		validation.Repaired = append(validation.Repaired, "target_role 已归一为 "+targetRole)
	}
	packet.TargetRole = targetRole

	packet.WorkspaceDirectory = normalizeWorkspacePath(packet.WorkspaceDirectory)
	packet.TargetAppDirectory = normalizeWorkspacePath(packet.TargetAppDirectory)
	rawExecuteDirectory := strings.TrimSpace(packet.ExecuteDirectory)
	if workspaceRolePacketIsPlaceholderDirectory(rawExecuteDirectory) {
		packet.ExecuteDirectory = ""
		validation.Errors = append(validation.Errors, "execute_directory 不能使用“当前目录”等占位描述")
		validation.Repaired = append(validation.Repaired, "execute_directory 占位描述已清空")
	} else {
		normalizedExecuteDirectory := normalizeWorkspacePath(rawExecuteDirectory)
		if normalizedExecuteDirectory != "" && normalizedExecuteDirectory != rawExecuteDirectory {
			validation.Repaired = append(validation.Repaired, "execute_directory 已归一为 "+normalizedExecuteDirectory)
		}
		packet.ExecuteDirectory = firstNonEmptyString(normalizedExecuteDirectory, rawExecuteDirectory)
	}
	if packet.ExecuteDirectory == "" {
		validation.Errors = append(validation.Errors, "execute_directory 必填，且必须是具体工作台目录")
	}
	if packet.TargetRole == WorkspaceRoleQAEngineer && packet.TargetAppDirectory != "" && packet.ExecuteDirectory != packet.TargetAppDirectory {
		packet.ExecuteDirectory = packet.TargetAppDirectory
		validation.Repaired = append(validation.Repaired, "qa_engineer.execute_directory 已收窄到 target_app_directory")
	}

	packet.TaskContext = sanitizeWorkspaceRolePacketStrings(packet.TaskContext, 8, "task_context", &validation)
	if len(packet.TaskContext) == 0 {
		packet.TaskContext = []string{"未提供上一阶段摘要；只按本轮用户需求、当前目录和角色文档推进。"}
		validation.Warnings = append(validation.Warnings, "task_context 缺少上一阶段或用户目标摘要")
		validation.Repaired = append(validation.Repaired, "task_context 已补默认摘要")
	} else if !workspaceRolePacketHasTaskSignal(packet.TaskContext) {
		validation.Warnings = append(validation.Warnings, "task_context 未明显包含用户目标或上一阶段摘要")
	}
	packet.KeyInformation = sanitizeWorkspaceRolePacketStrings(packet.KeyInformation, 16, "key_information", &validation)
	packet.References = sanitizeWorkspaceRolePacketReferences(packet.References, &validation)

	if packet.TargetRole != WorkspaceRoleBuildEngineer && packet.BuildDiagnostics != nil {
		packet.BuildDiagnostics = nil
		validation.Warnings = append(validation.Warnings, "build_diagnostics 只能交给 build_engineer")
		validation.Repaired = append(validation.Repaired, "已移除非 build_engineer 的 build_diagnostics")
	}
	if packet.TargetRole == WorkspaceRoleBuildEngineer && packet.ArtifactKind == workspaceBuildFailureKind && packet.BuildDiagnostics == nil {
		validation.Errors = append(validation.Errors, "agent_app_build_failure 交接给 build_engineer 时必须携带 build_diagnostics")
	}
	if packet.Artifact != nil {
		packet.Artifact.Kind = strings.TrimSpace(packet.Artifact.Kind)
		packet.Artifact.Source = compactText(packet.Artifact.Source, 120)
		if workspaceRolePacketLooksLikeFullArtifact(packet.Artifact.Source) {
			packet.Artifact.Source = strings.ToUpper(firstNonEmptyString(packet.Artifact.Kind, packet.ArtifactKind, "artifact")) + " JSON"
			validation.Repaired = append(validation.Repaired, "artifact.source 已从疑似完整 JSON 裁剪为产物块引用")
		}
	}
	packet.ExecutedHooks = trimWorkspaceExecutedHooks(packet.ExecutedHooks, 12)
	validation.Errors = trimRoleHandoffStrings(validation.Errors, 8)
	validation.Warnings = trimRoleHandoffStrings(validation.Warnings, 8)
	validation.Repaired = trimRoleHandoffStrings(validation.Repaired, 8)
	switch {
	case len(validation.Errors) > 0:
		validation.Status = "error"
	case len(validation.Warnings) > 0 || len(validation.Repaired) > 0:
		validation.Status = "warning"
	default:
		validation.Status = "ok"
	}
	packet.Validation = validation
}

func sanitizeWorkspaceRolePacketStrings(items []string, limit int, field string, validation *workspaceRolePacketValidation) []string {
	out := []string{}
	for _, item := range items {
		item = compactText(item, 300)
		if item == "" {
			continue
		}
		if workspaceRolePacketLooksLikeFullArtifact(item) {
			if validation != nil {
				validation.Warnings = append(validation.Warnings, field+" 包含疑似完整产物 JSON")
				validation.Repaired = append(validation.Repaired, field+" 中的疑似完整产物 JSON 已裁剪")
			}
			item = "完整产物 JSON 已裁剪；请读取 artifact 引用和 artifact_digest。"
		}
		out = appendUniqueRoleHandoffStrings(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func sanitizeWorkspaceRolePacketReferences(items []string, validation *workspaceRolePacketValidation) []string {
	out := []string{}
	for _, item := range items {
		item = compactText(item, 300)
		if item == "" {
			continue
		}
		if workspaceRolePacketIsPlaceholderDirectory(item) || workspaceRolePacketIsMeaninglessReference(item) {
			if validation != nil {
				validation.Warnings = append(validation.Warnings, "references 包含无效或泛化引用")
				validation.Repaired = append(validation.Repaired, "已移除无效 references 项："+compactText(item, 80))
			}
			continue
		}
		if workspaceRolePacketLooksLikeFullArtifact(item) {
			if validation != nil {
				validation.Warnings = append(validation.Warnings, "references 包含疑似完整产物 JSON")
				validation.Repaired = append(validation.Repaired, "已移除 references 中的疑似完整产物 JSON")
			}
			continue
		}
		out = appendUniqueRoleHandoffStrings(out, item)
		if len(out) >= 24 {
			break
		}
	}
	return out
}

func workspaceRolePacketIsPlaceholderDirectory(value string) bool {
	value = strings.TrimSpace(strings.Trim(value, "`'\" "))
	value = strings.Trim(value, "/")
	switch strings.ToLower(value) {
	case ".", "./", "current directory", "current_dir", "cwd", "当前目录", "当前工作台目录", "本目录", "这里", "当前":
		return true
	default:
		return false
	}
}

func workspaceRolePacketIsMeaninglessReference(value string) bool {
	value = strings.TrimSpace(strings.Trim(value, "`'\" "))
	if value == "" || value == "/" {
		return true
	}
	switch strings.ToLower(value) {
	case "none", "nil", "null", "n/a", "na", "无", "暂无", "无参考", "参考资料", "相关文件", "完整产物":
		return true
	default:
		return false
	}
}

func workspaceRolePacketLooksLikeFullArtifact(value string) bool {
	text := strings.TrimSpace(value)
	if len([]rune(text)) < 80 {
		return false
	}
	lower := strings.ToLower(text)
	if !(strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[")) {
		return false
	}
	return (strings.Contains(lower, `"kind"`) && strings.Contains(lower, `agent_app_`) && strings.Contains(lower, `"project"`)) ||
		(strings.Contains(lower, `"project"`) && strings.Contains(lower, `"tables"`)) ||
		(strings.Contains(lower, `"forms"`) && strings.Contains(lower, `"charts"`))
}

func workspaceRolePacketHasTaskSignal(items []string) bool {
	text := strings.ToLower(strings.Join(items, "；"))
	for _, keyword := range []string{"用户", "目标", "需求", "上一阶段", "阶段", "已确认", "prd", "build", "构建", "测试", "修复", "操作"} {
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func trimWorkspaceExecutedHooks(items []workspaceExecutedRoleHook, limit int) []workspaceExecutedRoleHook {
	out := []workspaceExecutedRoleHook{}
	seen := map[string]struct{}{}
	for _, item := range items {
		item.ID = strings.TrimSpace(item.ID)
		item.Stage = strings.TrimSpace(item.Stage)
		item.SourceRole = normalizeWorkspaceRole(item.SourceRole)
		item.TargetRole = normalizeWorkspaceRole(item.TargetRole)
		item.Status = strings.TrimSpace(item.Status)
		item.Note = compactText(item.Note, 220)
		item.Produced = trimRoleHandoffStrings(item.Produced, 8)
		if item.ID == "" && item.Stage == "" && item.Status == "" && item.Note == "" {
			continue
		}
		key := strings.Join([]string{item.ID, item.Stage, item.SourceRole, item.TargetRole, item.Status, item.Note}, "|")
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}

func workspaceRoleHandoffPacketHasValidationErrors(packet *workspaceRoleHandoffPacket) bool {
	if packet == nil {
		return true
	}
	return packet.Validation.Status == "error" || len(packet.Validation.Errors) > 0
}

func workspaceRoleHandoffPacketValidationSummary(packet *workspaceRoleHandoffPacket) string {
	if packet == nil {
		return "handoff_packet 为空"
	}
	if len(packet.Validation.Errors) > 0 {
		return strings.Join(packet.Validation.Errors, "；")
	}
	if len(packet.Validation.Warnings) > 0 {
		return strings.Join(packet.Validation.Warnings, "；")
	}
	return packet.Validation.Status
}
