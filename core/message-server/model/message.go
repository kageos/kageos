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

	From               string `json:"from" gorm:"size:255;index;comment:发送人"`
	RequestUser        string `json:"request_user" gorm:"size:255;index;comment:请求用户"`
	DepartmentFullPath string `json:"department_full_path" gorm:"size:500;comment:发送人部门路径"`
	FullCodePath       string `json:"full_code_path" gorm:"size:500;index;comment:来源目录或函数路径"`
	TraceID            string `json:"trace_id" gorm:"size:128;index;comment:链路追踪 ID"`
	ClientSource       string `json:"client_source" gorm:"size:64;comment:入口来源"`
	SourceType         string `json:"source_type" gorm:"size:64;index;comment:来源类型"`
	SourceRef          string `json:"source_ref" gorm:"size:255;index;comment:来源引用"`

	Title       string `json:"title" gorm:"size:500;comment:标题"`
	Content     string `json:"content" gorm:"type:longtext;comment:正文"`
	ContentType string `json:"content_type" gorm:"size:32;default:markdown;comment:markdown/html/text"`
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
