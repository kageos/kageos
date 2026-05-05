package service

import (
	"context"

	agentosskills "github.com/ai-agent-os/ai-agent-os/core/agent-server/skills"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type SearchSkillsTool struct{}

type searchSkillsArgs struct {
	Keyword string `json:"keyword" schema_desc:"搜索关键词。建议传用户意图、任务类型、关键对象，例如 创建项目、修改字段、执行表单、解释代码"`
	Mode    string `json:"mode" schema_desc:"当前工作台模式，用于过滤 skill，例如 dev、modify、execute、agent；不传则不过滤"`
	Limit   *int   `json:"limit" schema_desc:"最多返回条数，默认 10，最大 50"`
}

type searchSkillsResultData struct {
	Keyword string                       `json:"keyword" schema_desc:"搜索关键词" schema_required:"true"`
	Mode    string                       `json:"mode,omitempty" schema_desc:"过滤的工作台模式"`
	Count   int                          `json:"count" schema_desc:"返回 skill 数量" schema_required:"true"`
	Skills  []agentosskills.SearchResult `json:"skills" schema_desc:"匹配的 skills，按相关度排序" schema_required:"true"`
	Errors  []string                     `json:"errors,omitempty" schema_desc:"skills 加载错误；出现时表示部分 skill 不可用"`
}

var searchSkillsToolDef = toolDefinitionWithOutput[searchSkillsArgs, structuredToolResultSchema[searchSkillsResultData]](
	"search_skills",
	"兜底搜索本地 AgentOS Skills。系统提示已提供 Skills 目录，能判断意图时应直接用 read_skill；只有不确定该读哪个 skill、或用户需求超出目录大纲时才调用本工具。该工具只读、无副作用。",
)

func (t *SearchSkillsTool) Definition() dto.ToolDef {
	return searchSkillsToolDef
}

func (t *SearchSkillsTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	args, err := decodeToolArgs[searchSkillsArgs](call.Args)
	if err != nil {
		return toolResult("search_skills 参数解析失败: "+err.Error(), true)
	}
	registry := agentosskills.DefaultRegistry()
	limit := 0
	if args.Limit != nil {
		limit = *args.Limit
	}
	results := registry.Search(agentosskills.SearchOptions{
		Keyword: args.Keyword,
		Mode:    args.Mode,
		Limit:   limit,
	})
	data := searchSkillsResultData{
		Keyword: args.Keyword,
		Mode:    args.Mode,
		Count:   len(results),
		Skills:  results,
		Errors:  skillRegistryErrors(registry),
	}
	return toolResultWithStructuredData(data, registry.LoadError() != nil)
}

func skillRegistryErrors(registry *agentosskills.Registry) []string {
	if registry == nil {
		return nil
	}
	errs := registry.LoadErrors()
	if len(errs) == 0 {
		return nil
	}
	out := make([]string, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			out = append(out, err.Error())
		}
	}
	return out
}
