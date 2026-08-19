package service

import (
	workspaceprd "github.com/kageos/kageos/core/agent-server/workspace/prd"
)

const workspaceBuildArtifactKind = "agent_app_build"

type workspaceStageInteraction struct {
	CardType            string   `json:"card_type" schema_desc:"交互卡片类型，例如 prd_confirmation/build_repair/question_batch" schema_required:"true"`
	ArtifactKind        string   `json:"artifact_kind" schema_desc:"阶段产物类型" schema_required:"true"`
	Status              string   `json:"status" schema_desc:"交互状态，例如 pending_confirmation" schema_required:"true"`
	Blocking            bool     `json:"blocking" schema_desc:"是否阻塞普通工作台对话" schema_required:"true"`
	Title               string   `json:"title" schema_desc:"卡片标题" schema_required:"true"`
	Description         string   `json:"description" schema_desc:"卡片说明" schema_required:"true"`
	TargetRoleOnConfirm string   `json:"target_role_on_confirm" schema_desc:"确认后固定交接目标角色" schema_required:"true"`
	AllowedActions      []string `json:"allowed_actions" schema_desc:"允许的交互动作，例如 confirm_prd/start_build_repair/cancel" schema_required:"true"`
	ViewText            string   `json:"view_text" schema_desc:"查看按钮文案" schema_required:"true"`
	ConfirmText         string   `json:"confirm_text" schema_desc:"确认按钮文案" schema_required:"true"`
	ReviseText          string   `json:"revise_text" schema_desc:"修改按钮文案" schema_required:"true"`
	CancelText          string   `json:"cancel_text" schema_desc:"取消按钮文案" schema_required:"true"`
	HelpText            string   `json:"help_text" schema_desc:"看不到按钮时的文本兜底提示" schema_required:"true"`
}

type writePRDInteraction = workspaceStageInteraction

func pendingPRDInteraction() *workspaceStageInteraction {
	return &workspaceStageInteraction{
		CardType:            "prd_confirmation",
		ArtifactKind:        workspaceprd.Kind,
		Status:              "pending_confirmation",
		Blocking:            true,
		Title:               "PRD 等待确认",
		Description:         "确认后进入开发；如需求不完整，请一次性写清楚要修改的点。",
		TargetRoleOnConfirm: WorkspaceRoleAppDeveloper,
		AllowedActions:      []string{"confirm_prd", "revise_prd", "cancel_prd", "view_prd"},
		ViewText:            "查看 PRD",
		ConfirmText:         "确认 PRD",
		ReviseText:          "修改 PRD",
		CancelText:          "取消 PRD",
		HelpText:            "PRD 已生成，请确认后进入开发；看不到按钮也可以直接回复：确认 PRD / 修改 PRD：xxx / 取消 PRD。",
	}
}
