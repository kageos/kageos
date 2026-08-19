package service

import (
	"context"
	"strings"
)

func runWorkspacePRDToDeveloperHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	out := workspaceRoleHookOutput{}
	sourceRole := normalizeWorkspaceRole(input.SourceRole)
	markdown := renderWorkspacePRDExecutionMarkdown(input.Artifact, input.ExecuteDirectory, input.TargetAppDirectory)
	if strings.TrimSpace(markdown) == "" {
		return out
	}
	out.PRDExecutionMarkdown = markdown
	note := "已根据 agent_app_prd JSON 生成开发执行视图；当前会话完整历史持续保留，执行视图只用于突出开发重点。"
	if sourceRole == "" {
		note += " 来源角色未记录，按目标角色和产物类型兼容触发。"
	}
	out.ExecutedHooks = append(out.ExecutedHooks, workspaceExecutedRoleHook{
		ID:         workspaceRoleHookProductManagerToDeveloper,
		Stage:      workspaceRoleHookStageBeforeHandoff,
		SourceRole: sourceRole,
		TargetRole: WorkspaceRoleAppDeveloper,
		Status:     "ok",
		Produced:   []string{"PRD_EXECUTION_MARKDOWN"},
		Note:       note,
	})
	return out
}

func runWorkspaceAppOperatorCapabilitiesHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	snapshot := buildWorkspaceAppOperatorCapabilitySnapshot(ctx, input)
	return workspaceRoleHookOutput{
		AppCapabilities:       snapshot,
		HandoffKeyInformation: workspaceAppCapabilityHandoffLines(snapshot),
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookAppOperatorCapabilities,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleAppOperator,
				Status:     firstNonEmptyString(snapshot.Status, "skipped"),
				Produced:   []string{"available_capabilities", "operation_schema_summary"},
				Note:       workspaceAppCapabilityHookNote(snapshot),
			},
		},
	}
}

func runWorkspaceBuildEngineerDiagnosticsHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	diagnostics := buildWorkspaceDiagnostics(workspaceBuildErrorTextFromHandoff(input.Handoff), input.ExecuteDirectory)
	return workspaceRoleHookOutput{
		BuildDiagnostics:      diagnostics,
		HandoffKeyInformation: workspaceBuildDiagnosticsHandoffLines(diagnostics),
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookBuildEngineerDiagnostics,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleBuildEngineer,
				Status:     firstNonEmptyString(diagnostics.Status, "empty"),
				Produced:   []string{"build_diagnostics", "required_docs", "repair_policy", "executed_hooks"},
				Note:       workspaceBuildDiagnosticsHookNote(diagnostics),
			},
		},
	}
}

func runWorkspaceMaintenanceScopeHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	lines := workspaceMaintenanceScopeHandoffLines(input)
	status := "ok"
	if normalizeWorkspacePath(input.ExecuteDirectory) == "" {
		status = "empty"
	}
	return workspaceRoleHookOutput{
		HandoffKeyInformation: lines,
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookMaintenanceScope,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleMaintenanceEngineer,
				Status:     status,
				Produced:   []string{"maintenance_scope"},
				Note:       workspaceMaintenanceScopeHookNote(input),
			},
		},
	}
}

func runWorkspaceQABeforeEnterSchemaHook(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	_ = ctx
	lines := workspaceQAVerificationPlanHandoffLines(input)
	status := "ok"
	if normalizeWorkspacePath(input.ExecuteDirectory) == "" {
		status = "empty"
	}
	return workspaceRoleHookOutput{
		HandoffKeyInformation: lines,
		ExecutedHooks: []workspaceExecutedRoleHook{
			{
				ID:         workspaceRoleHookQABeforeEnterSchema,
				Stage:      workspaceRoleHookStageBeforeEnter,
				SourceRole: normalizeWorkspaceRole(input.SourceRole),
				TargetRole: WorkspaceRoleQAEngineer,
				Status:     status,
				Produced:   []string{"test_capability_snapshot", "verification_plan"},
				Note:       workspaceQABeforeEnterSchemaHookNote(input),
			},
		},
	}
}

func shouldRunWorkspacePRDToDeveloperHook(input workspaceRoleHookInput) bool {
	if strings.TrimSpace(input.Stage) != workspaceRoleHookStageBeforeHandoff {
		return false
	}
	if strings.TrimSpace(input.ArtifactKind) != "agent_app_prd" {
		return false
	}
	if normalizeWorkspaceRole(input.TargetRole) != WorkspaceRoleAppDeveloper {
		return false
	}
	if len(input.Artifact) == 0 {
		return false
	}
	sourceRole := normalizeWorkspaceRole(input.SourceRole)
	return sourceRole == "" || sourceRole == WorkspaceRoleProductManager
}

func shouldRunWorkspaceAppOperatorCapabilitiesHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleAppOperator
}

func shouldRunWorkspaceBuildEngineerDiagnosticsHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleBuildEngineer
}

func shouldRunWorkspaceMaintenanceScopeHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleMaintenanceEngineer
}

func shouldRunWorkspaceQABeforeEnterSchemaHook(input workspaceRoleHookInput) bool {
	return strings.TrimSpace(input.Stage) == workspaceRoleHookStageBeforeEnter &&
		normalizeWorkspaceRole(input.TargetRole) == WorkspaceRoleQAEngineer
}
