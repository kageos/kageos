package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/scheduledsdk"
)

const workspaceScheduledTaskContextTimeout = 2 * time.Second

func buildWorkspaceScheduledTasksSection(ctx context.Context, fullCodePath string) string {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" {
		return ""
	}
	req := listScheduledTasksRequest("all", fullCodePath, "", 0, 100)
	queryCtx, cancel := context.WithTimeout(withAgentToolClientSource(ctx), workspaceScheduledTaskContextTimeout)
	defer cancel()
	resp, err := listScheduledTasksAllPages(queryCtx, scheduledTaskClient(), req)
	if err != nil {
		return "### 当前目录自动执行摘要\n- 定时任务摘要加载失败；需要确认时调用 `list_scheduled_tasks` 重新查询。"
	}
	if resp == nil || len(resp.List) == 0 {
		return "### 当前目录自动执行摘要\n- 当前目录没有已配置的函数任务或 Agent 任务。"
	}

	var b strings.Builder
	b.WriteString("### 当前目录自动执行摘要\n")
	b.WriteString("以下只注入函数任务 / Agent 任务的稳定轻量元信息；未注入 Agent 任务 message、display_content、executor_payload 或每轮变化的执行状态正文。\n")
	sort.SliceStable(resp.List, func(i, j int) bool {
		left, right := resp.List[i], resp.List[j]
		if left == nil || right == nil {
			return right != nil
		}
		if left.ResourceKey != right.ResourceKey {
			return left.ResourceKey < right.ResourceKey
		}
		if left.Title != right.Title {
			return left.Title < right.Title
		}
		return left.ID < right.ID
	})
	for _, task := range resp.List {
		if task == nil {
			continue
		}
		b.WriteString(formatWorkspaceScheduledTaskSummary(task))
	}
	return strings.TrimSpace(b.String())
}

func formatWorkspaceScheduledTaskSummary(task *scheduledsdk.Task) string {
	if task == nil {
		return ""
	}
	var parts []string
	parts = append(parts, fmt.Sprintf("id=%d", task.ID))
	parts = append(parts, "类型="+workspaceScheduledTaskKindLabel(task))
	if title := strings.TrimSpace(task.Title); title != "" {
		parts = append(parts, "标题="+title)
	}
	if status := strings.TrimSpace(string(task.Status)); status != "" {
		parts = append(parts, "状态="+status)
	}
	if resource := strings.TrimSpace(task.ResourceKey); resource != "" {
		parts = append(parts, "资源="+resource)
	}
	if schedule := workspaceScheduledTaskScheduleLabel(task.Schedule); schedule != "" {
		parts = append(parts, "计划="+schedule)
	}
	if createdBy := strings.TrimSpace(task.CreatedBy); createdBy != "" {
		parts = append(parts, "创建人="+createdBy)
	}
	if requestUser := strings.TrimSpace(task.RequestUser); requestUser != "" && requestUser != strings.TrimSpace(task.CreatedBy) {
		parts = append(parts, "请求用户="+requestUser)
	}
	if description := strings.TrimSpace(task.Description); description != "" {
		parts = append(parts, "描述="+oneLineScheduledTaskText(description))
	}
	return "- " + strings.Join(parts, "；") + "\n"
}

func workspaceScheduledTaskKindLabel(task *scheduledsdk.Task) string {
	if task == nil {
		return "定时任务"
	}
	if kind := strings.TrimSpace(task.Metadata["kind"]); kind != "" {
		switch kind {
		case "scheduled_function":
			return "函数任务"
		case "scheduled_agent_session":
			return "数字员工"
		default:
			return kind
		}
	}
	switch strings.TrimSpace(task.ExecutorKey) {
	case "app.function":
		return "函数任务"
	case "agent.session":
		return "数字员工"
	default:
		return "定时任务"
	}
}

func workspaceScheduledTaskScheduleLabel(schedule scheduledsdk.Schedule) string {
	switch schedule.Type {
	case scheduledsdk.ScheduleAt:
		if schedule.RunAt.IsZero() {
			return "atime"
		}
		return "atime " + schedule.RunAt.Format(time.RFC3339)
	case scheduledsdk.ScheduleCron:
		out := "cron " + strings.TrimSpace(schedule.CronExpr)
		if tz := strings.TrimSpace(schedule.Timezone); tz != "" {
			out += " " + tz
		}
		return strings.TrimSpace(out)
	case scheduledsdk.ScheduleEvery:
		if schedule.IntervalSeconds <= 0 {
			return "every"
		}
		return fmt.Sprintf("every %ds", schedule.IntervalSeconds)
	default:
		return strings.TrimSpace(string(schedule.Type))
	}
}

func oneLineScheduledTaskText(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	return strings.Join(fields, " ")
}
