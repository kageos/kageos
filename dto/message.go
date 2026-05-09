package dto

import "time"

// MessageSendMeta 是消息发送的审计元数据。
// 这些字段应由 SDK 或平台上下文生成，不能让用户正文直接决定发送身份。
type MessageSendMeta struct {
	From               string `json:"from"`                 // 实际发送人，通常等于 request_user，无用户上下文时为 system/后台来源
	RequestUser        string `json:"request_user"`         // 请求用户
	DepartmentFullPath string `json:"department_full_path"` // 请求用户所属部门路径
	FullCodePath       string `json:"full_code_path"`       // 来源目录/函数路径，如 /luobei/example/inventory/stock_in
	TraceID            string `json:"trace_id"`             // 链路追踪 ID
	ClientSource       string `json:"client_source"`        // 入口来源，如 browser、agent、scheduled_task
	SourceType         string `json:"source_type,omitempty"`
	SourceRef          string `json:"source_ref,omitempty"`
}

// MessageSendPayload 是用户真正想发送的消息正文。
// 只描述“发给谁、发什么”，不承载发送人、来源目录等元数据。
type MessageSendPayload struct {
	ToUsers       string `json:"to_users"`       // 接收用户，逗号分隔，如 "zhangsan,lisi"
	ToDepartments string `json:"to_departments"` // 接收部门（full_code_path），逗号分隔
	Title         string `json:"title"`          // 标题/摘要（可选）
	Content       string `json:"content"`        // 正文
	ContentType   string `json:"content_type"`   // 内容类型："markdown"(默认) | "html" | "text"
}

// MessageSendEnvelope 是消息发送在 HTTP/NATS 上的统一包裹结构。
// Meta 来自 ctx，Message 来自用户输入。
type MessageSendEnvelope struct {
	Meta    MessageSendMeta    `json:"meta"`
	Message MessageSendPayload `json:"message"`
}

// MessageSendToUsersReq 按用户发送消息。发送人和来源由 token/ctx 自动解析，不从 body 读取。
type MessageSendToUsersReq struct {
	ToUsers     string `json:"to_users" binding:"required"`
	Title       string `json:"title"`
	Content     string `json:"content" binding:"required"`
	ContentType string `json:"content_type"`
}

// MessageSendToDepartmentsReq 按部门发送消息。发送人和来源由 token/ctx 自动解析，不从 body 读取。
type MessageSendToDepartmentsReq struct {
	ToDepartments string `json:"to_departments" binding:"required"`
	Title         string `json:"title"`
	Content       string `json:"content" binding:"required"`
	ContentType   string `json:"content_type"`
}

// MessageSendResp 消息发送响应。
type MessageSendResp struct {
	Message       string             `json:"message"`
	Meta          MessageSendMeta    `json:"meta"`
	Payload       MessageSendPayload `json:"payload"`
	From          string             `json:"from"`
	FullCodePath  string             `json:"full_code_path"`
	ToUsers       string             `json:"to_users"`
	ToDepartments string             `json:"to_departments"`
	ContentType   string             `json:"content_type"`
}

// MessageInboxItem 站内信列表/详情项。
type MessageInboxItem struct {
	ID                 int64                 `json:"id"`
	RecipientID        int64                 `json:"recipient_id"`
	From               string                `json:"from"`
	RequestUser        string                `json:"request_user"`
	DepartmentFullPath string                `json:"department_full_path"`
	FullCodePath       string                `json:"full_code_path"`
	TraceID            string                `json:"trace_id"`
	ClientSource       string                `json:"client_source"`
	SourceType         string                `json:"source_type"`
	SourceRef          string                `json:"source_ref"`
	Title              string                `json:"title"`
	Content            string                `json:"content"`
	ContentType        string                `json:"content_type"`
	ReadAt             *time.Time            `json:"read_at"`
	CreatedAt          time.Time             `json:"created_at"`
	SourceDisplay      *MessageSourceDisplay `json:"source_display,omitempty"`
}

// MessageSourceDisplay 是消息来源在服务树中的展示信息。
type MessageSourceDisplay struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	TemplateType string `json:"template_type,omitempty"`
	FullCodePath string `json:"full_code_path,omitempty"`
}

// MessageInboxListResp 站内信列表响应。
type MessageInboxListResp struct {
	List     []MessageInboxItem `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type MessageUnreadCountResp struct {
	UnreadCount int64 `json:"unread_count"`
}
