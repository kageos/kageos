package repository

import (
	"github.com/kageos/kageos/core/agent-server/model"
	"gorm.io/gorm"
)

// ChatSessionRepository 工作台聊天会话数据访问层
type ChatSessionRepository struct {
	db *gorm.DB
}

// NewChatSessionRepository 创建聊天会话 Repository
func NewChatSessionRepository(db *gorm.DB) *ChatSessionRepository {
	return &ChatSessionRepository{db: db}
}

// Transaction 在同一个数据库事务内执行会话仓储操作。
func (r *ChatSessionRepository) Transaction(fn func(tx *ChatSessionRepository) error) error {
	return r.db.Transaction(func(db *gorm.DB) error {
		return fn(&ChatSessionRepository{db: db})
	})
}

// TransactionWithMessages 在同一个数据库事务内执行会话与消息仓储操作。
func (r *ChatSessionRepository) TransactionWithMessages(fn func(sessionTx *ChatSessionRepository, messageTx *ChatMessageRepository) error) error {
	return r.db.Transaction(func(db *gorm.DB) error {
		return fn(&ChatSessionRepository{db: db}, &ChatMessageRepository{db: db})
	})
}

// TransactionWithMessagesAndHandoffPackets 在同一个数据库事务内执行会话、消息和 handoff packet 仓储操作。
func (r *ChatSessionRepository) TransactionWithMessagesAndHandoffPackets(fn func(sessionTx *ChatSessionRepository, messageTx *ChatMessageRepository, handoffTx *WorkspaceHandoffPacketRepository) error) error {
	return r.db.Transaction(func(db *gorm.DB) error {
		return fn(&ChatSessionRepository{db: db}, &ChatMessageRepository{db: db}, &WorkspaceHandoffPacketRepository{db: db})
	})
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

	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// ListByFullCodePath 根据 FullCodePath 获取会话列表（工作台使用）
func (r *ChatSessionRepository) ListByFullCodePath(fullCodePath string, offset, limit int) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.Model(&model.AgentChatSession{}).Where("full_code_path = ?", fullCodePath)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// ListByFullCodePathAndUser 根据 FullCodePath 和用户获取会话列表（工作台使用）。
func (r *ChatSessionRepository) ListByFullCodePathAndUser(fullCodePath string, user string, offset, limit int) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.Model(&model.AgentChatSession{}).Where("full_code_path = ? AND user = ?", fullCodePath, user)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

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

// TryMarkGenerating 只在会话当前未生成时把状态标记为 generating，避免同一会话并发进入模型。
func (r *ChatSessionRepository) TryMarkGenerating(sessionID string, user string, modeCode string) (bool, error) {
	updates := map[string]interface{}{
		"status": model.ChatSessionStatusGenerating,
	}
	if user != "" {
		updates["updated_by"] = user
	}
	if modeCode != "" {
		updates["mode_code"] = modeCode
	}
	res := r.db.Model(&model.AgentChatSession{}).
		Where("session_id = ? AND status <> ?", sessionID, model.ChatSessionStatusGenerating).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ArchiveForModelIfActive 只在来源会话尚未归档时归档，避免重复 handoff 创建多个目标会话。
func (r *ChatSessionRepository) ArchiveForModelIfActive(sessionID string, targetRoleName string, user string) (bool, error) {
	updates := map[string]interface{}{
		"archived_for_model": true,
		"context_policy":     "display_only",
		"archive_reason":     "已交接到" + targetRoleName + "，会话仅保留展示历史",
		"status":             model.ChatSessionStatusDone,
	}
	if user != "" {
		updates["updated_by"] = user
	}
	res := r.db.Model(&model.AgentChatSession{}).
		Where("session_id = ? AND archived_for_model = ?", sessionID, false).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Delete 删除会话（根据 SessionID）
func (r *ChatSessionRepository) Delete(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&model.AgentChatSession{}).Error
}

// DeleteByTreeID 根据 TreeID 删除所有会话
func (r *ChatSessionRepository) DeleteByTreeID(treeID int64) error {
	return r.db.Where("tree_id = ?", treeID).Delete(&model.AgentChatSession{}).Error
}

// ListRunningByUser 查询指定用户所有正在执行（generating）的工作台会话
func (r *ChatSessionRepository) ListRunningByUser(user string) ([]*model.AgentChatSession, error) {
	var sessions []*model.AgentChatSession
	if err := r.db.
		Where("user = ? AND source = ? AND status = ?", user, "workspace", model.ChatSessionStatusGenerating).
		Order("updated_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListFinishedByUser 查询指定用户最近非执行中的工作台会话（active/output/pending/done/cancelled）
func (r *ChatSessionRepository) ListFinishedByUser(user string, limit int) ([]*model.AgentChatSession, error) {
	var sessions []*model.AgentChatSession
	if err := r.db.
		Where("user = ? AND source = ? AND status IN ?", user, "workspace",
			[]string{
				model.ChatSessionStatusActive,
				model.ChatSessionStatusOutput,
				model.ChatSessionStatusPendingConfirmation,
				model.ChatSessionStatusPendingTest,
				model.ChatSessionStatusDone,
				model.ChatSessionStatusCancelled,
			}).
		Order("updated_at DESC").
		Limit(limit).
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// GetServiceTreeNamesByFullCodePaths 批量查询工作台会话所属目录的展示名称。
func (r *ChatSessionRepository) GetServiceTreeNamesByFullCodePaths(fullCodePaths []string) (map[string]string, error) {
	if len(fullCodePaths) == 0 {
		return map[string]string{}, nil
	}

	uniquePaths := make([]string, 0, len(fullCodePaths))
	seen := make(map[string]struct{}, len(fullCodePaths))
	for _, path := range fullCodePaths {
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		uniquePaths = append(uniquePaths, path)
	}
	if len(uniquePaths) == 0 {
		return map[string]string{}, nil
	}

	var rows []struct {
		FullCodePath string `gorm:"column:full_code_path"`
		Name         string `gorm:"column:name"`
	}
	if err := r.db.Table("service_tree").
		Select("full_code_path, name").
		Where("full_code_path IN ?", uniquePaths).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	result := make(map[string]string, len(rows))
	for _, row := range rows {
		if row.FullCodePath == "" || row.Name == "" {
			continue
		}
		result[row.FullCodePath] = row.Name
	}
	return result, nil
}
