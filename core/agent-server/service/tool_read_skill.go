package service

import (
	"context"
	"fmt"
	"strings"

	agentosskills "github.com/ai-agent-os/ai-agent-os/core/agent-server/skills"
	"github.com/ai-agent-os/ai-agent-os/dto"
)

type ReadSkillTool struct{}

type readSkillArgs struct {
	ID string `json:"id" schema_desc:"skill id、名称或路径前缀，例如 sop.create-project 或 创建项目 SOP" schema_required:"true"`
}

type readSkillResultData struct {
	RequiredDocs []readSkillRequiredDocData `json:"required_docs,omitempty" schema_desc:"后端自动读取并注入的 required_docs 文档正文；模型无需再次 read_doc 这些路径"`
	Skill        *agentosskills.Skill       `json:"skill,omitempty" schema_desc:"完整 skill 内容"`
	Errors       []string                   `json:"errors,omitempty" schema_desc:"skills 加载错误；出现时表示部分 skill 不可用"`
}

type readSkillRequiredDocData struct {
	Path    string `json:"path" schema_desc:"required_docs 文档路径"`
	Content string `json:"content,omitempty" schema_desc:"文档正文"`
	IsError bool   `json:"is_error,omitempty" schema_desc:"读取该文档是否失败"`
}

var readSkillToolDef = toolDefinitionWithOutput[readSkillArgs, structuredToolResultSchema[readSkillResultData]](
	"read_skill",
	"读取本地 AgentOS Skill 的完整 SKILL.md 内容，并自动读取该 skill 的 required_docs 后一并返回。通常根据系统提示中的 Skills 目录直接传 skill id；只有不确定时才先调用 search_skills 兜底查找。该工具只读、无副作用。",
)

func (t *ReadSkillTool) Definition() dto.ToolDef {
	return readSkillToolDef
}

func (t *ReadSkillTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readSkillArgs](call.Args)
	if err != nil {
		return toolResult("read_skill 参数解析失败: "+err.Error(), true)
	}
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return toolResult("read_skill 必填 id。请根据系统提示中的 Skills 目录直接传 skill id；不确定时再用 search_skills 搜索。", true)
	}

	registry := agentosskills.DefaultRegistry()
	skill, ok := registry.Get(id)
	if !ok {
		data := readSkillResultData{Errors: skillRegistryErrors(registry)}
		return toolResultWithData(fmt.Sprintf("未找到 skill: %s。请检查系统提示中的 Skills 目录；不确定时调用 search_skills 获取可用 skill id。", id), true, data)
	}
	requiredDocs, hasDocError := readRequiredDocsForSkill(ctx, skill, call.FullCodePath)
	data := readSkillResultData{
		Skill:        skill,
		RequiredDocs: requiredDocs,
		Errors:       skillRegistryErrors(registry),
	}
	return toolResultWithStructuredData(data, registry.LoadError() != nil || hasDocError)
}

func readRequiredDocsForSkill(ctx context.Context, skill *agentosskills.Skill, currentFullCodePath string) ([]readSkillRequiredDocData, bool) {
	paths := requiredDocPathsForSkill(skill)
	if len(paths) == 0 {
		return nil, false
	}
	out := make([]readSkillRequiredDocData, 0, len(paths))
	var hasError bool
	for _, path := range paths {
		content, isError := runReadDocTool(ctx, readDocArgs{Directory: path}, currentFullCodePath)
		if isError {
			hasError = true
		}
		out = append(out, readSkillRequiredDocData{
			Path:    path,
			Content: content,
			IsError: isError,
		})
	}
	return out, hasError
}
