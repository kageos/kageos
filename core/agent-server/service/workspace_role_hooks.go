package service

import (
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
)

const (
	workspaceRoleHookStageBeforeHandoff        = "before_handoff"
	workspaceRoleHookProductManagerToDeveloper = "product_manager.to_app_developer"
)

type workspaceRoleHookInput struct {
	Stage              string
	SourceRole         string
	TargetRole         string
	ArtifactKind       string
	Artifact           map[string]interface{}
	FullCodePath       string
	WorkspaceDirectory string
	TargetAppDirectory string
	ExecuteDirectory   string
	Messages           []*model.AgentChatMessage
}

type workspaceRoleHookOutput struct {
	PRDExecutionMarkdown string
	ExecutedHooks        []workspaceExecutedRoleHook
}

type workspaceExecutedRoleHook struct {
	ID         string   `json:"id"`
	Stage      string   `json:"stage"`
	SourceRole string   `json:"source_role,omitempty"`
	TargetRole string   `json:"target_role,omitempty"`
	Status     string   `json:"status"`
	Produced   []string `json:"produced,omitempty"`
	Note       string   `json:"note,omitempty"`
}

func runWorkspaceRoleHooks(input workspaceRoleHookInput) workspaceRoleHookOutput {
	out := workspaceRoleHookOutput{}
	if shouldRunWorkspacePRDToDeveloperHook(input) {
		sourceRole := normalizeWorkspaceRole(input.SourceRole)
		markdown := renderWorkspacePRDExecutionMarkdown(input.Artifact, input.ExecuteDirectory, input.TargetAppDirectory)
		if strings.TrimSpace(markdown) != "" {
			out.PRDExecutionMarkdown = markdown
			note := "已根据 agent_app_prd JSON 生成开发执行视图；目标模型不接收来源会话完整历史。"
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
		}
	}
	return out
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
