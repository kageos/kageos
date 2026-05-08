package model

import (
	"encoding/json"
	"time"
)

type TimerOutboxEvent struct {
	ID          int64           `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	EventID     string          `json:"event_id" gorm:"size:128;not null;uniqueIndex;comment:事件ID"`
	EventType   string          `json:"event_type" gorm:"size:128;not null;index;comment:事件类型"`
	AggregateID int64           `json:"aggregate_id" gorm:"not null;index;comment:聚合ID"`
	Payload     json.RawMessage `json:"payload" gorm:"type:json;not null;comment:事件JSON"`
	Status      string          `json:"status" gorm:"size:20;not null;index;default:pending;comment:pending/published/failed"`
	Attempts    int             `json:"attempts" gorm:"default:0;comment:投递次数"`
	LastError   string          `json:"last_error" gorm:"type:text;comment:最近投递错误"`
	PublishedAt *time.Time      `json:"published_at" gorm:"comment:发布时间"`
}

func (TimerOutboxEvent) TableName() string {
	return "timer_outbox_event"
}
