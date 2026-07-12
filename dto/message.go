package dto

import "time"

// MessageSendMeta is audit metadata for a message send request.
// SDK/platform callers should populate this from request context instead of
// letting business payload decide sender identity.
type MessageSendMeta struct {
	From                  string `json:"from"`
	RequestUser           string `json:"request_user"`
	DepartmentFullPath    string `json:"department_full_path"`
	FullCodePath          string `json:"full_code_path"`
	TraceID               string `json:"trace_id"`
	ClientSource          string `json:"client_source"`
	SourceType            string `json:"source_type,omitempty"`
	SourceRef             string `json:"source_ref,omitempty"`
	SourcePath            string `json:"source_path,omitempty"`
	SourceTitle           string `json:"source_title,omitempty"`
	SourceParentPath      string `json:"source_parent_path,omitempty"`
	SourceParentTitle     string `json:"source_parent_title,omitempty"`
	SourceTemplateType    string `json:"source_template_type,omitempty"`
	WorkspaceSessionID    string `json:"workspace_session_id,omitempty"`
	WorkspaceSessionTitle string `json:"workspace_session_title,omitempty"`
	WorkspaceRole         string `json:"workspace_role,omitempty"`
	ThreadKey             string `json:"thread_key,omitempty"`
}

// MessageSendPayload describes recipients and content. Delivery channels are
// intentionally owned by message-service.
type MessageSendPayload struct {
	ToUsers     string `json:"to_users"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Files       string `json:"files,omitempty"`
}

type MessageSendEnvelope struct {
	Meta    MessageSendMeta    `json:"meta"`
	Message MessageSendPayload `json:"message"`
}

type MessageSendToUsersReq struct {
	ToUsers     string `json:"to_users"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"content_type"`
	Files       string `json:"files,omitempty"`
}

type MessageSendResp struct {
	Message      string             `json:"message"`
	Meta         MessageSendMeta    `json:"meta"`
	Payload      MessageSendPayload `json:"payload"`
	From         string             `json:"from"`
	FullCodePath string             `json:"full_code_path"`
	ToUsers      string             `json:"to_users"`
	ContentType  string             `json:"content_type"`
	Files        string             `json:"files,omitempty"`
}

type MessageInboxItem struct {
	ID                    int64                 `json:"id"`
	RecipientID           int64                 `json:"recipient_id"`
	From                  string                `json:"from"`
	RequestUser           string                `json:"request_user"`
	DepartmentFullPath    string                `json:"department_full_path"`
	FullCodePath          string                `json:"full_code_path"`
	TraceID               string                `json:"trace_id"`
	ClientSource          string                `json:"client_source"`
	SourceType            string                `json:"source_type"`
	SourceRef             string                `json:"source_ref"`
	SourcePath            string                `json:"source_path"`
	SourceTitle           string                `json:"source_title"`
	SourceParentPath      string                `json:"source_parent_path"`
	SourceParentTitle     string                `json:"source_parent_title"`
	SourceTemplateType    string                `json:"source_template_type"`
	WorkspaceSessionID    string                `json:"workspace_session_id"`
	WorkspaceSessionTitle string                `json:"workspace_session_title"`
	WorkspaceRole         string                `json:"workspace_role"`
	ThreadKey             string                `json:"thread_key"`
	ScheduledTaskID       int64                 `json:"scheduled_task_id,omitempty" gorm:"-"`
	ScheduledExecutionID  int64                 `json:"scheduled_execution_id,omitempty" gorm:"-"`
	Title                 string                `json:"title"`
	Content               string                `json:"content"`
	ContentType           string                `json:"content_type"`
	Files                 string                `json:"files,omitempty"`
	ReadAt                *time.Time            `json:"read_at"`
	CreatedAt             time.Time             `json:"created_at"`
	SourceDisplay         *MessageSourceDisplay `json:"source_display,omitempty" gorm:"-"`
}

type MessageSourceDisplay struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	TemplateType       string `json:"template_type,omitempty"`
	FullCodePath       string `json:"full_code_path,omitempty"`
	ParentName         string `json:"parent_name,omitempty"`
	ParentFullCodePath string `json:"parent_full_code_path,omitempty"`
	ThreadKey          string `json:"thread_key,omitempty"`
}

type MessageInboxListResp struct {
	List     []MessageInboxItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type MessageInboxSourceCount struct {
	SourcePath   string    `json:"source_path"`
	UnreadCount  int64     `json:"unread_count"`
	MessageCount int64     `json:"message_count"`
	LatestAt     time.Time `json:"latest_at"`
}

type MessageInboxSourceCountResp struct {
	List []MessageInboxSourceCount `json:"list"`
}

type MessageInboxWorkspaceCount struct {
	WorkspaceKey      string    `json:"workspace_key"`
	WorkspaceUser     string    `json:"workspace_user"`
	WorkspaceCode     string    `json:"workspace_code"`
	WorkspacePath     string    `json:"workspace_path"`
	Title             string    `json:"title"`
	UnreadCount       int64     `json:"unread_count"`
	MessageCount      int64     `json:"message_count"`
	LatestAt          time.Time `json:"latest_at"`
	LatestSourcePath  string    `json:"latest_source_path"`
	LatestSourceTitle string    `json:"latest_source_title"`
}

type MessageInboxWorkspaceCountResp struct {
	List []MessageInboxWorkspaceCount `json:"list"`
}

type MessageInboxThread struct {
	Key                  string           `json:"key"`
	Kind                 string           `json:"kind"`
	Title                string           `json:"title"`
	Subtitle             string           `json:"subtitle"`
	Path                 string           `json:"path"`
	UnreadCount          int64            `json:"unread_count"`
	MessageCount         int64            `json:"message_count"`
	LatestAt             time.Time        `json:"latest_at"`
	LastMessage          MessageInboxItem `json:"last_message"`
	ScheduledTaskID      int64            `json:"scheduled_task_id,omitempty"`
	ScheduledExecutionID int64            `json:"scheduled_execution_id,omitempty"`
}

type MessageInboxThreadListResp struct {
	List     []MessageInboxThread `json:"list"`
	Total    int64                `json:"total"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"page_size"`
}

type MessageUnreadCountResp struct {
	UnreadCount int64 `json:"unread_count"`
}

type MessageActionTokenStatus string

const (
	MessageActionTokenStatusOpen       MessageActionTokenStatus = "open"
	MessageActionTokenStatusProcessing MessageActionTokenStatus = "processing"
	MessageActionTokenStatusSubmitted  MessageActionTokenStatus = "submitted"
	MessageActionTokenStatusExpired    MessageActionTokenStatus = "expired"
	MessageActionTokenStatusRevoked    MessageActionTokenStatus = "revoked"
)

type MessageActionViewResp struct {
	TokenStatus       string             `json:"token_status"`
	RecipientUser     string             `json:"recipient_user"`
	Channel           string             `json:"channel,omitempty"`
	AuthenticatedUser string             `json:"authenticated_user,omitempty"`
	AllowedActions    []string           `json:"allowed_actions"`
	CanReply          bool               `json:"can_reply"`
	ExpiresAt         time.Time          `json:"expires_at"`
	Message           MessageInboxItem   `json:"message"`
	Thread            []MessageInboxItem `json:"thread"`
	MobileAskURL      string             `json:"mobile_ask_url"`
	WorkspaceSession  string             `json:"workspace_session_id,omitempty"`
	SubmittedAt       *time.Time         `json:"submitted_at,omitempty"`
	ReplyMessageID    int64              `json:"reply_message_id,omitempty"`
}

type MessageActionReplyReq struct {
	Content string `json:"content" binding:"required"`
	Files   string `json:"files,omitempty"`
	Action  string `json:"action"`
}

type MessageActionReplyResp struct {
	Status             string    `json:"status"`
	ReplyMessageID     int64     `json:"reply_message_id"`
	SubmittedAt        time.Time `json:"submitted_at"`
	MobileAskURL       string    `json:"mobile_ask_url,omitempty"`
	Channel            string    `json:"channel,omitempty"`
	SourcePath         string    `json:"source_path,omitempty"`
	FullCodePath       string    `json:"full_code_path,omitempty"`
	WorkspaceSessionID string    `json:"workspace_session_id,omitempty"`
	AgentSubmitted     bool      `json:"agent_submitted"`
	AgentSubmitError   string    `json:"agent_submit_error,omitempty"`
	WorkstationDraft   string    `json:"workstation_draft,omitempty"`
}

type MessageNotificationChannelInfo struct {
	Channel       string            `json:"channel"`
	Enabled       bool              `json:"enabled"`
	DeliveryType  string            `json:"delivery_type"`
	DisplayName   string            `json:"display_name"`
	HasWebhookURL bool              `json:"has_webhook_url"`
	HasSecret     bool              `json:"has_secret"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at"`
	LastSuccessAt *time.Time        `json:"last_success_at,omitempty"`
	LastFailedAt  *time.Time        `json:"last_failed_at,omitempty"`
	LastTestAt    *time.Time        `json:"last_test_at,omitempty"`
	LastError     string            `json:"last_error,omitempty"`
	FailCount     int               `json:"fail_count"`
}

type MessageNotificationChannelListResp struct {
	List []MessageNotificationChannelInfo `json:"list"`
}

type MessageNotificationRouteInfo struct {
	ID            int64             `json:"id"`
	ScopePath     string            `json:"scope_path"`
	ScopeType     string            `json:"scope_type"`
	Channel       string            `json:"channel"`
	Enabled       bool              `json:"enabled"`
	DeliveryType  string            `json:"delivery_type"`
	DisplayName   string            `json:"display_name"`
	Remark        string            `json:"remark"`
	HasWebhookURL bool              `json:"has_webhook_url"`
	HasSecret     bool              `json:"has_secret"`
	Metadata      map[string]string `json:"metadata,omitempty"`
	UpdatedAt     time.Time         `json:"updated_at"`
	LastSuccessAt *time.Time        `json:"last_success_at,omitempty"`
	LastFailedAt  *time.Time        `json:"last_failed_at,omitempty"`
	LastTestAt    *time.Time        `json:"last_test_at,omitempty"`
	LastError     string            `json:"last_error,omitempty"`
	FailCount     int               `json:"fail_count"`
}

type MessageNotificationRouteListResp struct {
	List []MessageNotificationRouteInfo `json:"list"`
}

type MessageNotificationRoutePathSummary struct {
	ScopePath string                         `json:"scope_path"`
	Routes    []MessageNotificationRouteInfo `json:"routes"`
}

type MessageNotificationRouteSummaryResp struct {
	Routes map[string]MessageNotificationRoutePathSummary `json:"routes"`
}

type UpsertMessageNotificationChannelReq struct {
	Channel         string            `json:"channel"`
	Enabled         *bool             `json:"enabled"`
	DeliveryType    string            `json:"delivery_type"`
	DisplayName     string            `json:"display_name"`
	WebhookURL      string            `json:"webhook_url"`
	Secret          string            `json:"secret"`
	ClearWebhookURL bool              `json:"clear_webhook_url"`
	ClearSecret     bool              `json:"clear_secret"`
	Metadata        map[string]string `json:"metadata"`
}

type UpsertMessageNotificationRouteReq struct {
	ScopePath       string            `json:"scope_path" binding:"required"`
	ScopeType       string            `json:"scope_type"`
	Channel         string            `json:"channel"`
	Enabled         *bool             `json:"enabled"`
	DeliveryType    string            `json:"delivery_type"`
	DisplayName     string            `json:"display_name"`
	Remark          *string           `json:"remark"`
	WebhookURL      string            `json:"webhook_url"`
	Secret          string            `json:"secret"`
	ClearWebhookURL bool              `json:"clear_webhook_url"`
	ClearSecret     bool              `json:"clear_secret"`
	Metadata        map[string]string `json:"metadata"`
}

type TestMessageNotificationChannelResp struct {
	Message string `json:"message"`
	Channel string `json:"channel"`
}
