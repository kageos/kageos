package model

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// PubKey 发布密钥（用于跨站发布到本 Hub）
type PubKey struct {
	ID         int64       `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt  models.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	Username   string      `json:"username" gorm:"column:username;type:varchar(100);index;not null"`
	Name       string      `json:"name" gorm:"column:name;type:varchar(100)"`
	Key        string      `json:"-" gorm:"column:key;type:varchar(64);uniqueIndex;not null"`
	KeyPrefix  string      `json:"key_prefix" gorm:"column:key_prefix;type:varchar(12)"`
	LastUsedAt *time.Time  `json:"last_used_at" gorm:"column:last_used_at"`
}

func (PubKey) TableName() string {
	return "pub_key"
}
