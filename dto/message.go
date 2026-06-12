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
	SourceIcon            string `json:"source_icon,omitempty"`
	SourceColor           string `json:"source_color,omitempty"`
	SourceParentIcon      string `json:"source_parent_icon,omitempty"`
	SourceParentColor     string `json:"source_parent_color,omitempty"`
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
}

type MessageSendEnvelope struct {
	Meta    MessageSendMeta    `json:"meta"`
	Message MessageSendPayload `json:"message"`
}

type MessageSendToUsersReq struct {
	ToUsers     string `json:"to_users" binding:"required"`
	Title       string `json:"title"`
	Content     string `json:"content" binding:"required"`
	ContentType string `json:"content_type"`
}

type MessageSendResp struct {
	Message      string             `json:"message"`
	Meta         MessageSendMeta    `json:"meta"`
	Payload      MessageSendPayload `json:"payload"`
	From         string             `json:"from"`
	FullCodePath string             `json:"full_code_path"`
	ToUsers      string             `json:"to_users"`
	ContentType  string             `json:"content_type"`
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
	SourceIcon            string                `json:"source_icon"`
	SourceColor           string                `json:"source_color"`
	SourceParentIcon      string                `json:"source_parent_icon"`
	SourceParentColor     string                `json:"source_parent_color"`
	WorkspaceSessionID    string                `json:"workspace_session_id"`
	WorkspaceSessionTitle string                `json:"workspace_session_title"`
	WorkspaceRole         string                `json:"workspace_role"`
	ThreadKey             string                `json:"thread_key"`
	ScheduledTaskID       int64                 `json:"scheduled_task_id,omitempty" gorm:"-"`
	ScheduledExecutionID  int64                 `json:"scheduled_execution_id,omitempty" gorm:"-"`
	Title                 string                `json:"title"`
	Content               string                `json:"content"`
	ContentType           string                `json:"content_type"`
	ReadAt                *time.Time            `json:"read_at"`
	CreatedAt             time.Time             `json:"created_at"`
	SourceDisplay         *MessageSourceDisplay `json:"source_display,omitempty" gorm:"-"`
}

type MessageSourceDisplay struct {
	Name               string `json:"name"`
	Type               string `json:"type"`
	TemplateType       string `json:"template_type,omitempty"`
	FullCodePath       string `json:"full_code_path,omitempty"`
	Icon               string `json:"icon,omitempty"`
	Color              string `json:"color,omitempty"`
	ParentName         string `json:"parent_name,omitempty"`
	ParentFullCodePath string `json:"parent_full_code_path,omitempty"`
	ParentIcon         string `json:"parent_icon,omitempty"`
	ParentColor        string `json:"parent_color,omitempty"`
	ThreadKey          string `json:"thread_key,omitempty"`
}

type MessageInboxListResp struct {
	List     []MessageInboxItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type MessageInboxThread struct {
	Key                  string           `json:"key"`
	Kind                 string           `json:"kind"`
	Title                string           `json:"title"`
	Subtitle             string           `json:"subtitle"`
	Path                 string           `json:"path"`
	Icon                 string           `json:"icon,omitempty"`
	Color                string           `json:"color,omitempty"`
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
