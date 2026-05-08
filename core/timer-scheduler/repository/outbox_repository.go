package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/timer-scheduler/model"
	"gorm.io/gorm"
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

func (r *TimerOutboxRepository) Create(event *model.TimerOutboxEvent) error {
	return r.db.Create(event).Error
}

func (r *TimerOutboxRepository) ListPending(limit int) ([]*model.TimerOutboxEvent, error) {
	query := r.db.Where("status = ?", "pending").Order("created_at ASC, id ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	var list []*model.TimerOutboxEvent
	if err := query.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *TimerOutboxRepository) MarkPublished(id int64, publishedAt time.Time) error {
	return r.db.Model(&model.TimerOutboxEvent{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{
			"status":       "published",
			"published_at": publishedAt,
			"last_error":   "",
		}).Error
}

func (r *TimerOutboxRepository) MarkFailed(id int64, errMessage string) error {
	return r.db.Model(&model.TimerOutboxEvent{}).
		Where("id = ? AND status = ?", id, "pending").
		Updates(map[string]interface{}{
			"attempts":   gorm.Expr("attempts + ?", 1),
			"last_error": errMessage,
		}).Error
}
