package repository

import (
	"context"
	"time"

	"github.com/kageos/kageos/core/timer-scheduler/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TimerOutboxRepository struct {
	db *gorm.DB
}

func NewTimerOutboxRepository(db *gorm.DB) *TimerOutboxRepository {
	return &TimerOutboxRepository{db: db}
}

func (r *TimerOutboxRepository) WithDB(db *gorm.DB) *TimerOutboxRepository {
	return &TimerOutboxRepository{db: db}
}

func (r *TimerOutboxRepository) Create(ctx context.Context, event *model.TimerOutboxEvent) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(event).Error
}

func (r *TimerOutboxRepository) ListReady(ctx context.Context, now time.Time, limit int) ([]*model.TimerOutboxEvent, error) {
	query := r.db.WithContext(ctx).
		Where("status IN ? AND (next_attempt_at IS NULL OR next_attempt_at <= ?)", []string{"pending", "retry"}, now).
		Order("created_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var list []*model.TimerOutboxEvent
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *TimerOutboxRepository) MarkPublished(ctx context.Context, id int64, publishedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&model.TimerOutboxEvent{}).
		Where("id = ? AND status IN ?", id, []string{"pending", "retry"}).
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": publishedAt,
			"last_error":   "",
		}).Error
}

func (r *TimerOutboxRepository) MarkRetry(ctx context.Context, id int64, attempts int, nextAttemptAt time.Time, errMessage string) error {
	return r.db.WithContext(ctx).Model(&model.TimerOutboxEvent{}).
		Where("id = ? AND status IN ?", id, []string{"pending", "retry"}).
		Updates(map[string]interface{}{
			"status":          "retry",
			"attempts":        attempts,
			"next_attempt_at": nextAttemptAt,
			"last_error":      errMessage,
		}).Error
}

func (r *TimerOutboxRepository) MarkDeadLetter(ctx context.Context, id int64, attempts int, errMessage string) error {
	return r.db.WithContext(ctx).Model(&model.TimerOutboxEvent{}).
		Where("id = ? AND status IN ?", id, []string{"pending", "retry"}).
		Updates(map[string]interface{}{
			"status":     "dead_letter",
			"attempts":   attempts,
			"last_error": errMessage,
		}).Error
}

func (r *TimerOutboxRepository) DeadLetterExecutionRequestsForTask(ctx context.Context, taskID int64, errMessage string) error {
	executionIDs := r.db.WithContext(ctx).Model(&model.TimerExecution{}).Select("id").Where("task_id = ?", taskID)
	return r.db.WithContext(ctx).Model(&model.TimerOutboxEvent{}).
		Where("event_type = ? AND status IN ? AND aggregate_id IN (?)", "timer.execution.requested", []string{"pending", "retry"}, executionIDs).
		Updates(map[string]interface{}{
			"status":          "dead_letter",
			"next_attempt_at": nil,
			"last_error":      errMessage,
		}).Error
}
