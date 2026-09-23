package service

import (
	"context"
	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"time"
)

type ExecutionSummary struct {
	ID             int64      `json:"id"`
	TaskID         int64      `json:"task_id"`
	Status         string     `json:"status"`
	ScheduledAt    time.Time  `json:"scheduled_at"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	HeartbeatAt    *time.Time `json:"heartbeat_at"`
	LeaseUntil     *time.Time `json:"lease_until"`
	DurationMillis int64      `json:"duration_millis"`
}

// Called only for the already-authorized task list; does not load result payloads.
func (s *Service) TaskExecutionSummaries(ctx context.Context, tasks []*scheduledsdk.Task) (map[int64]ExecutionSummary, error) {
	out := map[int64]ExecutionSummary{}
	ids := []int64{}
	for _, task := range tasks {
		if task.LastExecutionID > 0 {
			ids = append(ids, task.LastExecutionID)
		}
	}
	if len(ids) == 0 {
		return out, nil
	}
	var rows []ExecutionSummary
	if err := s.db.WithContext(ctx).Model(&model.TimerExecution{}).Select("id, task_id, status, scheduled_at, started_at, finished_at, heartbeat_at, lease_until, duration_millis").Where("id IN ?", ids).Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.TaskID] = row
	}
	return out, nil
}
