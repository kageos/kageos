package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
)

type SummarizeTaskStateTool struct{}

type summarizeTaskStateArgs struct {
	Intent       string   `json:"intent" schema_desc:"当前身份 ID，例如 app.plan/app.create/app.modify/app.operate_test" schema_required:"true"`
	Directory    string   `json:"directory" schema_desc:"当前工作目录"`
	Outcome      string   `json:"outcome" schema_desc:"当前阶段结果，必须简短" schema_required:"true"`
	ChangedFiles []string `json:"changed_files" schema_desc:"本阶段改动文件路径"`
	Verified     []string `json:"verified" schema_desc:"已验证事项或命令"`
	Blockers     []string `json:"blockers" schema_desc:"未解决问题"`
	NextIntent   string   `json:"next_intent" schema_desc:"建议下一身份 ID"`
	NextGoal     string   `json:"next_goal" schema_desc:"下一阶段目标"`
}

type taskStateSummaryData struct {
	Intent       string   `json:"intent" schema_desc:"当前身份 ID" schema_required:"true"`
	Directory    string   `json:"directory,omitempty" schema_desc:"工作目录"`
	Summary      string   `json:"summary" schema_desc:"供 change_role 携带的极简摘要" schema_required:"true"`
	ChangedFiles []string `json:"changed_files,omitempty" schema_desc:"改动文件"`
	Verified     []string `json:"verified,omitempty" schema_desc:"已验证事项"`
	Blockers     []string `json:"blockers,omitempty" schema_desc:"未解决问题"`
	NextIntent   string   `json:"next_intent,omitempty" schema_desc:"建议下一身份"`
	NextGoal     string   `json:"next_goal,omitempty" schema_desc:"下一阶段目标"`
}

var summarizeTaskStateToolDef = toolDefinitionWithOutput[summarizeTaskStateArgs, structuredToolResultSchema[taskStateSummaryData]](
	"summarize_task_state",
	"把当前任务阶段压缩成可传给 change_role.task_summary 的极简状态摘要。只读、无副作用；用于避免旧 PRD、旧错误和长上下文污染下一身份。",
)

func (t *SummarizeTaskStateTool) Definition() dto.ToolDef {
	return summarizeTaskStateToolDef
}

func (t *SummarizeTaskStateTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	_ = ctx
	args, err := decodeToolArgs[summarizeTaskStateArgs](call.Args)
	if err != nil {
		return toolResult("summarize_task_state 参数解析失败: "+err.Error(), true)
	}
	data := buildTaskStateSummary(args)
	return toolResultWithStructuredData(data, false)
}

func buildTaskStateSummary(args summarizeTaskStateArgs) taskStateSummaryData {
	intent := strings.TrimSpace(args.Intent)
	if intent == "" {
		intent = "app.explain_review"
	}
	parts := []string{
		"身份=" + intent,
		"结果=" + compactText(args.Outcome, 160),
	}
	if len(args.ChangedFiles) > 0 {
		parts = append(parts, fmt.Sprintf("改动=%d 个文件", len(args.ChangedFiles)))
	}
	if len(args.Verified) > 0 {
		parts = append(parts, "验证="+compactText(strings.Join(args.Verified, "；"), 160))
	}
	if len(args.Blockers) > 0 {
		parts = append(parts, "阻塞="+compactText(strings.Join(args.Blockers, "；"), 160))
	}
	if strings.TrimSpace(args.NextIntent) != "" {
		parts = append(parts, "下一身份="+strings.TrimSpace(args.NextIntent))
	}
	if strings.TrimSpace(args.NextGoal) != "" {
		parts = append(parts, "下一目标="+compactText(args.NextGoal, 160))
	}
	return taskStateSummaryData{
		Intent:       intent,
		Directory:    strings.TrimSpace(args.Directory),
		Summary:      strings.Join(parts, " | "),
		ChangedFiles: trimStringSlice(args.ChangedFiles),
		Verified:     trimStringSlice(args.Verified),
		Blockers:     trimStringSlice(args.Blockers),
		NextIntent:   strings.TrimSpace(args.NextIntent),
		NextGoal:     strings.TrimSpace(args.NextGoal),
	}
}

func trimStringSlice(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func compactText(s string, maxRunes int) string {
	s = strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
	if maxRunes <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxRunes {
		return s
	}
	return string(runes[:maxRunes]) + "..."
}
