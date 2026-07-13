package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"github.com/kageos/kageos/pkg/scheduledsdk"
	"github.com/kageos/kageos/pkg/subjects"
)

func (s *Service) PublishPendingOutbox(ctx context.Context, publisher OutboxPublisher, limit int) (int, error) {
	if publisher == nil {
		return 0, fmt.Errorf("timer-scheduler: outbox publisher is nil")
	}
	now := s.now()
	events, err := s.outboxRepo.ListReady(ctx, now, limit)
	if err != nil {
		return 0, err
	}
	published := 0
	for _, event := range events {
		if err := publisher.Publish(ctx, event.Subject, event.Payload); err != nil {
			attempts := event.Attempts + 1
			if attempts >= s.opts.MaxOutboxAttempts {
				if markErr := s.outboxRepo.MarkDeadLetter(ctx, event.ID, attempts, err.Error()); markErr != nil {
					return published, markErr
				}
				continue
			}
			if markErr := s.outboxRepo.MarkRetry(ctx, event.ID, attempts, now.Add(outboxBackoff(attempts)), err.Error()); markErr != nil {
				return published, markErr
			}
			continue
		}
		if err := s.outboxRepo.MarkPublished(ctx, event.ID, now); err != nil {
			return published, err
		}
		published++
	}
	return published, nil
}

func (s *Service) executionRequestedOutbox(task *model.TimerTask, exec *model.TimerExecution) *model.TimerOutboxEvent {
	metadata := decodeStringMap(task.MetadataJSON)
	if metadata == nil {
		metadata = map[string]string{}
	}
	metadata["task_title"] = strings.TrimSpace(task.Title)
	event := scheduledsdk.ExecutionRequestedEvent{
		EventID:         fmt.Sprintf("timer-execution-requested-%d-attempt-%d", exec.ID, exec.Attempt),
		TaskID:          task.ID,
		ExecutionID:     exec.ID,
		ExecutorKey:     task.ExecutorKey,
		ScheduledAt:     exec.ScheduledAt,
		TraceID:         exec.TraceID,
		Attempt:         exec.Attempt,
		SourceType:      task.SourceType,
		SourceRef:       task.SourceRef,
		ResourceScope:   task.ResourceScope,
		ResourceKey:     task.ResourceKey,
		RequestUser:     scheduledTaskRequestUser(task),
		RequestUserDept: task.RequestUserDept,
		Metadata:        metadata,
		ExecutorPayload: cloneRaw(task.ExecutorPayload),
	}
	payload, _ := json.Marshal(event)
	return &model.TimerOutboxEvent{
		EventID:     event.EventID,
		EventType:   eventTypeRequested,
		Subject:     subjects.TimerExecutionRequestedSubject(task.ExecutorKey),
		AggregateID: exec.ID,
		Payload:     payload,
		Status:      outboxStatusPending,
	}
}
