package service

import workspaceprd "github.com/kageos/kageos/core/agent-server/workspace/prd"

const workspaceBuildArtifactKind = "agent_app_build"

type workspaceStageInteraction struct {
	ArtifactKind        string   `json:"artifact_kind" schema_desc:"阶段产物类型" schema_required:"true"`
	Status              string   `json:"status" schema_desc:"交互状态，例如 pending_confirmation/pending_test" schema_required:"true"`
	TargetRoleOnConfirm string   `json:"target_role_on_confirm" schema_desc:"确认后固定交接目标角色" schema_required:"true"`
	AllowedActions      []string `json:"allowed_actions" schema_desc:"允许的交互动作，例如 confirm_prd/start_test/cancel" schema_required:"true"`
	ConfirmText         string   `json:"confirm_text" schema_desc:"确认按钮文案" schema_required:"true"`
	ReviseText          string   `json:"revise_text" schema_desc:"修改按钮文案" schema_required:"true"`
	CancelText          string   `json:"cancel_text" schema_desc:"取消按钮文案" schema_required:"true"`
	HelpText            string   `json:"help_text" schema_desc:"看不到按钮时的文本兜底提示" schema_required:"true"`
}

type writePRDInteraction = workspaceStageInteraction

func pendingPRDInteraction() *workspaceStageInteraction {
	return &workspaceStageInteraction{
		ArtifactKind:        workspaceprd.Kind,
		Status:              "pending_confirmation",
		TargetRoleOnConfirm: WorkspaceRoleAppDeveloper,
		AllowedActions:      []string{"confirm_prd", "revise_prd", "cancel_prd", "view_prd"},
		ConfirmText:         "确认 PRD",
		ReviseText:          "修改 PRD",
		CancelText:          "取消 PRD",
		HelpText:            "PRD 已生成，请确认后进入开发；看不到按钮也可以直接回复：确认 PRD / 修改 PRD：xxx / 取消 PRD。",
	}
}

func pendingBuildTestInteraction() *workspaceStageInteraction {
	return &workspaceStageInteraction{
		ArtifactKind:        workspaceBuildArtifactKind,
		Status:              "pending_test",
		TargetRoleOnConfirm: WorkspaceRoleQAEngineer,
		AllowedActions:      []string{"start_test", "continue_development", "skip_test", "view_build"},
		ConfirmText:         "开始测试",
		ReviseText:          "继续修改",
		CancelText:          "暂不测试",
		HelpText:            "应用已编译部署，请进入测试工程师验证；看不到按钮也可以直接回复：开始测试 / 测试 / 暂不测试。",
	}
}
