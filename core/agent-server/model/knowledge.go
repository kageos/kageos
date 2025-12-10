package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// KnowledgeBase 知识库模型
type KnowledgeBase struct {
	models.Base
	Name          string `gorm:"column:name;comment:知识库名称" json:"name"`
	Description   string `gorm:"column:description;comment:知识库描述" json:"description"`
	Status        string `gorm:"column:status;comment:状态(active/inactive)" json:"status"`
	DocumentCount int    `gorm:"column:document_count;comment:文档数量" json:"document_count"`
	ContentHash   string `gorm:"column:content_hash;type:varchar(64);comment:知识库内容哈希值" json:"content_hash"`
	User          string `gorm:"column:user;comment:创建用户" json:"user"`
}

// TableName 指定表名
func (KnowledgeBase) TableName() string {
	return "knowledge_bases"
}

// KnowledgeDocument 知识库文档模型
type KnowledgeDocument struct {
	models.Base
	KnowledgeBaseID int64  `gorm:"column:knowledge_base_id;type:bigint;not null;index;comment:知识库ID" json:"knowledge_base_id"`
	ParentID        int64  `gorm:"column:parent_id;type:bigint;default:0;index;comment:父文档ID（0表示根目录）" json:"parent_id"`
	DocID           string `gorm:"column:doc_id;type:varchar(255);comment:文档ID" json:"doc_id"`
	Title           string `gorm:"column:title;comment:文档标题" json:"title"`
	Content         string `gorm:"column:content;type:longtext;comment:文档内容" json:"content"`
	FileType        string `gorm:"column:file_type;comment:文件类型(pdf/txt/doc/md)" json:"file_type"`
	FileSize        int64  `gorm:"column:file_size;comment:文件大小(字节)" json:"file_size"`
	Status          string `gorm:"column:status;comment:状态(completed/failed)" json:"status"`
	SortOrder       int    `gorm:"column:sort_order;default:0;index;comment:排序字段（数字越小越靠前）" json:"sort_order"`
	Path            string `gorm:"column:path;type:varchar(512);comment:路径（用于快速查询，如：/目录1/目录2/文档）" json:"path"`
	User            string `gorm:"column:user;comment:上传用户" json:"user"`
}

// TableName 指定表名
func (KnowledgeDocument) TableName() string {
	return "knowledge_documents"
}

// KnowledgeChunk 知识库文档分块模型
type KnowledgeChunk struct {
	models.Base
	KnowledgeBaseID int64  `gorm:"column:knowledge_base_id;type:bigint;not null;index;comment:知识库ID" json:"knowledge_base_id"`
	DocID           string `gorm:"column:doc_id;type:varchar(255);index;comment:文档ID" json:"doc_id"`
	ChunkID         string `gorm:"column:chunk_id;type:varchar(255);comment:分块ID" json:"chunk_id"`
	Content         string `gorm:"column:content;type:longtext;comment:分块内容" json:"content"`
	ChunkIndex      int    `gorm:"column:chunk_index;comment:分块序号" json:"chunk_index"`
	User            string `gorm:"column:user;comment:用户标识" json:"user"`
}

// TableName 指定表名
func (KnowledgeChunk) TableName() string {
	return "knowledge_chunks"
}

