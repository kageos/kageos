package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// ChatSessionRepository 智能体聊天会话数据访问层
type ChatSessionRepository struct {
	db *gorm.DB
}

// NewChatSessionRepository 创建聊天会话 Repository
func NewChatSessionRepository(db *gorm.DB) *ChatSessionRepository {
	return &ChatSessionRepository{db: db}
}

// Create 创建会话
func (r *ChatSessionRepository) Create(session *model.AgentChatSession) error {
	return r.db.Create(session).Error
}

// GetBySessionID 根据 SessionID 获取会话
func (r *ChatSessionRepository) GetBySessionID(sessionID string) (*model.AgentChatSession, error) {
	var session model.AgentChatSession
	if err := r.db.Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// ListByTreeID 根据 TreeID 获取会话列表
func (r *ChatSessionRepository) ListByTreeID(treeID int64, offset, limit int) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.Model(&model.AgentChatSession{}).Where("tree_id = ?", treeID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}


// Update 更新会话
func (r *ChatSessionRepository) Update(session *model.AgentChatSession) error {
	return r.db.Save(session).Error
}

// Delete 删除会话（根据 SessionID）
func (r *ChatSessionRepository) Delete(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&model.AgentChatSession{}).Error
}

// DeleteByTreeID 根据 TreeID 删除所有会话
func (r *ChatSessionRepository) DeleteByTreeID(treeID int64) error {
	return r.db.Where("tree_id = ?", treeID).Delete(&model.AgentChatSession{}).Error
}

