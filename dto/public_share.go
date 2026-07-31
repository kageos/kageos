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

// PublicShareSubmissionItem 是公开表单当前匿名访客可见的提交记录。
// 这里只返回展示所需的脱敏字段，不暴露租户、操作者、IP、User-Agent 等内部审计信息。
type PublicShareSubmissionItem struct {
	Status         string      `json:"status"`
	Summary        string      `json:"summary"`
	RequestBody    interface{} `json:"request_body,omitempty"`
	ResponseBody   interface{} `json:"response_body,omitempty"`
	DurationMillis int64       `json:"duration_millis,omitempty"`
	TraceID        string      `json:"trace_id,omitempty"`
	CreatedAt      string      `json:"created_at"`
}

type PublicShareSubmissionListResp struct {
	Items    []*PublicShareSubmissionItem `json:"items"`
	Total    int64                        `json:"total"`
	Page     int                          `json:"page"`
	PageSize int                          `json:"page_size"`
}
