package service

import (
	"context"
	"strings"

	"github.com/kageos/kageos/core/agent-server/model"
	"github.com/kageos/kageos/pkg/apicall"
)

const (
	workspaceRoleHookStageBeforeHandoff        = "before_handoff"
	workspaceRoleHookStageBeforeEnter          = "before_enter"
	workspaceRoleHookProductManagerToDeveloper = "product_manager.to_app_developer"
	workspaceRoleHookAppOperatorCapabilities   = "app_operator.before_enter_capabilities"
	workspaceRoleHookBuildEngineerDiagnostics  = "build_engineer.before_enter_diagnostics"
	workspaceRoleHookMaintenanceScope          = "maintenance.before_enter_scope"
	workspaceRoleHookQABeforeEnterSchema       = "qa.before_enter_schema"
)

var workspaceRoleHookSearchFunctions = apicall.SearchFunctions

const (
	workspaceRoleHookImplementationImplemented = "implemented"
	workspaceRoleHookImplementationPlanned     = "planned"
)

type workspaceRoleHookRegistration struct {
	ID        string
	Stage     string
	ShouldRun func(workspaceRoleHookInput) bool
	Run       func(context.Context, workspaceRoleHookInput) workspaceRoleHookOutput
}

var workspaceRoleHookRegistry = []workspaceRoleHookRegistration{
	{
		ID:        workspaceRoleHookProductManagerToDeveloper,
		Stage:     workspaceRoleHookStageBeforeHandoff,
		ShouldRun: shouldRunWorkspacePRDToDeveloperHook,
		Run:       runWorkspacePRDToDeveloperHook,
	},
	{
		ID:        workspaceRoleHookMaintenanceScope,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceMaintenanceScopeHook,
		Run:       runWorkspaceMaintenanceScopeHook,
	},
	{
		ID:        workspaceRoleHookQABeforeEnterSchema,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceQABeforeEnterSchemaHook,
		Run:       runWorkspaceQABeforeEnterSchemaHook,
	},
	{
		ID:        workspaceRoleHookAppOperatorCapabilities,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceAppOperatorCapabilitiesHook,
		Run:       runWorkspaceAppOperatorCapabilitiesHook,
	},
	{
		ID:        workspaceRoleHookBuildEngineerDiagnostics,
		Stage:     workspaceRoleHookStageBeforeEnter,
		ShouldRun: shouldRunWorkspaceBuildEngineerDiagnosticsHook,
		Run:       runWorkspaceBuildEngineerDiagnosticsHook,
	},
}

var workspaceRolePlannedHookIDs = map[string]struct{}{
	"product_manager.prd_ready":         {},
	"app_developer.before_enter_prd":    {},
	"app_developer.after_build":         {},
	"maintenance.after_build":           {},
	"automation.before_enter_scope":     {},
	"qa.after_run":                      {},
	"app_operator.after_run":            {},
	"build_engineer.after_build":        {},
	"data_operator.before_enter_inputs": {},
	"platform.before_enter_boundary":    {},
	"reviewer.before_handoff":           {},
}

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
	Handoff            roleHandoffData
	Messages           []*model.AgentChatMessage
}

type workspaceRoleHookOutput struct {
	PRDExecutionMarkdown  string
	AppCapabilities       *workspaceAppCapabilitySnapshot
	BuildDiagnostics      *workspaceBuildDiagnostics
	ExecutedHooks         []workspaceExecutedRoleHook
	HandoffKeyInformation []string
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

type workspaceAppCapabilitySnapshot struct {
	Status             string                           `json:"status"`
	ExecuteDirectory   string                           `json:"execute_directory,omitempty"`
	Scope              string                           `json:"scope,omitempty"`
	User               string                           `json:"user,omitempty"`
	App                string                           `json:"app,omitempty"`
	Keyword            string                           `json:"keyword,omitempty"`
	TotalFunctions     int                              `json:"total_functions"`
	DisplayedFunctions int                              `json:"displayed_functions"`
	Counts             workspaceAppCapabilityCounts     `json:"counts"`
	Functions          []workspaceAppFunctionCapability `json:"functions,omitempty"`
	Guidance           []string                         `json:"guidance,omitempty"`
	Error              string                           `json:"error,omitempty"`
}

type workspaceAppCapabilityCounts struct {
	Tables int `json:"tables"`
	Forms  int `json:"forms"`
	Charts int `json:"charts"`
}

type workspaceAppFunctionCapability struct {
	Name          string   `json:"name,omitempty"`
	Code          string   `json:"code,omitempty"`
	FullCodePath  string   `json:"full_code_path,omitempty"`
	Type          string   `json:"type,omitempty"`
	Capabilities  string   `json:"capabilities,omitempty"`
	RunTools      []string `json:"run_tools,omitempty"`
	Description   string   `json:"description,omitempty"`
	SchemaSummary []string `json:"schema_summary,omitempty"`
}

func runWorkspaceRoleHooks(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	return runWorkspaceRoleHookRegistry(ctx, input)
}

func runWorkspaceRoleBeforeEnterHooks(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	return runWorkspaceRoleHookRegistry(ctx, input)
}

func runWorkspaceRoleHookRegistry(ctx context.Context, input workspaceRoleHookInput) workspaceRoleHookOutput {
	out := workspaceRoleHookOutput{}
	stage := strings.TrimSpace(input.Stage)
	for _, registration := range workspaceRoleHookRegistry {
		if strings.TrimSpace(registration.Stage) != stage || registration.Run == nil {
			continue
		}
		if registration.ShouldRun != nil && !registration.ShouldRun(input) {
			continue
		}
		mergeWorkspaceRoleHookOutput(&out, registration.Run(ctx, input))
	}
	return out
}

func mergeWorkspaceRoleHookOutput(dst *workspaceRoleHookOutput, src workspaceRoleHookOutput) {
	if dst == nil {
		return
	}
	if strings.TrimSpace(src.PRDExecutionMarkdown) != "" {
		dst.PRDExecutionMarkdown = src.PRDExecutionMarkdown
	}
	if src.AppCapabilities != nil {
		dst.AppCapabilities = src.AppCapabilities
	}
	if src.BuildDiagnostics != nil {
		dst.BuildDiagnostics = src.BuildDiagnostics
	}
	dst.ExecutedHooks = append(dst.ExecutedHooks, src.ExecutedHooks...)
	dst.HandoffKeyInformation = append(dst.HandoffKeyInformation, src.HandoffKeyInformation...)
}

func annotateWorkspaceRoleRuntimeContractHooks(contract roleRuntimeContract) roleRuntimeContract {
	if len(contract.Hooks) == 0 {
		return contract
	}
	for i := range contract.Hooks {
		contract.Hooks[i].ImplementationStatus = workspaceRoleHookImplementationStatus(contract.Hooks[i].ID)
	}
	return contract
}

func workspaceRoleHookImplementationStatus(hookID string) string {
	hookID = strings.TrimSpace(hookID)
	if hookID == "" {
		return ""
	}
	for _, registration := range workspaceRoleHookRegistry {
		if strings.TrimSpace(registration.ID) == hookID {
			return workspaceRoleHookImplementationImplemented
		}
	}
	if _, ok := workspaceRolePlannedHookIDs[hookID]; ok {
		return workspaceRoleHookImplementationPlanned
	}
	return "unknown"
}
