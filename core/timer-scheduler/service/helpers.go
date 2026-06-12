package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/robfig/cron/v3"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

func validateCreateTaskRequest(req scheduledsdk.CreateTaskRequest, payloadLimit int) error {
	if strings.TrimSpace(req.ExecutorKey) == "" {
		return fmt.Errorf("%w: executor_key is required", scheduledsdk.ErrInvalidRequest)
	}
	if err := req.Schedule.Validate(); err != nil {
		return err
	}
	return validatePayload(req.ExecutorPayload, payloadLimit)
}

func validatePayload(payload []byte, limit int) error {
	if len(payload) == 0 {
		return nil
	}
	if limit > 0 && len(payload) > limit {
		return fmt.Errorf("%w: executor_payload exceeds %d bytes", scheduledsdk.ErrInvalidRequest, limit)
	}
	if !json.Valid(payload) {
		return fmt.Errorf("%w: executor_payload must be valid json", scheduledsdk.ErrInvalidRequest)
	}
	return nil
}

func nextRunForTask(task *model.TimerTask, now time.Time) (*time.Time, error) {
	schedule := scheduledsdk.Schedule{
		Type:            scheduledsdk.ScheduleType(task.ScheduleType),
		CronExpr:        task.CronExpr,
		IntervalSeconds: task.IntervalSeconds,
		Timezone:        task.Timezone,
		MaxRuns:         task.MaxRuns,
	}
	if task.RunAt != nil {
		schedule.RunAt = *task.RunAt
	}
	return nextRunForSchedule(schedule, now)
}

func nextRunForSchedule(schedule scheduledsdk.Schedule, now time.Time) (*time.Time, error) {
	if err := schedule.Validate(); err != nil {
		return nil, err
	}
	loc, err := locationForSchedule(schedule.Timezone)
	if err != nil {
		return nil, err
	}
	switch schedule.Type {
	case scheduledsdk.ScheduleAt:
		runAt := schedule.RunAt
		runAt = runAt.UTC()
		return &runAt, nil
	case scheduledsdk.ScheduleEvery:
		next := now.Add(time.Duration(schedule.IntervalSeconds) * time.Second).UTC()
		return &next, nil
	case scheduledsdk.ScheduleCron:
		parsed, err := cronParser.Parse(strings.TrimSpace(schedule.CronExpr))
		if err != nil {
			return nil, err
		}
		next := parsed.Next(now.In(loc)).UTC()
		return &next, nil
	default:
		return nil, fmt.Errorf("%w: unsupported schedule type %q", scheduledsdk.ErrInvalidRequest, schedule.Type)
	}
}

func locationForSchedule(timezone string) (*time.Location, error) {
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid timezone %q", scheduledsdk.ErrInvalidRequest, timezone)
	}
	return loc, nil
}

func computeTaskNextState(task *model.TimerTask, success bool, now time.Time) (string, *time.Time, string) {
	lastErr := task.LastErrorMessage
	if task.MaxRuns > 0 && task.RunCount >= task.MaxRuns {
		return string(scheduledsdk.TaskStatusDone), nil, lastErr
	}
	switch scheduledsdk.ScheduleType(task.ScheduleType) {
	case scheduledsdk.ScheduleAt:
		if success {
			return string(scheduledsdk.TaskStatusDone), nil, ""
		}
		return string(scheduledsdk.TaskStatusFailed), nil, lastErr
	case scheduledsdk.ScheduleCron, scheduledsdk.ScheduleEvery:
		nextRunAt, err := nextRunForTask(task, now)
		if err != nil {
			return string(scheduledsdk.TaskStatusFailed), nil, err.Error()
		}
		return string(scheduledsdk.TaskStatusPending), nextRunAt, lastErr
	default:
		return string(scheduledsdk.TaskStatusFailed), nil, "unsupported schedule type"
	}
}

func scheduledAtForTask(task *model.TimerTask, fallback time.Time) time.Time {
	if task.NextRunAt != nil {
		return *task.NextRunAt
	}
	return fallback
}

func cloneRaw(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}

func mustJSON(v interface{}) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil || len(data) == 0 || string(data) == "null" {
		return nil
	}
	return data
}

func decodeStringList(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var list []string
	if err := json.Unmarshal(raw, &list); err != nil {
		return nil
	}
	return list
}

func decodeStringMap(raw json.RawMessage) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultListPageSize
	}
	if pageSize > maxListPageSize {
		pageSize = maxListPageSize
	}
	return page, pageSize
}

func outboxBackoff(attempts int) time.Duration {
	switch {
	case attempts <= 1:
		return time.Second
	case attempts == 2:
		return 5 * time.Second
	case attempts == 3:
		return 30 * time.Second
	default:
		return 5 * time.Minute
	}
}
