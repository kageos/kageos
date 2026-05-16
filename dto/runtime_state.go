package dto

import "time"

// RuntimeStateItem 表示服务端当前维护的一条运行态。
// 运行态是短生命周期数据，当前由 agent-server 内存维护，后续可迁移到 Redis/NATS KV 等分布式实现。
type RuntimeStateItem struct {
	Key          string                 `json:"key"`
	Kind         string                 `json:"kind"`
	Status       string                 `json:"status"`
	Stage        string                 `json:"stage,omitempty"`
	FullCodePath string                 `json:"full_code_path"`
	Title        string                 `json:"title,omitempty"`
	User         string                 `json:"user,omitempty"`
	ModeCode     string                 `json:"mode_code,omitempty"`
	SessionID    string                 `json:"session_id,omitempty"`
	SourceType   string                 `json:"source_type,omitempty"`
	SourceRef    string                 `json:"source_ref,omitempty"`
	StartedAt    time.Time              `json:"started_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	ExpiresAt    *time.Time             `json:"expires_at,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// RuntimeStateSummary 是按服务目录聚合后的运行态摘要，用于服务树 badge 展示。
type RuntimeStateSummary struct {
	RunningCount         int       `json:"running_count"`
	ManualRunningCount   int       `json:"manual_running_count"`
	ThinkingCount        int       `json:"thinking_count"`
	ToolRunningCount     int       `json:"tool_running_count"`
	WaitingApprovalCount int       `json:"waiting_approval_count"`
	FailedRecentCount    int       `json:"failed_recent_count"`
	LastActivityAt       time.Time `json:"last_activity_at"`
	DominantStatus       string    `json:"dominant_status,omitempty"`
	BadgeText            string    `json:"badge_text,omitempty"`
	BadgeTone            string    `json:"badge_tone,omitempty"`
	Tooltip              string    `json:"tooltip,omitempty"`
}

type RuntimeStateSummaryResp struct {
	Summaries map[string]RuntimeStateSummary `json:"summaries"`
}

type RuntimeStateItemsResp struct {
	Items []RuntimeStateItem `json:"items"`
}
