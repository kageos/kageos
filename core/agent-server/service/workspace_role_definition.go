package service

import (
	"slices"

	"github.com/kageos/kageos/core/agent-server/prompt"
)

type workspaceRoleDefinition struct {
	ID                 string              `json:"id" schema_desc:"角色 ID" schema_required:"true"`
	DisplayName        string              `json:"display_name" schema_desc:"角色展示名称" schema_required:"true"`
	Responsibility     string              `json:"responsibility" schema_desc:"角色职责边界" schema_required:"true"`
	RouteDescription   string              `json:"route_description,omitempty" schema_desc:"路由说明"`
	RequiredDocs       []string            `json:"required_docs" schema_desc:"必读角色文档" schema_required:"true"`
	OptionalDocs       []string            `json:"optional_docs,omitempty" schema_desc:"按场景读取的补充文档"`
	DocumentPackage    []string            `json:"document_package" schema_desc:"进入角色时实际加载的文档包" schema_required:"true"`
	AllowedTools       []string            `json:"allowed_tools" schema_desc:"该角色允许使用的工具" schema_required:"true"`
	ForbiddenTools     []string            `json:"forbidden_tools,omitempty" schema_desc:"该角色禁止使用的工具"`
	RuntimeContract    roleRuntimeContract `json:"runtime_contract" schema_desc:"进入条件、SOP、完成标准和 Hook" schema_required:"true"`
	HandoffRequired    []string            `json:"handoff_required" schema_desc:"切换到该角色必须携带的交接字段" schema_required:"true"`
	AllowedTransitions []nextWorkspaceRole `json:"allowed_transitions,omitempty" schema_desc:"该角色完成后建议切换的目标角色"`
	DefaultNextAction  string              `json:"default_next_action" schema_desc:"进入角色后的默认下一步动作" schema_required:"true"`
	ProtocolVersion    string              `json:"protocol_version" schema_desc:"角色协议版本" schema_required:"true"`
}

const workspaceRoleDefinitionProtocolVersion = "role_definition.v1"

func workspaceRoleDefinitionFor(role string) (workspaceRoleDefinition, bool) {
	spec, ok := workspaceRoleSpecFor(role)
	if !ok {
		return workspaceRoleDefinition{}, false
	}
	return buildWorkspaceRoleDefinition(spec), true
}

func buildWorkspaceRoleDefinition(spec workspaceRoleSpec) workspaceRoleDefinition {
	runtime := annotateWorkspaceRoleRuntimeContractHooks(spec.Runtime)
	if len(runtime.HandoffRequired) == 0 {
		runtime.HandoffRequired = []string{"execute_directory", "task_context", "key_information", "references"}
	}
	return workspaceRoleDefinition{
		ID:                 spec.ID,
		DisplayName:        spec.DisplayName,
		Responsibility:     spec.Action,
		RouteDescription:   spec.RouteDescription,
		RequiredDocs:       normalizeWorkspaceRoleDocPaths(spec.Docs),
		OptionalDocs:       normalizeWorkspaceRoleDocPaths(spec.Optional),
		DocumentPackage:    buildWorkspaceRoleDocumentPackage(spec.ID, spec),
		AllowedTools:       workspaceRoleAllowedToolsForSpec(spec),
		ForbiddenTools:     append([]string(nil), spec.ForbiddenTools...),
		RuntimeContract:    runtime,
		HandoffRequired:    append([]string(nil), runtime.HandoffRequired...),
		AllowedTransitions: append([]nextWorkspaceRole(nil), spec.NextRoles...),
		DefaultNextAction:  spec.Action,
		ProtocolVersion:    workspaceRoleDefinitionProtocolVersion,
	}
}

func workspaceRoleAllowedToolsForSpec(spec workspaceRoleSpec) []string {
	allowed := append([]string(nil), workspaceRoleBaseReadOnlyTools()...)
	for _, tool := range spec.AllowedTools {
		if !containsWorkspaceRoleString(allowed, tool) {
			allowed = append(allowed, tool)
		}
	}
	return allowed
}

func buildWorkspaceRoleDocumentPackage(role string, spec workspaceRoleSpec) []string {
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
		addDoc("/system/prompt/platform-capability-boundaries")
	}
	return docs
}

func normalizeWorkspaceRoleDocPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		path = prompt.NormalizePromptDocPath(path)
		if path == "" || slices.Contains(out, path) {
			continue
		}
		out = append(out, path)
	}
	return out
}
