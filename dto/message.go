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
	WorkspaceSessionID    string                `json:"workspace_session_id"`
	WorkspaceSessionTitle string                `json:"workspace_session_title"`
	WorkspaceRole         string                `json:"workspace_role"`
	ThreadKey             string                `json:"thread_key"`
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

type MessageUnreadCountResp struct {
	UnreadCount int64 `json:"unread_count"`
}
