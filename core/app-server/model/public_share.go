package model

import (
	"encoding/json"
	"time"

	"github.com/kageos/kageos/pkg/gormx/models"
)

const (
	PublicShareResourceTypeForm = "form"
	PublicShareActionFormSubmit = "form.submit"
)

type PublicShare struct {
	models.Base
	ShareID      string          `json:"share_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	TenantUser   string          `json:"tenant_user" gorm:"type:varchar(255);not null;index:idx_public_share_owner"`
	App          string          `json:"app" gorm:"type:varchar(255);not null;index:idx_public_share_owner"`
	FullCodePath string          `json:"full_code_path" gorm:"type:varchar(1024);not null"`
	ResourceType string          `json:"resource_type" gorm:"type:varchar(64);not null"`
	Action       string          `json:"action" gorm:"type:varchar(64);not null"`
	Title        string          `json:"title" gorm:"type:varchar(255)"`
	Description  string          `json:"description" gorm:"type:text"`
	Enabled      bool            `json:"enabled" gorm:"not null;default:true;index"`
	ExpiresAt    *time.Time      `json:"expires_at" gorm:"index"`
	MaxUses      int             `json:"max_uses" gorm:"not null;default:0"`
	UseCount     int             `json:"use_count" gorm:"not null;default:0"`
	LastUsedAt   *time.Time      `json:"last_used_at"`
	PresetValues json.RawMessage `json:"preset_values" gorm:"type:json"`
}

func (PublicShare) TableName() string {
	return "public_share"
}

type PublicShareEvent struct {
	models.Base
	ShareID       string `json:"share_id" gorm:"type:varchar(64);not null;index"`
	TenantUser    string `json:"tenant_user" gorm:"type:varchar(255);not null;index"`
	App           string `json:"app" gorm:"type:varchar(255);not null;index"`
	FullCodePath  string `json:"full_code_path" gorm:"type:varchar(1024);not null"`
	AnonActorID   string `json:"anon_actor_id" gorm:"type:varchar(255);index"`
	Action        string `json:"action" gorm:"type:varchar(64);not null"`
	Status        string `json:"status" gorm:"type:varchar(32);not null"`
	TraceID       string `json:"trace_id" gorm:"type:varchar(255);index"`
	ErrorMessage  string `json:"error_message" gorm:"type:text"`
	IPAddressHash string `json:"ip_address_hash" gorm:"type:varchar(64)"`
	UserAgentHash string `json:"user_agent_hash" gorm:"type:varchar(64)"`
}

func (PublicShareEvent) TableName() string {
	return "public_share_event"
}
