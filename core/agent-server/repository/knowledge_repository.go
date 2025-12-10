package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// KnowledgeRepository 知识库数据访问层
type KnowledgeRepository struct {
	db *gorm.DB
}

// NewKnowledgeRepository 创建知识库 Repository
func NewKnowledgeRepository(db *gorm.DB) *KnowledgeRepository {
	return &KnowledgeRepository{db: db}
}

// Create 创建知识库
func (r *KnowledgeRepository) Create(kb *model.KnowledgeBase) error {
	return r.db.Create(kb).Error
}

// GetByID 根据 ID 获取知识库
func (r *KnowledgeRepository) GetByID(id int64) (*model.KnowledgeBase, error) {
	var kb model.KnowledgeBase
	if err := r.db.Where("id = ?", id).First(&kb).Error; err != nil {
		return nil, err
	}
	return &kb, nil
}


// List 获取知识库列表
func (r *KnowledgeRepository) List(offset, limit int) ([]*model.KnowledgeBase, int64, error) {
	var kbs []*model.KnowledgeBase
	var total int64

	query := r.db.Model(&model.KnowledgeBase{})

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.Offset(offset).Limit(limit).Order("id DESC").Find(&kbs).Error; err != nil {
		return nil, 0, err
	}

	return kbs, total, nil
}

// Update 更新知识库
func (r *KnowledgeRepository) Update(kb *model.KnowledgeBase) error {
	return r.db.Save(kb).Error
}

// Delete 删除知识库（根据 ID）
func (r *KnowledgeRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.KnowledgeBase{}).Error
}


// AddDocument 添加文档
func (r *KnowledgeRepository) AddDocument(doc *model.KnowledgeDocument) error {
	return r.db.Create(doc).Error
}

// GetDocumentsByKBID 根据知识库 ID 获取文档列表（保留兼容，平铺列表）
func (r *KnowledgeRepository) GetDocumentsByKBID(kbID int64, offset, limit int) ([]*model.KnowledgeDocument, int64, error) {
	var docs []*model.KnowledgeDocument
	var total int64

	query := r.db.Model(&model.KnowledgeDocument{}).Where("knowledge_base_id = ?", kbID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表（按排序字段和ID排序）
	if err := query.Offset(offset).Limit(limit).Order("sort_order ASC, id ASC").Find(&docs).Error; err != nil {
		return nil, 0, err
	}

	return docs, total, nil
}

// GetDocumentsTreeByKBID 根据知识库 ID 获取文档树（目录结构）
func (r *KnowledgeRepository) GetDocumentsTreeByKBID(kbID int64) ([]*model.KnowledgeDocument, error) {
	var docs []*model.KnowledgeDocument
	// 获取所有文档（不分页），按排序字段和ID排序
	if err := r.db.
		Where("knowledge_base_id = ?", kbID).
		Order("parent_id ASC, sort_order ASC, id ASC").
		Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// GetDocumentByID 根据 ID 获取文档
func (r *KnowledgeRepository) GetDocumentByID(id int64) (*model.KnowledgeDocument, error) {
	var doc model.KnowledgeDocument
	if err := r.db.Where("id = ?", id).First(&doc).Error; err != nil {
		return nil, err
	}
	return &doc, nil
}

// UpdateDocument 更新文档
func (r *KnowledgeRepository) UpdateDocument(doc *model.KnowledgeDocument) error {
	return r.db.Save(doc).Error
}

// DeleteDocument 删除文档
func (r *KnowledgeRepository) DeleteDocument(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.KnowledgeDocument{}).Error
}

// UpdateDocumentSort 批量更新文档排序
func (r *KnowledgeRepository) UpdateDocumentSort(updates []struct {
	ID        int64
	ParentID  int64
	SortOrder int
	Path      string
}) error {
	// 使用事务批量更新
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			if err := tx.Model(&model.KnowledgeDocument{}).
				Where("id = ?", update.ID).
				Updates(map[string]interface{}{
					"parent_id":  update.ParentID,
					"sort_order": update.SortOrder,
					"path":       update.Path,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetAllDocumentsByKBID 根据知识库 ID 获取所有文档（用于 RAG，全量加载）
func (r *KnowledgeRepository) GetAllDocumentsByKBID(kbID int64) ([]*model.KnowledgeDocument, error) {
	var docs []*model.KnowledgeDocument
	if err := r.db.
		Where("knowledge_base_id = ? AND status = ?", kbID, "completed").
		Order("created_at ASC").
		Find(&docs).Error; err != nil {
		return nil, err
	}
	return docs, nil
}

// GetAllChunksByKBID 根据知识库 ID 获取所有 chunks（用于 RAG，全量加载）
func (r *KnowledgeRepository) GetAllChunksByKBID(kbID int64) ([]*model.KnowledgeChunk, error) {
	var chunks []*model.KnowledgeChunk
	if err := r.db.
		Where("knowledge_base_id = ?", kbID).
		Order("doc_id ASC, chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetAllChunksByDocID 根据文档 ID 获取所有 chunks
func (r *KnowledgeRepository) GetAllChunksByDocID(kbID int64, docID string) ([]*model.KnowledgeChunk, error) {
	var chunks []*model.KnowledgeChunk
	if err := r.db.
		Where("knowledge_base_id = ? AND doc_id = ?", kbID, docID).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return nil, err
	}
	return chunks, nil
}


