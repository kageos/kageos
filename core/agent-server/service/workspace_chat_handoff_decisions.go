package service

import (
	"encoding/json"
	"fmt"
	"strings"
)

func workspaceHandoffConfirmedScope(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	out := []string{}
	if project := firstNonEmptyString(digest.ProjectName, digest.Summary); project != "" {
		out = append(out, "项目："+project)
	}
	for _, table := range digest.Tables {
		out = append(out, compactText("表格："+resourceName(table)+"；字段："+strings.Join(table.Fields, "、"), 220))
	}
	for _, form := range digest.Forms {
		note := "表单：" + resourceName(form)
		if form.TargetTable != "" {
			note += "；写入：" + form.TargetTable
		}
		out = append(out, compactText(note, 220))
	}
	for _, chart := range digest.Charts {
		note := "图表：" + resourceName(chart)
		if chart.SourceTable != "" {
			note += "；来源：" + chart.SourceTable
		}
		if chart.ChartType != "" {
			note += "；类型：" + chart.ChartType
		}
		out = append(out, compactText(note, 220))
	}
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffKeyDecisions(artifactKind, targetRole string, digest *workspaceArtifactDigest, remark string) []string {
	role := normalizeWorkspaceRole(targetRole)
	out := []string{workspaceHandoffBaseDecision(artifactKind, role)}
	switch {
	case artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper:
		out = appendUniqueWorkspaceString(out, "完整 PRD JSON 和 PRD_EXECUTION_MARKDOWN 已随本次 agent_app_prd artifact 传入；JSON 是唯一精确需求源，Markdown 是开发执行视图。", 12)
	case artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer:
		out = appendUniqueWorkspaceString(out, "构建产物 JSON 已随本次 agent_app_build artifact 传入；测试以目标应用目录和函数 schema 为准。", 12)
	}
	for _, rule := range digestRules(digest) {
		out = appendUniqueWorkspaceString(out, rule, 12)
	}
	if remark = strings.TrimSpace(remark); remark != "" {
		out = appendUniqueWorkspaceString(out, "用户确认备注："+compactText(remark, 220), 12)
	}
	return out
}

func workspaceHandoffBaseDecision(artifactKind, role string) string {
	switch {
	case artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper:
		return "已确认 PRD，开发阶段不重新设计 PRD、不再次询问确认，除非结构化产物缺失关键字段。"
	case artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer:
		return "已确认构建产物，测试阶段不修改代码、不重新 build；按交接产物和当前工作区函数清单验证。"
	default:
		target := workspaceHandoffRoleLabel(role)
		return fmt.Sprintf("已确认%s，进入%s阶段；下一角色只基于交接摘要、结构化产物、用户补充备注和参考资料推进。", workspaceHandoffConfirmedArtifactText(artifactKind), target)
	}
}

func workspaceHandoffConstraints(artifactKind, targetRole string, digest *workspaceArtifactDigest, notes []string) []string {
	role := normalizeWorkspaceRole(targetRole)
	out := workspaceHandoffBaseConstraints(artifactKind, role)
	if artifactKind == "agent_app_prd" && digest != nil && len(digest.Tables) > 0 {
		out = append(out, "tables.fields 是业务模型字段；tables.search_fields 是查询请求字段，不自动生成业务列。")
	}
	out = append(out, workspaceHandoffFilteredNotes(notes, digestRules(digest), []string{"必须", "不能", "不要", "只允许", "限制"})...)
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffBaseConstraints(artifactKind, role string) []string {
	switch {
	case artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper:
		return []string{
			"业务能力必须落地为 kageos SDK Go 应用，不生成独立 HTML/CSS/JS 页面。",
			"PRD v2 只消费 project/tables/forms/charts/rules；不要回退到旧 models/functions/workflow。",
		}
	case artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer:
		return []string{
			"测试阶段只验证现有构建产物和工作区函数，不直接修改业务代码。",
			"测试结论必须区分测试数据问题、业务缺陷、构建/schema 问题和环境问题。",
		}
	default:
		return []string{
			"不要携带来源会话完整历史，只依据交接摘要、结构化产物、用户补充备注和参考资料推进。",
		}
	}
}

func workspaceHandoffBuildDiagnostics(artifactKind string, artifact map[string]interface{}, artifactJSON string) *workspaceBuildDiagnostics {
	if strings.TrimSpace(artifactKind) != workspaceBuildFailureKind {
		return nil
	}
	if raw, ok := artifact["build_diagnostics"]; ok && raw != nil {
		data, err := json.Marshal(raw)
		if err == nil {
			var diagnostics workspaceBuildDiagnostics
			if err := json.Unmarshal(data, &diagnostics); err == nil && strings.TrimSpace(diagnostics.Status) != "" {
				return &diagnostics
			}
		}
	}
	errText := firstNonEmptyString(workspaceStringField(artifact, "error"), workspaceStringField(artifact, "content"), artifactJSON)
	return buildWorkspaceDiagnostics(errText, workspaceStringField(artifact, "workspace_path"))
}

func workspaceHandoffBuildFailureDecisions(artifactKind, targetRole string, diagnostics *workspaceBuildDiagnostics) []string {
	if artifactKind != workspaceBuildFailureKind || normalizeWorkspaceRole(targetRole) != WorkspaceRoleBuildEngineer || diagnostics == nil {
		return nil
	}
	out := []string{"构建失败诊断已随 agent_app_build_failure artifact 传入；修复阶段以 build_diagnostics 和完整 build_workspace 错误为准。"}
	if len(diagnostics.Categories) > 0 {
		out = append(out, "构建错误类型："+strings.Join(trimWorkspaceStrings(diagnostics.Categories, 8), "、"))
	}
	if len(diagnostics.Routers) > 0 {
		out = append(out, "涉及 router："+strings.Join(trimWorkspaceStrings(diagnostics.Routers, 8), "、"))
	}
	if len(diagnostics.FieldIssues) > 0 {
		fields := []string{}
		for _, issue := range diagnostics.FieldIssues {
			fields = appendUniqueWorkspaceString(fields, compactText(fmt.Sprintf("%s(%s): %s", issue.Field, issue.JSONName, issue.Message), 180), 5)
		}
		out = append(out, "字段问题："+strings.Join(fields, "；"))
	}
	if len(diagnostics.RequiredDocs) > 0 {
		out = append(out, "修复前必读："+strings.Join(trimWorkspaceStrings(diagnostics.RequiredDocs, 6), "、"))
	}
	return trimWorkspaceStrings(out, 8)
}

func workspaceHandoffBuildFailureConstraints(artifactKind, targetRole string) []string {
	if artifactKind != workspaceBuildFailureKind || normalizeWorkspaceRole(targetRole) != WorkspaceRoleBuildEngineer {
		return nil
	}
	return []string{
		"构建修复阶段只处理 build/schema/widget/SDK API/路由注册等构建问题，不重新设计 PRD。",
		"同类错误第二次出现前必须补读 required_docs、匹配案例或相关 SDK 源码；不要继续同一方案重试。",
	}
}

func workspaceHandoffBuildRepairFocus(artifactKind, targetRole string, diagnostics *workspaceBuildDiagnostics) []string {
	if artifactKind != workspaceBuildFailureKind || normalizeWorkspaceRole(targetRole) != WorkspaceRoleBuildEngineer {
		return nil
	}
	out := []string{
		"先读完整 build_workspace 错误，再按 router/字段/文件归类同类问题。",
		"小范围批量修复后重新 build_workspace；不要扩大到整个工作空间测试。",
	}
	if diagnostics != nil {
		out = append(out, trimWorkspaceStrings(diagnostics.RepairPolicy, 5)...)
		if diagnostics.RetryPolicy != "" {
			out = append(out, diagnostics.RetryPolicy)
		}
	}
	return trimWorkspaceStrings(out, 8)
}

func workspaceHandoffArtifactLabel(artifactKind string) string {
	switch strings.TrimSpace(artifactKind) {
	case "agent_app_prd":
		return "PRD"
	case workspaceBuildArtifactKind:
		return "构建产物"
	case workspaceBuildFailureKind:
		return "构建失败诊断"
	case "":
		return "阶段产物"
	default:
		return artifactKind
	}
}

func workspaceHandoffConfirmedArtifactText(artifactKind string) string {
	label := workspaceHandoffArtifactLabel(artifactKind)
	if label == "PRD" || strings.Contains(label, "_") {
		return " " + label
	}
	return label
}

func workspaceHandoffRoleLabel(role string) string {
	role = normalizeWorkspaceRole(role)
	if label := strings.TrimSpace(workspaceRoleDisplayName(role)); label != "" {
		return label
	}
	if role != "" {
		return role
	}
	return "下一角色"
}

func workspaceHandoffWorkflowNotes(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	out := []string{}
	if len(digest.Tables) > 0 {
		out = append(out, "先生成基础/配置表和可维护 Table。")
	}
	for _, form := range digest.Forms {
		if form.TargetTable != "" {
			out = append(out, compactText("Form "+resourceName(form)+" 提交后应写入目标表 "+form.TargetTable+"，再通过对应记录表查询验证。", 220))
		}
	}
	for _, chart := range digest.Charts {
		if chart.SourceTable != "" {
			out = append(out, compactText("Chart "+resourceName(chart)+" 基于 "+chart.SourceTable+" 统计，不能只返回静态示例。", 220))
		}
	}
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffDataModelNotes(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	out := []string{}
	for _, table := range digest.Tables {
		if len(table.Fields) > 0 {
			out = append(out, compactText(resourceName(table)+" 业务字段："+strings.Join(table.Fields, "、"), 220))
		}
		if len(table.SearchFields) > 0 {
			out = append(out, compactText(resourceName(table)+" 查询字段："+strings.Join(table.SearchFields, "、"), 220))
		}
		if len(table.Handlers) == 0 {
			out = append(out, resourceName(table)+" 为只读查询表，不要补新增、编辑、删除。")
		}
	}
	return trimWorkspaceStrings(out, 16)
}

func workspaceHandoffImplementationFocus(artifactKind, targetRole string, digest *workspaceArtifactDigest) []string {
	role := normalizeWorkspaceRole(targetRole)
	if artifactKind != "agent_app_prd" || role != WorkspaceRoleAppDeveloper {
		return nil
	}
	out := []string{
		"先使用 change_role 返回的 app_developer 角色文档包和 SDK 主文档；再读取 1 到多个匹配案例。",
		"按 PRD 派生目录、Go 文件、Table/Form/Chart 路由和 build，不重新输出 PRD。",
		"严格区分业务字段和搜索字段；创建开始时间/创建结束时间/创建人默认映射系统字段。",
	}
	if digest != nil {
		if len(digest.Forms) > 0 {
			out = append(out, "Form 必须按 target_table 产生可查询记录；目标记录表默认只读，除非 PRD 明确维护能力。")
		}
		if len(digest.Charts) > 0 {
			out = append(out, "Chart 必须基于 source_table、filters 和 examples 实现真实统计。")
		}
	}
	return out
}

func workspaceHandoffVerificationFocus(artifactKind, targetRole string, digest *workspaceArtifactDigest) []string {
	out := []string{}
	if artifactKind == "agent_app_prd" && normalizeWorkspaceRole(targetRole) == WorkspaceRoleAppDeveloper {
		out = append(out, "build 成功后进入测试阶段，按基础表 -> Form -> 记录表 -> Chart 顺序验证。")
	}
	if digest != nil {
		for _, table := range digest.Tables {
			if len(table.SearchFields) > 0 {
				out = append(out, compactText("验证 "+resourceName(table)+" 的核心筛选："+strings.Join(table.SearchFields, "、"), 220))
			}
		}
		for _, form := range digest.Forms {
			if form.TargetTable != "" {
				out = append(out, compactText("验证 "+resourceName(form)+" 提交后在 "+form.TargetTable+" 可查到记录。", 220))
			}
		}
	}
	return trimWorkspaceStrings(out, 12)
}

func workspaceHandoffReferenceDocs(targetRole, artifactKind string) []string {
	role := normalizeWorkspaceRole(targetRole)
	docs := []string{}
	if definition, ok := workspaceRoleDefinitionFor(role); ok {
		docs = append([]string(nil), definition.DocumentPackage...)
	}
	if artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper {
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/case_catalog", 0)
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/sdk/agent-app-sdk-readme", 0)
	}
	if artifactKind == workspaceBuildArtifactKind && role == WorkspaceRoleQAEngineer {
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/roles/qa-engineer", 0)
	}
	if artifactKind == workspaceBuildFailureKind && role == WorkspaceRoleBuildEngineer {
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/roles/build-engineer", 0)
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/sdk/reference/build-validation", 0)
		docs = appendUniqueWorkspaceString(docs, "/system/prompt/sdk/agent-app-sdk-readme", 0)
	}
	return trimWorkspaceStrings(docs, 12)
}

func workspaceHandoffReferenceFiles(fullCodePath, workspaceDirectory, targetAppDirectory, artifactKind, targetRole string, digest *workspaceArtifactDigest) []string {
	out := []string{}
	role := normalizeWorkspaceRole(targetRole)
	if artifactKind == "agent_app_prd" && role == WorkspaceRoleAppDeveloper {
		if path := strings.TrimSpace(workspaceDirectory); path != "" {
			out = appendUniqueWorkspaceString(out, path, 16)
		}
	} else if path := strings.TrimSpace(fullCodePath); path != "" {
		out = appendUniqueWorkspaceString(out, path, 16)
	}
	if path := strings.TrimSpace(targetAppDirectory); path != "" {
		out = appendUniqueWorkspaceString(out, path, 16)
	}
	if digest != nil {
		for _, table := range digest.Tables {
			if table.Code != "" {
				out = appendUniqueWorkspaceString(out, workspaceHandoffReferenceFilePath(targetAppDirectory, table.Code+".go"), 16)
			}
		}
		for _, form := range digest.Forms {
			if form.Code != "" {
				out = appendUniqueWorkspaceString(out, workspaceHandoffReferenceFilePath(targetAppDirectory, form.Code+".go"), 16)
			}
		}
		for _, chart := range digest.Charts {
			if chart.Code != "" {
				out = appendUniqueWorkspaceString(out, workspaceHandoffReferenceFilePath(targetAppDirectory, chart.Code+".go"), 16)
			}
		}
	}
	return out
}

func workspaceHandoffReferenceFilePath(targetAppDirectory, fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return ""
	}
	if target := normalizeWorkspacePath(targetAppDirectory); target != "" {
		return target + "/" + fileName
	}
	return fileName
}

func workspaceHandoffFilteredNotes(notes []string, rules []string, keywords []string) []string {
	out := []string{}
	for _, item := range append(notes, rules...) {
		if workspaceContainsAny(item, keywords) {
			out = appendUniqueWorkspaceString(out, compactText(item, 220), 8)
		}
	}
	return out
}

func workspaceContainsAny(s string, keywords []string) bool {
	for _, keyword := range keywords {
		if keyword != "" && strings.Contains(s, keyword) {
			return true
		}
	}
	return false
}

func digestRules(digest *workspaceArtifactDigest) []string {
	if digest == nil {
		return nil
	}
	return digest.Rules
}

func resourceName(r workspaceResourceDigest) string {
	return firstNonEmptyString(r.Name, r.Code, "未命名资源")
}
