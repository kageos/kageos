package model

const (
	WorkflowStatusDraft    = "draft"
	WorkflowStatusEnabled  = "enabled"
	WorkflowStatusDisabled = "disabled"
)

const (
	VersionStatusDraft     = "draft"
	VersionStatusPublished = "published"
	VersionStatusArchived  = "archived"
)

const (
	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusWaiting   = "waiting"
	RunStatusSuccess   = "success"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"
	RunStatusTimeout   = "timeout"
)

const (
	StepStatusPending   = "pending"
	StepStatusRunning   = "running"
	StepStatusWaiting   = "waiting"
	StepStatusSuccess   = "success"
	StepStatusFailed    = "failed"
	StepStatusSkipped   = "skipped"
	StepStatusCancelled = "cancelled"
)
