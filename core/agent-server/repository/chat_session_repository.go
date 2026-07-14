package repository

import (
	"context"

	"github.com/kageos/kageos/core/agent-server/model"
	"gorm.io/gorm"
)

type WorkspaceSessionListOptions struct {
	FullCodePath         string
	ResourceFullCodePath string
	User                 string
	SessionScope         string
	AutomationTaskID     int64
	Offset               int
	Limit                int
}

type WorkspaceAutomationAgent struct {
	TaskID    int64  `gorm:"column:automation_task_id"`
	TaskCode  string `gorm:"column:automation_task_code"`
	TaskTitle string `gorm:"column:automation_task_title"`
}

// ChatSessionRepository 工作台聊天会话数据访问层
type ChatSessionRepository struct {
	db *gorm.DB
}

// NewChatSessionRepository 创建聊天会话 Repository
func NewChatSessionRepository(db *gorm.DB) *ChatSessionRepository {
	return &ChatSessionRepository{db: db}
}

// Transaction 在同一个数据库事务内执行会话仓储操作。
func (r *ChatSessionRepository) Transaction(ctx context.Context, fn func(tx *ChatSessionRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		return fn(&ChatSessionRepository{db: db})
	})
}

// TransactionWithMessages 在同一个数据库事务内执行会话与消息仓储操作。
func (r *ChatSessionRepository) TransactionWithMessages(ctx context.Context, fn func(sessionTx *ChatSessionRepository, messageTx *ChatMessageRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		return fn(&ChatSessionRepository{db: db}, &ChatMessageRepository{db: db})
	})
}

// TransactionWithMessagesAndHandoffPackets 在同一个数据库事务内执行会话、消息和 handoff packet 仓储操作。
func (r *ChatSessionRepository) TransactionWithMessagesAndHandoffPackets(ctx context.Context, fn func(sessionTx *ChatSessionRepository, messageTx *ChatMessageRepository, handoffTx *WorkspaceHandoffPacketRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(db *gorm.DB) error {
		return fn(&ChatSessionRepository{db: db}, &ChatMessageRepository{db: db}, &WorkspaceHandoffPacketRepository{db: db})
	})
}

// Create 创建会话
func (r *ChatSessionRepository) Create(ctx context.Context, session *model.AgentChatSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

// GetBySessionID 根据 SessionID 获取会话
func (r *ChatSessionRepository) GetBySessionID(ctx context.Context, sessionID string) (*model.AgentChatSession, error) {
	var session model.AgentChatSession
	if err := r.db.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// ListByTreeID 根据 TreeID 获取会话列表
func (r *ChatSessionRepository) ListByTreeID(ctx context.Context, treeID int64, offset, limit int) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AgentChatSession{}).Where("tree_id = ?", treeID)

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
func (r *ChatSessionRepository) ListByFullCodePath(ctx context.Context, fullCodePath string, offset, limit int) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AgentChatSession{}).Where("full_code_path = ?", fullCodePath)

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
func (r *ChatSessionRepository) ListByFullCodePathAndUser(ctx context.Context, fullCodePath string, user string, offset, limit int) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AgentChatSession{}).Where("full_code_path = ? AND user = ?", fullCodePath, user)

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

func (r *ChatSessionRepository) ListWorkspaceSessions(ctx context.Context, opts WorkspaceSessionListOptions) ([]*model.AgentChatSession, int64, error) {
	var sessions []*model.AgentChatSession
	var total int64

	query := r.db.WithContext(ctx).Model(&model.AgentChatSession{})
	if opts.ResourceFullCodePath != "" {
		query = query.Where(
			"((full_code_path = ? AND resource_full_code_path = ?) OR (full_code_path = ? AND (resource_full_code_path IS NULL OR resource_full_code_path = '')))",
			opts.FullCodePath,
			opts.ResourceFullCodePath,
			opts.ResourceFullCodePath,
		)
	} else {
		// FullCodePath may itself be a plain, suffix-less function code. Match
		// both the execution directory and the concrete resource so callers do
		// not need to guess node type from the path string.
		query = query.Where("(full_code_path = ? OR resource_full_code_path = ?)", opts.FullCodePath, opts.FullCodePath)
	}
	if opts.User != "" {
		query = query.Where("user = ?", opts.User)
	}
	switch opts.SessionScope {
	case "automation":
		query = query.Where("source = ?", model.ChatSessionSourceAutomationAgent)
		if opts.AutomationTaskID > 0 {
			query = query.Where("automation_task_id = ?", opts.AutomationTaskID)
		}
	case "human":
		query = query.Where("source IS NULL OR source = '' OR source <> ?", model.ChatSessionSourceAutomationAgent)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.
		Offset(opts.Offset).
		Limit(opts.Limit).
		Order("updated_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, 0, err
	}
	return sessions, total, nil
}

func (r *ChatSessionRepository) ListWorkspaceAutomationAgents(ctx context.Context, fullCodePath string, resourceFullCodePath string, user string) ([]*WorkspaceAutomationAgent, error) {
	var agents []*WorkspaceAutomationAgent
	query := r.db.WithContext(ctx).Model(&model.AgentChatSession{}).
		Select("automation_task_id, MAX(automation_task_code) AS automation_task_code, MAX(automation_task_title) AS automation_task_title").
		Where("source = ? AND automation_task_id > 0", model.ChatSessionSourceAutomationAgent)
	if resourceFullCodePath != "" {
		query = query.Where(
			"((full_code_path = ? AND resource_full_code_path = ?) OR (full_code_path = ? AND (resource_full_code_path IS NULL OR resource_full_code_path = '')))",
			fullCodePath,
			resourceFullCodePath,
			resourceFullCodePath,
		)
	} else {
		query = query.Where("(full_code_path = ? OR resource_full_code_path = ?)", fullCodePath, fullCodePath)
	}
	if user != "" {
		query = query.Where("user = ?", user)
	}
	if err := query.
		Group("automation_task_id").
		Order("MAX(id) DESC").
		Scan(&agents).Error; err != nil {
		return nil, err
	}
	return agents, nil
}

// Update 更新会话
func (r *ChatSessionRepository) Update(ctx context.Context, session *model.AgentChatSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

// TryMarkGenerating 只在会话当前未生成时把状态标记为 generating，避免同一会话并发进入模型。
func (r *ChatSessionRepository) TryMarkGenerating(ctx context.Context, sessionID string, user string, modeCode string) (bool, error) {
	updates := map[string]interface{}{
		"status": model.ChatSessionStatusGenerating,
	}
	if user != "" {
		updates["updated_by"] = user
	}
	if modeCode != "" {
		updates["mode_code"] = modeCode
	}
	res := r.db.WithContext(ctx).Model(&model.AgentChatSession{}).
		Where("session_id = ? AND status <> ?", sessionID, model.ChatSessionStatusGenerating).
		Updates(updates)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// Delete 删除会话（根据 SessionID）
func (r *ChatSessionRepository) Delete(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Where("session_id = ?", sessionID).Delete(&model.AgentChatSession{}).Error
}

// DeleteByTreeID 根据 TreeID 删除所有会话
func (r *ChatSessionRepository) DeleteByTreeID(ctx context.Context, treeID int64) error {
	return r.db.WithContext(ctx).Where("tree_id = ?", treeID).Delete(&model.AgentChatSession{}).Error
}

// ListRunningByUser 查询指定用户所有正在执行（generating）的工作台会话
func (r *ChatSessionRepository) ListRunningByUser(ctx context.Context, user string) ([]*model.AgentChatSession, error) {
	var sessions []*model.AgentChatSession
	if err := r.db.WithContext(ctx).
		Where("user = ? AND source = ? AND status = ?", user, "workspace", model.ChatSessionStatusGenerating).
		Order("updated_at DESC").
		Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// ListFinishedByUser 查询指定用户最近非执行中的工作台会话（active/output/pending/done/cancelled）
func (r *ChatSessionRepository) ListFinishedByUser(ctx context.Context, user string, limit int) ([]*model.AgentChatSession, error) {
	var sessions []*model.AgentChatSession
	if err := r.db.WithContext(ctx).
		Where("user = ? AND source = ? AND status IN ?", user, "workspace",
			[]string{
				model.ChatSessionStatusActive,
				model.ChatSessionStatusOutput,
				model.ChatSessionStatusPendingConfirmation,
				model.ChatSessionStatusPendingTest,
				model.ChatSessionStatusPendingBuildRepair,
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
func (r *ChatSessionRepository) GetServiceTreeNamesByFullCodePaths(ctx context.Context, fullCodePaths []string) (map[string]string, error) {
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
	if err := r.db.WithContext(ctx).Table("service_tree").
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
