package model

import (
	"encoding/json"
	"time"
)

type TimerOutboxEvent struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	EventID       string          `json:"event_id" gorm:"size:160;not null;uniqueIndex"`
	EventType     string          `json:"event_type" gorm:"size:128;not null;index"`
	Subject       string          `json:"subject" gorm:"size:255;not null;index"`
	AggregateID   int64           `json:"aggregate_id" gorm:"not null;index"`
	Payload       json.RawMessage `json:"payload" gorm:"type:json;not null"`
	Status        string          `json:"status" gorm:"size:20;not null;index;default:pending"`
	Attempts      int             `json:"attempts" gorm:"default:0"`
	NextAttemptAt *time.Time      `json:"next_attempt_at" gorm:"index"`
	LastError     string          `json:"last_error" gorm:"type:text"`
	PublishedAt   *time.Time      `json:"published_at"`
}

func (TimerOutboxEvent) TableName() string {
	return "timer_outbox_event"
}
