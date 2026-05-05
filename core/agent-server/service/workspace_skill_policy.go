package service

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	agentosskills "github.com/ai-agent-os/ai-agent-os/core/agent-server/skills"
	"github.com/ai-agent-os/ai-agent-os/pkg/llms"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
)

var skillPolicyBootstrapTools = map[string]struct{}{
	"search_skills": {},
	"read_skill":    {},
}

func workspaceSkillToolGateResult(toolName string, args map[string]interface{}, loadedSkills map[string]*agentosskills.Skill, loadedGuideDocs map[string]struct{}) (ToolResult, bool) {
	if currentWorkspaceSkillsMode() != skillsModeOn {
		return ToolResult{}, false
	}
	name := strings.TrimSpace(toolName)
	if isSkillBootstrapTool(name) {
		return ToolResult{}, false
	}
	missingDocs := missingRequiredDocsForSkills(loadedSkills, loadedGuideDocs)
	if name == "read_doc" {
		if gateRes, blocked := workspaceSkillReadDocGateResult(args, loadedSkills); blocked {
			return gateRes, true
		}
		if len(missingDocs) > 0 && !readDocTargetsMissingRequiredDocs(args, missingDocs) {
			return toolResult(missingRequiredDocsGateMessage(name, missingDocs), true), true
		}
		if gateRes, blocked := workspaceSkillAllowedToolsGateResult(name, loadedSkills); blocked {
			return gateRes, true
		}
		return ToolResult{}, false
	}
	if len(missingDocs) > 0 {
		return toolResult(missingRequiredDocsGateMessage(name, missingDocs), true), true
	}
	if gateRes, blocked := workspaceSkillAllowedToolsGateResult(name, loadedSkills); blocked {
		return gateRes, true
	}
	req, ok := skillRequirementForTool(name)
	if !ok || skillRequirementSatisfied(req, loadedSkills) {
		if gateRes, blocked := workspaceSDKTaskPackageGateResult(name, loadedSkills); blocked {
			return gateRes, true
		}
		return ToolResult{}, false
	}
	return toolResult(skillRequirementGateMessage(name, req), true), true
}

func isSkillBootstrapTool(toolName string) bool {
	_, ok := skillPolicyBootstrapTools[strings.TrimSpace(toolName)]
	return ok
}

func shouldSkipAfterSkillGateBlock(toolName string) bool {
	name := strings.TrimSpace(toolName)
	if isSkillBootstrapTool(name) {
		return false
	}
	if _, ok := skillRequirementForTool(name); ok {
		return true
	}
	return true
}

func skillGateBatchSkipResult(toolName string) ToolResult {
	name := strings.TrimSpace(toolName)
	req, ok := skillRequirementForTool(name)
	hint := "读取匹配的 skill"
	if ok && strings.TrimSpace(req.Hint) != "" {
		hint = strings.TrimSpace(req.Hint)
	}
	return toolResult("已跳过工具调用：同一批工具中已有执行/写入类工具因未读取匹配 skill 被拦截。本批次不继续执行 `"+name+"`，避免重复失败。下一步只调用 "+hint+"；读取成功后再按 skill 规范重试。", true)
}

func isWriteMutationTool(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "write_go_file", "search_replace_file", "delete_file", "create_directory":
		return true
	default:
		return false
	}
}

func shouldSkipAfterWriteMutationFailure(toolName string) bool {
	switch strings.TrimSpace(toolName) {
	case "write_go_file", "search_replace_file", "delete_file", "create_directory", "build_workspace":
		return true
	default:
		return false
	}
}

func writeMutationFailureBatchSkipResult(toolName string) ToolResult {
	name := strings.TrimSpace(toolName)
	return toolResult("已跳过工具调用：同一批工具中已有写入/替换/删除/创建目录工具失败，失败的文件或修改未落盘。本批次不继续执行 `"+name+"`，避免基于错误状态继续写代码或编译。下一步先读取相关文件，修正上一条失败工具调用后再继续。", true)
}

func shouldGateWriteAfterBatchLimit(toolName string, successfulWriteGoFilesInBatch int) bool {
	return strings.TrimSpace(toolName) == "write_go_file" && successfulWriteGoFilesInBatch >= maxWriteGoFilesPerToolBatch
}

func writeGoFileBatchLimitResult(toolName string, successfulWriteGoFilesInBatch int) ToolResult {
	name := strings.TrimSpace(toolName)
	if name == "" {
		name = "write_go_file"
	}
	return toolResult("已跳过工具调用：同一批次已成功写入 "+strconv.Itoa(successfulWriteGoFilesInBatch)+" 个 Go 文件，达到分阶段上限。复杂系统必须先对当前阶段调用 `build_workspace`，根据结果修复后再继续写后续 Form/Chart 文件；本次不继续执行 `"+name+"`。", true)
}

func workspaceSkillReadDocGateResult(args map[string]interface{}, loadedSkills map[string]*agentosskills.Skill) (ToolResult, bool) {
	for _, target := range guideDocPathsFromReadDocArgs(args) {
		if isLegacyWorkspacePromptDocPath(target) {
			return toolResult("旧 `/system/prompt/workspace/*` SOP 路径已下线。请先按 Skills 目录调用匹配的 `read_skill(\"<skill id>\")`；读取 skill 后只按其中的 required_docs 读取 SDK、平台总览或案例文档。", true), true
		}
		req, ok := skillRequirementForPromptDoc(target)
		if !ok || skillRequirementSatisfied(req, loadedSkills) {
			continue
		}
		return toolResult(skillRequirementGateMessage("read_doc("+normalizeGuideDocPath(target)+")", req), true), true
	}
	return ToolResult{}, false
}

func workspaceSkillAllowedToolsGateResult(toolName string, loadedSkills map[string]*agentosskills.Skill) (ToolResult, bool) {
	name := strings.TrimSpace(toolName)
	if name == "" || len(loadedSkills) == 0 || isSkillBootstrapTool(name) {
		return ToolResult{}, false
	}
	if skillAllowedToolsContain(name, loadedSkills) {
		return ToolResult{}, false
	}
	return toolResult(skillAllowedToolsGateMessage(name, loadedSkills), true), true
}

func skillAllowedToolsContain(toolName string, loadedSkills map[string]*agentosskills.Skill) bool {
	name := strings.TrimSpace(toolName)
	if name == "" {
		return false
	}
	for _, skill := range loadedSkills {
		if skill == nil {
			continue
		}
		for _, allowed := range skill.Meta.AllowedTools {
			if strings.TrimSpace(allowed) == name {
				return true
			}
		}
	}
	return false
}

func skillAllowedToolsGateMessage(toolName string, loadedSkills map[string]*agentosskills.Skill) string {
	ids := make([]string, 0, len(loadedSkills))
	for _, skill := range loadedSkills {
		if skill == nil || strings.TrimSpace(skill.Meta.ID) == "" {
			continue
		}
		ids = append(ids, strings.TrimSpace(skill.Meta.ID))
	}
	sort.Strings(ids)
	loaded := strings.Join(ids, ",")
	if loaded == "" {
		loaded = "<loaded skills>"
	}
	return "已拦截工具调用：`" + strings.TrimSpace(toolName) + "` 不在当前已读取 skill 的 allowed_tools 中。当前已读取 skill：" + loaded + "。请先读取更匹配的 skill，或只使用当前 skill 明确声明的工具；不要用提示词以外的工具和能力继续写代码。"
}

func workspaceSDKTaskPackageGateResult(toolName string, loadedSkills map[string]*agentosskills.Skill) (ToolResult, bool) {
	name := strings.TrimSpace(toolName)
	switch name {
	case "write_go_file", "search_replace_file", "build_workspace":
	default:
		return ToolResult{}, false
	}
	if !hasLoadedSkillID(loadedSkills, "sop.create-project") && !hasLoadedSkillID(loadedSkills, "sop.modify-project") {
		return ToolResult{}, false
	}
	if hasLoadedSDKImplementationSkill(loadedSkills) {
		return ToolResult{}, false
	}
	return toolResult("已拦截工具调用：`"+name+"` 将写入或编译 SDK Go 应用，但当前只读取了创建/修改 SOP，尚未读取具体 SDK 任务包。请先按场景读取一个任务包 skill：`read_skill(\"sdk.form-submit-basic\")`、`read_skill(\"sdk.table-crud-basic\")`、`read_skill(\"sdk.combo-table-form\")` 或 `read_skill(\"sdk.combo-table-form-chart\")`，再继续写代码；不要只靠案例和经验生成。", true), true
}

func hasLoadedSkillID(loadedSkills map[string]*agentosskills.Skill, targetID string) bool {
	targetID = strings.TrimSpace(targetID)
	if targetID == "" {
		return false
	}
	for _, skill := range loadedSkills {
		if skill == nil {
			continue
		}
		if strings.TrimSpace(skill.Meta.ID) == targetID {
			return true
		}
	}
	return false
}

func hasLoadedSDKImplementationSkill(loadedSkills map[string]*agentosskills.Skill) bool {
	for _, skill := range loadedSkills {
		if skill == nil {
			continue
		}
		switch strings.TrimSpace(skill.Meta.ID) {
		case "sdk.form-submit-basic", "sdk.table-crud-basic", "sdk.combo-table-form", "sdk.combo-table-form-chart", "sdk.create-form-table-chart":
			return true
		}
	}
	return false
}

func skillRequirementForPromptDoc(docPath string) (skillToolRequirement, bool) {
	docPath = normalizeGuideDocPath(docPath)
	switch {
	case promptDocPathMatches(docPath, "/system/prompt/platform-capability-boundaries"),
		promptDocPathMatches(docPath, "/system/prompt/platform-overview"),
		promptDocPathMatches(docPath, "/system/prompt/platform-cross-cutting-capabilities"):
		return skillToolRequirement{
			AnyOf: []string{"sop", "sdk", "system"},
			Hint:  "匹配当前任务的 `read_skill(\"<skill id>\")`；不确定时再用 `search_skills`",
		}, true
	case promptDocPathMatches(docPath, "/system/prompt/sdk/agent-app-sdk-readme"):
		return skillToolRequirement{
			AnyOf: []string{"sop.create-project", "sop.modify-project", "sdk"},
			Hint:  "`read_skill(\"sop.create-project\")`、`read_skill(\"sop.modify-project\")` 或具体 `sdk.*` skill",
		}, true
	default:
		return skillToolRequirement{}, false
	}
}

func promptDocPathMatches(docPath string, required string) bool {
	docPath = normalizeGuideDocPath(docPath)
	required = normalizeGuideDocPath(required)
	return docPath == required || strings.HasPrefix(docPath, required+"/")
}

func isLegacyWorkspacePromptDocPath(docPath string) bool {
	docPath = normalizeGuideDocPath(docPath)
	return docPath == "/system/prompt/workspace" || strings.HasPrefix(docPath, "/system/prompt/workspace/")
}

type skillToolRequirement struct {
	AnyOf []string
	Hint  string
}

func skillRequirementForTool(toolName string) (skillToolRequirement, bool) {
	switch strings.TrimSpace(toolName) {
	case "run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete",
		"run_form_submit", "run_chart_query", "run_on_select_fuzzy":
		return skillToolRequirement{
			AnyOf: []string{
				"sop.execute-function",
				"sop.create-project",
				"sop.modify-project",
				"sdk",
				"system.tools",
				"system.openapi",
			},
			Hint: "`read_skill(\"sop.execute-function\")`；创建/修改/SDK 开发任务也可在读取对应 skill 后验证；如果是官方工具任务读 `system.tools` 或具体 `system.tools.*`；如果是平台接口任务读 `system.openapi` 或具体 `system.openapi.*`",
		}, true
	case "run_official_python":
		return skillToolRequirement{
			AnyOf: []string{"system.tools", "system.tools.runtime", "sop.execute-function"},
			Hint:  "`read_skill(\"system.tools.runtime\")`，或先读 `system.tools` / `sop.execute-function`",
		}, true
	case "create_scheduled_task", "list_scheduled_tasks", "cancel_scheduled_task", "list_scheduled_task_executions",
		"create_scheduled_agent_task", "list_scheduled_agent_tasks", "list_scheduled_agent_task_executions", "run_scheduled_agent_task_now":
		return skillToolRequirement{
			AnyOf: []string{"system.openapi", "system.openapi.scheduled-task", "sop.execute-function"},
			Hint:  "`read_skill(\"system.openapi.scheduled-task\")`，或先读 `system.openapi` / `sop.execute-function`",
		}, true
	case "write_doc", "write_go_file", "search_replace_file", "delete_file", "create_directory":
		return skillToolRequirement{
			AnyOf: []string{"sop.create-project", "sop.modify-project"},
			Hint:  "`read_skill(\"sop.create-project\")` 或 `read_skill(\"sop.modify-project\")`",
		}, true
	case "build_workspace":
		return skillToolRequirement{
			AnyOf: []string{"sop.create-project", "sop.modify-project", "sdk.build-validation"},
			Hint:  "`read_skill(\"sop.create-project\")`、`read_skill(\"sop.modify-project\")` 或 `read_skill(\"sdk.build-validation\")`",
		}, true
	case "publish_to_hub", "push_to_hub", "copy_directory":
		return skillToolRequirement{
			AnyOf: []string{"system.openapi", "system.openapi.hub", "sop.execute-function"},
			Hint:  "`read_skill(\"system.openapi.hub\")`，或先读 `system.openapi` / `sop.execute-function`",
		}, true
	default:
		return skillToolRequirement{}, false
	}
}

func skillRequirementSatisfied(req skillToolRequirement, loadedSkills map[string]*agentosskills.Skill) bool {
	if len(req.AnyOf) == 0 {
		return true
	}
	for _, skill := range loadedSkills {
		if skill == nil {
			continue
		}
		id := strings.TrimSpace(skill.Meta.ID)
		for _, allowed := range req.AnyOf {
			allowed = strings.TrimSpace(allowed)
			if id == allowed || strings.HasPrefix(id, allowed+".") {
				return true
			}
		}
	}
	return false
}

func skillRequirementGateMessage(toolName string, req skillToolRequirement) string {
	hint := strings.TrimSpace(req.Hint)
	if hint == "" {
		hint = "读取匹配的 skill"
	}
	return "已拦截工具调用：当前启用了 Skills，`" + toolName + "` 属于执行/写入类工具。本会话需要先按 Skills 目录读取匹配的 skill，再按 skill 的 required_docs、schema 和授权要求执行。下一步只调用 " + hint + "；读取成功前不要继续调用 `" + toolName + "` 或其他执行/写入类工具。不确定时再用 `search_skills`。普通只读搜索和问答不受此限制。"
}

func missingRequiredDocsGateMessage(toolName string, missingDocs []string) string {
	docs := strings.Join(missingDocs, ",")
	if docs == "" {
		docs = "<skill required_docs>"
	}
	return "已拦截工具调用：当前 skill 的 required_docs 尚未全部进入已读文档 map，不能继续调用 `" + strings.TrimSpace(toolName) + "`。下一步只调用 `read_doc` 读取这些文档：" + docs + "。正常情况下 `read_skill` 会自动注入 required_docs；如果你刚读过 skill，重新调用该 skill 即可修复历史状态。"
}

func readDocTargetsMissingRequiredDocs(args map[string]interface{}, missingDocs []string) bool {
	if len(missingDocs) == 0 {
		return false
	}
	targets := guideDocPathsFromReadDocArgs(args)
	if len(targets) == 0 {
		return false
	}
	for _, target := range targets {
		if !docPathMatchesAnyMissingRequiredDoc(target, missingDocs) {
			return false
		}
	}
	return true
}

func docPathMatchesAnyMissingRequiredDoc(target string, missingDocs []string) bool {
	target = normalizeGuideDocPath(target)
	if target == "" {
		return false
	}
	for _, missing := range missingDocs {
		missing = normalizeGuideDocPath(missing)
		if missing == "" {
			continue
		}
		if target == missing || strings.HasPrefix(missing, target+"/") || strings.HasPrefix(target, missing+"/") {
			return true
		}
	}
	return false
}

func missingRequiredDocsForSkills(loadedSkills map[string]*agentosskills.Skill, loadedGuideDocs map[string]struct{}) []string {
	if len(loadedSkills) == 0 {
		return nil
	}
	missing := make([]string, 0)
	seen := make(map[string]struct{})
	for _, skill := range loadedSkills {
		if skill == nil {
			continue
		}
		for _, doc := range skill.Meta.RequiredDocs {
			doc = normalizeGuideDocPath(doc)
			if doc == "" {
				continue
			}
			if _, ok := seen[doc]; ok {
				continue
			}
			seen[doc] = struct{}{}
			if !hasLoadedGuideDoc(loadedGuideDocs, doc) {
				missing = append(missing, doc)
			}
		}
	}
	sort.Strings(missing)
	return missing
}

func requiredDocPathsForSkill(skill *agentosskills.Skill) []string {
	if skill == nil || len(skill.Meta.RequiredDocs) == 0 {
		return nil
	}
	out := make([]string, 0, len(skill.Meta.RequiredDocs))
	seen := make(map[string]struct{}, len(skill.Meta.RequiredDocs))
	for _, doc := range skill.Meta.RequiredDocs {
		doc = normalizeGuideDocPath(doc)
		if doc == "" {
			continue
		}
		if _, ok := seen[doc]; ok {
			continue
		}
		seen[doc] = struct{}{}
		out = append(out, doc)
	}
	return out
}

func (s *WorkspaceChatService) loadedSkillsForSession(ctx context.Context, sessionID string) map[string]*agentosskills.Skill {
	loaded := make(map[string]*agentosskills.Skill)
	if s == nil || s.messageRepo == nil || strings.TrimSpace(sessionID) == "" {
		return loaded
	}
	messages, err := s.messageRepo.ListBySessionID(sessionID)
	if err != nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] 查询会话已读 skill 失败 SessionID=%s: %v", sessionID, err)
		return loaded
	}
	return loadedSkillsFromMessages(ctx, messages, agentosskills.DefaultRegistry())
}

func loadedSkillsFromMessages(ctx context.Context, messages []*model.AgentChatMessage, registry *agentosskills.Registry) map[string]*agentosskills.Skill {
	loaded := make(map[string]*agentosskills.Skill)
	if registry == nil {
		return loaded
	}
	readSkillCalls := make(map[string]string)
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		switch msg.Role {
		case RoleAssistant:
			if msg.ToolCalls == nil || strings.TrimSpace(*msg.ToolCalls) == "" {
				continue
			}
			var toolCalls []llms.ToolCall
			if err := json.Unmarshal([]byte(*msg.ToolCalls), &toolCalls); err != nil {
				logger.Warnf(ctx, "[WorkspaceChatStream] 解析历史 tool_calls 失败 MessageID=%d: %v", msg.ID, err)
				continue
			}
			for _, tc := range toolCalls {
				if tc.Function.Name != "read_skill" || strings.TrimSpace(tc.ID) == "" {
					continue
				}
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &args); err != nil {
					logger.Warnf(ctx, "[WorkspaceChatStream] 解析历史 read_skill 参数失败 ToolCallID=%s: %v", tc.ID, err)
					continue
				}
				if id := skillIDFromReadSkillArgs(args); id != "" {
					readSkillCalls[tc.ID] = id
				}
			}
		case RoleTool:
			if msg.ToolStatus != ToolCallStatusOK {
				continue
			}
			id := readSkillCalls[msg.ToolCallID]
			if id == "" {
				continue
			}
			skill, ok := registry.Get(id)
			if !ok || skill == nil {
				continue
			}
			loaded[skill.Meta.ID] = skill
		}
	}
	return loaded
}

func skillIDFromReadSkillArgs(args map[string]interface{}) string {
	if len(args) == 0 {
		return ""
	}
	id, _ := args["id"].(string)
	return strings.TrimSpace(id)
}

func markLoadedRequiredDocsForSkill(ctx context.Context, loaded map[string]struct{}, registry *agentosskills.Registry, id string) {
	if loaded == nil {
		return
	}
	if registry == nil {
		registry = agentosskills.DefaultRegistry()
	}
	skill, ok := registry.Get(id)
	if !ok || skill == nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] read_skill 成功但 registry 未找到 required_docs 来源 skill=%s", id)
		return
	}
	for _, docPath := range requiredDocPathsForSkill(skill) {
		loaded[docPath] = struct{}{}
	}
}

func updateLoadedSkillsAfterToolCall(ctx context.Context, loaded map[string]*agentosskills.Skill, toolName string, args map[string]interface{}, status string) {
	if status != ToolCallStatusOK || strings.TrimSpace(toolName) != "read_skill" || loaded == nil {
		return
	}
	id := skillIDFromReadSkillArgs(args)
	if id == "" {
		return
	}
	skill, ok := agentosskills.DefaultRegistry().Get(id)
	if !ok || skill == nil {
		logger.Warnf(ctx, "[WorkspaceChatStream] read_skill 成功但 registry 未找到 skill=%s", id)
		return
	}
	loaded[skill.Meta.ID] = skill
}
