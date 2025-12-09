package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// ChatMessageRepository 智能体聊天消息数据访问层
type ChatMessageRepository struct {
	db *gorm.DB
}

// NewChatMessageRepository 创建聊天消息 Repository
func NewChatMessageRepository(db *gorm.DB) *ChatMessageRepository {
	return &ChatMessageRepository{db: db}
}

// Create 创建消息
func (r *ChatMessageRepository) Create(message *model.AgentChatMessage) error {
	return r.db.Create(message).Error
}

// BatchCreate 批量创建消息
func (r *ChatMessageRepository) BatchCreate(messages []*model.AgentChatMessage) error {
	if len(messages) == 0 {
		return nil
	}
	return r.db.CreateInBatches(messages, 100).Error
}

// GetByID 根据 ID 获取消息
func (r *ChatMessageRepository) GetByID(id int64) (*model.AgentChatMessage, error) {
	var message model.AgentChatMessage
	if err := r.db.Where("id = ?", id).First(&message).Error; err != nil {
		return nil, err
	}
	return &message, nil
}

// ListBySessionID 根据 SessionID 获取消息列表（按创建时间升序）
func (r *ChatMessageRepository) ListBySessionID(sessionID string) ([]*model.AgentChatMessage, error) {
	var messages []*model.AgentChatMessage
	if err := r.db.
		Where("session_id = ?", sessionID).
		Order("created_at ASC").
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// ListBySessionIDWithLimit 根据 SessionID 获取消息列表（限制数量，按创建时间降序）
func (r *ChatMessageRepository) ListBySessionIDWithLimit(sessionID string, limit int) ([]*model.AgentChatMessage, error) {
	var messages []*model.AgentChatMessage
	if err := r.db.
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	// 反转顺序，使其按创建时间升序
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// DeleteBySessionID 根据 SessionID 删除所有消息
func (r *ChatMessageRepository) DeleteBySessionID(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&model.AgentChatMessage{}).Error
}

// Delete 删除消息（根据 ID）
func (r *ChatMessageRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.AgentChatMessage{}).Error
}

