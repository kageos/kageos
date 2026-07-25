package repository

import (
	"errors"

	"github.com/kageos/kageos/core/agent-server/model"
	"gorm.io/gorm"
)

// ChatMessageRepository 工作台聊天消息数据访问层
type ChatMessageRepository struct {
	db *gorm.DB
}

func (r *ChatMessageRepository) SupportsContextCheckpoints() bool {
	return r != nil && r.db != nil && r.db.Migrator().HasTable(&model.AgentChatContextCheckpoint{})
}

func (r *ChatMessageRepository) GetLatestContextCheckpoint(sessionID string) (*model.AgentChatContextCheckpoint, error) {
	if !r.SupportsContextCheckpoints() {
		return nil, gorm.ErrRecordNotFound
	}
	var checkpoint model.AgentChatContextCheckpoint
	err := r.db.Where("session_id = ?", sessionID).Order("covered_to_message_id DESC, id DESC").First(&checkpoint).Error
	if err != nil {
		return nil, err
	}
	return &checkpoint, nil
}

func (r *ChatMessageRepository) CreateContextCheckpoint(checkpoint *model.AgentChatContextCheckpoint) error {
	if checkpoint == nil {
		return errors.New("context checkpoint is nil")
	}
	if !r.SupportsContextCheckpoints() {
		return errors.New("context checkpoint table is unavailable")
	}
	return r.db.Create(checkpoint).Error
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

// ListBySessionID 根据 SessionID 获取消息列表（按 id 升序，保证 assistant 在对应 tool 之前，避免 API 报错「tool 必须紧接在含 tool_calls 的 assistant 之后」）
func (r *ChatMessageRepository) ListBySessionID(sessionID string) ([]*model.AgentChatMessage, error) {
	var messages []*model.AgentChatMessage
	if err := r.db.
		Where("session_id = ?", sessionID).
		Order("id ASC").
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		if tx.Migrator().HasTable(&model.AgentChatContextCheckpoint{}) {
			if err := tx.Where("session_id = ?", sessionID).Delete(&model.AgentChatContextCheckpoint{}).Error; err != nil {
				return err
			}
		}
		return tx.Where("session_id = ?", sessionID).Delete(&model.AgentChatMessage{}).Error
	})
}

// Delete 删除消息（根据 ID）
func (r *ChatMessageRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.AgentChatMessage{}).Error
}
