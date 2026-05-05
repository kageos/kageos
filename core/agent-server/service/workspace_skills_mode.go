package service

import (
	"fmt"
	"strings"

	agentosskills "github.com/ai-agent-os/ai-agent-os/core/agent-server/skills"
)

const (
	skillsModeOn = "on"
)

var workspaceSkillToolNames = []string{"read_skill", "search_skills"}

func currentWorkspaceSkillsMode() string {
	return skillsModeOn
}

func appendWorkspaceSkillToolNames(toolNames []string) []string {
	out := append([]string(nil), toolNames...)
	seen := make(map[string]struct{}, len(out)+len(workspaceSkillToolNames))
	for _, name := range out {
		seen[name] = struct{}{}
	}
	for _, name := range workspaceSkillToolNames {
		if _, ok := seen[name]; ok {
			continue
		}
		out = append(out, name)
		seen[name] = struct{}{}
	}
	return out
}

func workspaceSkillsPrompt(modeCode string) string {
	modeCode = strings.TrimSpace(modeCode)
	if modeCode == "" {
		modeCode = "dev"
	}
	lines := []string{
		"## Skills 工作规则",
		"",
		"本工作台只使用本地 Skills 作为 SOP/能力入口。旧 `/system/prompt/workspace/*` SOP 路径已下线；`read_skill` 会自动注入该 skill 的 required_docs 内容。",
		"",
		"- 接到任务后先做意图识别：创建项目、修改项目、解释/问答、执行函数、平台 OpenAPI、system 工具/杂项。",
		"- 除纯闲聊外，先看下面的 Skills 目录；能判断意图时直接 `read_skill(\"<skill id>\")`，不要先 search。",
		"- 只有不确定该读哪个 skill、或用户需求超出目录大纲时，才用 `search_skills` 兜底查找。",
		"- 读取 skill 后按其中的 SOP、已自动注入的 required_docs、allowed_tools、验收清单执行；不要重复读取刚刚自动注入过的 required_docs。",
		"- 普通信息搜索、临时问答、闲聊或找不到匹配 skill 的任务，不要为了凑流程强行搜索/读取 skill，可直接使用合适的只读工具。",
		"- 如果需要使用当前 skill 未声明的工具，不能直接继续；必须先读取更匹配的 skill，或把方案降级到当前 skill 的 allowed_tools 范围内。",
		"- 问答模式只做读取和解释，不进行写文件、创建目录、编译、运行函数等有副作用操作；执行模式允许按 SOP 调用写操作工具。",
		"",
		"当前 mode: " + modeCode,
		"",
		workspaceSkillCatalogForMode(modeCode),
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func workspaceSkillCatalogForMode(modeCode string) string {
	modeCode = strings.TrimSpace(modeCode)
	results := agentosskills.DefaultRegistry().Search(agentosskills.SearchOptions{
		Mode:  modeCode,
		Limit: 50,
	})
	if len(results) == 0 {
		return "### Skills 目录\n\n当前 mode 未匹配到专属 skill；请用 `search_skills` 兜底搜索。"
	}
	var b strings.Builder
	b.WriteString("### Skills 目录\n\n")
	b.WriteString("优先直接按下列 ID 调用 `read_skill`：\n\n")
	groups := []struct {
		title  string
		prefix string
	}{
		{title: "SOP 场景", prefix: "sop."},
		{title: "SDK 写法", prefix: "sdk."},
		{title: "System OpenAPI 工作空间", prefix: "system.openapi"},
		{title: "System 工具工作空间", prefix: "system.tools"},
	}
	written := make(map[string]struct{}, len(results))
	for _, group := range groups {
		if writeSkillCatalogGroup(&b, results, written, group.title, group.prefix) {
			b.WriteString("\n")
		}
	}
	if writeSkillCatalogGroup(&b, results, written, "其他", "") {
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

func writeSkillCatalogGroup(b *strings.Builder, results []agentosskills.SearchResult, written map[string]struct{}, title string, prefix string) bool {
	var items []agentosskills.SearchResult
	for _, result := range results {
		id := result.Meta.ID
		if _, ok := written[id]; ok {
			continue
		}
		if prefix != "" && !strings.HasPrefix(id, prefix) {
			continue
		}
		if prefix == "" && (strings.HasPrefix(id, "sop.") || strings.HasPrefix(id, "sdk.") || strings.HasPrefix(id, "system.")) {
			continue
		}
		items = append(items, result)
		written[id] = struct{}{}
	}
	if len(items) == 0 {
		return false
	}
	b.WriteString("#### " + title + "\n\n")
	for _, result := range items {
		meta := result.Meta
		b.WriteString(fmt.Sprintf("- `%s`：%s", meta.ID, meta.Description))
		if len(meta.Triggers) > 0 {
			triggers := meta.Triggers
			if len(triggers) > 5 {
				triggers = triggers[:5]
			}
			b.WriteString("；触发词：" + strings.Join(triggers, "、"))
		}
		b.WriteString("\n")
	}
	return true
}
