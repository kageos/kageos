package model

import (
	"time"

	"gorm.io/gorm"
)

type MessageEntry struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	From                  string `json:"from" gorm:"size:255;index;comment:发送人"`
	RequestUser           string `json:"request_user" gorm:"size:255;index;comment:请求用户"`
	DepartmentFullPath    string `json:"department_full_path" gorm:"size:500;comment:发送人部门路径"`
	FullCodePath          string `json:"full_code_path" gorm:"size:500;index;comment:来源目录或函数路径"`
	TraceID               string `json:"trace_id" gorm:"size:128;index;comment:链路追踪 ID"`
	ClientSource          string `json:"client_source" gorm:"size:64;comment:入口来源"`
	SourceType            string `json:"source_type" gorm:"size:64;index;comment:来源类型"`
	SourceRef             string `json:"source_ref" gorm:"size:255;index;comment:来源引用"`
	SourcePath            string `json:"source_path" gorm:"size:500;index;comment:消息来源路径"`
	SourceTitle           string `json:"source_title" gorm:"size:500;comment:消息来源展示名"`
	SourceParentPath      string `json:"source_parent_path" gorm:"size:500;index;comment:消息来源父目录路径"`
	SourceParentTitle     string `json:"source_parent_title" gorm:"size:500;comment:消息来源父目录展示名"`
	SourceTemplateType    string `json:"source_template_type" gorm:"size:64;comment:来源函数模板类型"`
	WorkspaceSessionID    string `json:"workspace_session_id" gorm:"size:128;index;comment:关联工作台会话 ID"`
	WorkspaceSessionTitle string `json:"workspace_session_title" gorm:"size:500;comment:关联工作台会话标题"`
	WorkspaceRole         string `json:"workspace_role" gorm:"size:128;comment:关联工作台角色"`
	ThreadKey             string `json:"thread_key" gorm:"size:700;index;comment:站内信聚合线程键"`

	Title       string `json:"title" gorm:"size:500;comment:标题"`
	Content     string `json:"content" gorm:"type:longtext;comment:正文"`
	ContentType string `json:"content_type" gorm:"size:32;default:markdown;comment:markdown/html/text"`
	Files       string `json:"files" gorm:"type:text;comment:通知附件文件引用，多个逗号分隔"`
}

func (MessageEntry) TableName() string {
	return "message_entry"
}

type MessageRecipient struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time      `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	MessageID int64      `json:"message_id" gorm:"not null;index;uniqueIndex:idx_message_recipient_user;comment:消息 ID"`
	Username  string     `json:"username" gorm:"size:255;not null;index;uniqueIndex:idx_message_recipient_user;comment:收件人用户名"`
	ReadAt    *time.Time `json:"read_at" gorm:"index;comment:已读时间"`
}

func (MessageRecipient) TableName() string {
	return "message_recipient"
}

type NotificationChannelSetting struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	OwnerUsername    string     `json:"owner_username" gorm:"size:255;not null;uniqueIndex:idx_notification_owner_channel;index;comment:配置所属用户"`
	Channel          string     `json:"channel" gorm:"size:64;not null;uniqueIndex:idx_notification_owner_channel;index;comment:通知渠道"`
	Enabled          bool       `json:"enabled" gorm:"not null;index;comment:是否启用"`
	DeliveryType     string     `json:"delivery_type" gorm:"size:64;not null;default:webhook;comment:投递类型"`
	DisplayName      string     `json:"display_name" gorm:"size:255;comment:用户展示名"`
	WebhookURLCipher string     `json:"-" gorm:"type:text;comment:webhook 地址密文"`
	SecretCipher     string     `json:"-" gorm:"type:text;comment:签名 secret 密文"`
	Metadata         string     `json:"metadata" gorm:"type:text;comment:渠道扩展配置 JSON"`
	LastSuccessAt    *time.Time `json:"last_success_at" gorm:"index;comment:最近投递成功时间"`
	LastFailedAt     *time.Time `json:"last_failed_at" gorm:"index;comment:最近投递失败时间"`
	LastTestAt       *time.Time `json:"last_test_at" gorm:"index;comment:最近测试时间"`
	LastError        string     `json:"last_error" gorm:"type:text;comment:最近投递错误"`
	FailCount        int        `json:"fail_count" gorm:"not null;default:0;comment:连续失败次数"`
}

func (NotificationChannelSetting) TableName() string {
	return "notification_channel_setting"
}

type NotificationRouteSetting struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	ScopePath        string     `json:"scope_path" gorm:"size:700;not null;uniqueIndex:idx_notification_route_scope_channel;index;comment:服务树作用域路径"`
	ScopeType        string     `json:"scope_type" gorm:"size:64;not null;default:directory;index;comment:作用域类型 workspace/directory/function"`
	Channel          string     `json:"channel" gorm:"size:64;not null;uniqueIndex:idx_notification_route_scope_channel;index;comment:通知渠道"`
	Enabled          bool       `json:"enabled" gorm:"not null;index;comment:是否启用"`
	DeliveryType     string     `json:"delivery_type" gorm:"size:64;not null;default:webhook;comment:投递类型"`
	DisplayName      string     `json:"display_name" gorm:"size:255;comment:展示名称"`
	RequireAuth      bool       `json:"require_auth" gorm:"not null;default:true;comment:群/目录通知处理链接是否必须登录"`
	WebhookURLCipher string     `json:"-" gorm:"type:text;comment:webhook 地址密文"`
	SecretCipher     string     `json:"-" gorm:"type:text;comment:签名 secret 密文"`
	Metadata         string     `json:"metadata" gorm:"type:text;comment:渠道扩展配置 JSON"`
	LastSuccessAt    *time.Time `json:"last_success_at" gorm:"index;comment:最近投递成功时间"`
	LastFailedAt     *time.Time `json:"last_failed_at" gorm:"index;comment:最近投递失败时间"`
	LastTestAt       *time.Time `json:"last_test_at" gorm:"index;comment:最近测试时间"`
	LastError        string     `json:"last_error" gorm:"type:text;comment:最近投递错误"`
	FailCount        int        `json:"fail_count" gorm:"not null;default:0;comment:连续失败次数"`
}

func (NotificationRouteSetting) TableName() string {
	return "notification_route_setting"
}

type MessageActionToken struct {
	ID        int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

	TokenHash          string     `json:"-" gorm:"size:128;not null;uniqueIndex;comment:短期处理 token 哈希"`
	MessageID          int64      `json:"message_id" gorm:"not null;index;comment:原始消息 ID"`
	RecipientUsername  string     `json:"recipient_username" gorm:"size:255;not null;index;comment:被授权处理人"`
	AuthorizedUsers    string     `json:"authorized_users" gorm:"size:2000;comment:允许登录处理的用户，逗号分隔；为空时只允许 recipient_username"`
	Channel            string     `json:"channel" gorm:"size:64;index;comment:触达渠道"`
	RequireAuth        bool       `json:"require_auth" gorm:"not null;default:false;comment:处理链接是否必须登录后使用"`
	AllowedActions     string     `json:"allowed_actions" gorm:"size:255;comment:允许动作，逗号分隔"`
	Status             string     `json:"status" gorm:"size:32;not null;default:open;index;comment:open/submitted/expired/revoked"`
	ExpiresAt          time.Time  `json:"expires_at" gorm:"not null;index;comment:有效期"`
	UsedAt             *time.Time `json:"used_at" gorm:"index;comment:提交时间"`
	ReplyMessageID     int64      `json:"reply_message_id" gorm:"index;comment:回复消息 ID"`
	WorkspaceSessionID string     `json:"workspace_session_id" gorm:"size:128;index;comment:关联工作台会话 ID"`
	ThreadKey          string     `json:"thread_key" gorm:"size:700;index;comment:消息线程键"`
	SourcePath         string     `json:"source_path" gorm:"size:500;index;comment:来源路径"`
	TraceID            string     `json:"trace_id" gorm:"size:128;index;comment:链路追踪 ID"`
}

func (MessageActionToken) TableName() string {
	return "message_action_token"
}
