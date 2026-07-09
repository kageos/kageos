package dto

import (
	"encoding/json"
	"time"

	"github.com/kageos/kageos/pkg/functionschema"
)

type CreatePublicShareReq struct {
	FullCodePath string          `json:"full_code_path" binding:"required"`
	Title        string          `json:"title,omitempty"`
	Description  string          `json:"description,omitempty"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	MaxUses      int             `json:"max_uses,omitempty"`
	PresetValues json.RawMessage `json:"preset_values,omitempty" swaggertype:"object"`
}

type PublicShareResp struct {
	ShareID      string          `json:"share_id"`
	TenantUser   string          `json:"tenant_user"`
	App          string          `json:"app"`
	FullCodePath string          `json:"full_code_path"`
	ResourceType string          `json:"resource_type"`
	Action       string          `json:"action"`
	Title        string          `json:"title"`
	Description  string          `json:"description"`
	Enabled      bool            `json:"enabled"`
	ExpiresAt    *time.Time      `json:"expires_at,omitempty"`
	MaxUses      int             `json:"max_uses"`
	UseCount     int             `json:"use_count"`
	LastUsedAt   *time.Time      `json:"last_used_at,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	CreatedBy    string          `json:"created_by"`
	PublicURL    string          `json:"public_url,omitempty"`
	PresetValues json.RawMessage `json:"preset_values,omitempty" swaggertype:"object"`
}

type PublicShareListResp struct {
	Items []*PublicShareResp `json:"items"`
}

type PublicShareViewResp struct {
	ShareID       string                         `json:"share_id"`
	Title         string                         `json:"title"`
	Description   string                         `json:"description"`
	FullCodePath  string                         `json:"full_code_path"`
	Schema        *functionschema.FunctionSchema `json:"schema"`
	ExpiresAt     *time.Time                     `json:"expires_at,omitempty"`
	RemainingUses *int                           `json:"remaining_uses,omitempty"`
	PresetValues  json.RawMessage                `json:"preset_values,omitempty" swaggertype:"object"`
}

type PublicAnonymousTokenResp struct {
	AnonymousToken string    `json:"anonymous_token"`
	ExpiresAt      time.Time `json:"expires_at"`
}
