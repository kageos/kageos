package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/serviceconfig"
)

func scheduledTaskClient() *scheduledsdk.Client {
	return scheduledsdk.NewClient(scheduledsdk.Options{
		BaseURL: serviceconfig.BuildInternalGatewayURL("/timer/api/v1"),
	})
}

func buildScheduledTaskSchedule(args scheduledTaskScheduleArgs) (scheduledsdk.Schedule, error) {
	args = normalizeScheduledTaskScheduleArgs(args)
	scheduleType := scheduledsdk.ScheduleType(strings.TrimSpace(args.ScheduleType))
	schedule := scheduledsdk.Schedule{
		Type:    scheduleType,
		MaxRuns: args.MaxRuns,
	}
	switch scheduleType {
	case scheduledsdk.ScheduleAt:
		runAt, err := parseScheduledRunAt(args.RunAt, args.Timezone)
		if err != nil {
			return schedule, err
		}
		schedule.RunAt = runAt
	case scheduledsdk.ScheduleCron:
		schedule.CronExpr = strings.TrimSpace(args.CronExpr)
		schedule.Timezone = scheduledTaskTimezone(args.Timezone)
	case scheduledsdk.ScheduleEvery:
		schedule.IntervalSeconds = args.IntervalSeconds
	default:
		return schedule, fmt.Errorf("schedule_type is required unless exactly one of run_at, cron_expr or interval_seconds is provided")
	}
	return schedule, schedule.Validate()
}

func normalizeCreateScheduledFunctionTaskArgs(args createScheduledFunctionTaskArgs) createScheduledFunctionTaskArgs {
	if strings.TrimSpace(args.FullCodePath) == "" {
		args.FullCodePath = strings.TrimSpace(args.FunctionPath)
	}
	if strings.TrimSpace(args.CronExpr) == "" {
		args.CronExpr = strings.TrimSpace(args.Cron)
	}
	if strings.TrimSpace(args.Title) == "" {
		args.Title = strings.TrimSpace(args.TaskName)
	}
	if args.MaxRuns == 0 {
		args.MaxRuns = parseScheduledCompatInt(args.MaxExecutions)
	}
	if strings.TrimSpace(args.Body) == "" {
		args.Body = scheduledBodyFromCompatValue(args.InvokeParams)
	}
	if strings.TrimSpace(args.Body) == "" {
		args.Body = scheduledBodyFromCompatValue(args.Payload)
	}
	args.ScheduleType = normalizeScheduledTaskScheduleArgs(args.scheduleArgs()).ScheduleType
	return args
}

func normalizeCreateScheduledAgentTaskArgs(args createScheduledAgentTaskArgs) createScheduledAgentTaskArgs {
	if strings.TrimSpace(args.FullCodePath) == "" {
		args.FullCodePath = strings.TrimSpace(args.Directory)
	}
	args.ScheduleType = normalizeScheduledTaskScheduleArgs(args.scheduleArgs()).ScheduleType
	return args
}

func rejectCreateScheduledAgentTaskUnknownArgs(args map[string]interface{}) error {
	allowed := map[string]struct{}{
		"full_code_path":       {},
		"directory":            {},
		"title":                {},
		"message":              {},
		"schedule_type":        {},
		"run_at":               {},
		"cron_expr":            {},
		"interval_seconds":     {},
		"timezone":             {},
		"max_runs":             {},
		"mode_code":            {},
		"files":                {},
		"llm_config_id":        {},
		"max_duration_seconds": {},
		"description":          {},
		"idempotency_key":      {},
	}
	for key := range args {
		if _, ok := allowed[key]; !ok {
			return fmt.Errorf("不支持参数 %q；Agent 任务只接收顶层 title、message、full_code_path（directory 仅兼容旧别名）和计划配置，间隔执行请用 interval_seconds", key)
		}
	}
	return nil
}

func normalizeScheduledTaskScheduleArgs(args scheduledTaskScheduleArgs) scheduledTaskScheduleArgs {
	if strings.TrimSpace(args.ScheduleType) != "" {
		return args
	}
	hasRunAt := strings.TrimSpace(args.RunAt) != ""
	hasCron := strings.TrimSpace(args.CronExpr) != ""
	hasEvery := args.IntervalSeconds > 0
	switch {
	case hasRunAt && !hasCron && !hasEvery:
		args.ScheduleType = string(scheduledsdk.ScheduleAt)
	case hasCron && !hasRunAt && !hasEvery:
		args.ScheduleType = string(scheduledsdk.ScheduleCron)
	case hasEvery && !hasRunAt && !hasCron:
		args.ScheduleType = string(scheduledsdk.ScheduleEvery)
	}
	return args
}

func parseScheduledRunAt(raw string, timezone string) (time.Time, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return time.Time{}, fmt.Errorf("run_at is required")
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t, nil
	}
	location := time.Local
	if tz := scheduledTaskTimezone(timezone); tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			location = loc
		}
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", "2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, value, location); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("run_at must be RFC3339 or YYYY-MM-DD HH:mm:ss")
}

func scheduledTaskTimezone(raw string) string {
	tz := strings.TrimSpace(raw)
	if tz == "" {
		return ""
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return ""
	}
	return tz
}
