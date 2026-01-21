package repository

import (
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"gorm.io/gorm"
)

// FunctionGenRepository 函数生成记录数据访问层
type FunctionGenRepository struct {
	db *gorm.DB
}

// NewFunctionGenRepository 创建函数生成记录 Repository
func NewFunctionGenRepository(db *gorm.DB) *FunctionGenRepository {
	return &FunctionGenRepository{db: db}
}

// Create 创建函数生成记录
func (r *FunctionGenRepository) Create(record *model.FunctionGenRecord) error {
	return r.db.Create(record).Error
}

// GetByID 根据 ID 获取记录
func (r *FunctionGenRepository) GetByID(id int64) (*model.FunctionGenRecord, error) {
	var record model.FunctionGenRecord
	if err := r.db.Where("id = ?", id).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// GetBySessionID 根据 SessionID 获取记录（最新的）
func (r *FunctionGenRepository) GetBySessionID(sessionID string) (*model.FunctionGenRecord, error) {
	var record model.FunctionGenRecord
	if err := r.db.
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// ListByTreeID 根据 TreeID 获取记录列表
func (r *FunctionGenRepository) ListByTreeID(treeID int64, offset, limit int) ([]*model.FunctionGenRecord, int64, error) {
	var records []*model.FunctionGenRecord
	var total int64

	query := r.db.Model(&model.FunctionGenRecord{}).Where("tree_id = ?", treeID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListByAgentID 根据 AgentID 获取记录列表
func (r *FunctionGenRepository) ListByAgentID(agentID int64, offset, limit int) ([]*model.FunctionGenRecord, int64, error) {
	var records []*model.FunctionGenRecord
	var total int64

	query := r.db.Model(&model.FunctionGenRecord{}).Where("agent_id = ?", agentID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// ListByStatus 根据状态获取记录列表
func (r *FunctionGenRepository) ListByStatus(status string, offset, limit int) ([]*model.FunctionGenRecord, int64, error) {
	var records []*model.FunctionGenRecord
	var total int64

	query := r.db.Model(&model.FunctionGenRecord{}).Where("status = ?", status)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取列表
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at ASC"). // 按创建时间升序，优先处理较早的记录
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// Update 更新记录
func (r *FunctionGenRepository) Update(record *model.FunctionGenRecord) error {
	return r.db.Save(record).Error
}

// UpdateStatus 更新状态（自动计算耗时）
func (r *FunctionGenRepository) UpdateStatus(id int64, status, errorMsg string) error {
	return r.UpdateStatusWithFullCodePaths(id, status, errorMsg, nil)
}

// UpdateStatusWithFullCodePaths 更新状态和完整代码路径列表（自动计算耗时）
func (r *FunctionGenRepository) UpdateStatusWithFullCodePaths(id int64, status, errorMsg string, fullCodePaths []string) error {
	// 获取记录以计算耗时
	var record model.FunctionGenRecord
	if err := r.db.Where("id = ?", id).First(&record).Error; err != nil {
		return err
	}
	
	// 计算耗时（从创建时间到当前时间的秒数）
	duration := int(time.Since(time.Time(record.CreatedAt)).Seconds())
	
	updates := map[string]interface{}{
		"status":   status,
		"duration": duration,
	}
	if errorMsg != "" {
		updates["error_msg"] = errorMsg
	}
	if fullCodePaths != nil {
		// 转换为逗号分隔的字符串
		record.SetFullCodePaths(fullCodePaths)
		updates["full_code_paths"] = record.FullCodePaths
	}
	return r.db.Model(&model.FunctionGenRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateCode 更新代码（不更新状态，状态由回调更新）
func (r *FunctionGenRepository) UpdateCode(id int64, code string) error {
	return r.db.Model(&model.FunctionGenRecord{}).
		Where("id = ?", id).
		Update("code", code).Error
}

// UpdateCodeAndStatus 更新代码和状态（自动计算耗时，用于兼容旧代码）
func (r *FunctionGenRepository) UpdateCodeAndStatus(id int64, code string, status string) error {
	// 获取记录以计算耗时
	var record model.FunctionGenRecord
	if err := r.db.Where("id = ?", id).First(&record).Error; err != nil {
		return err
	}
	
	// 计算耗时（从创建时间到当前时间的秒数）
	duration := int(time.Since(time.Time(record.CreatedAt)).Seconds())
	
	updates := map[string]interface{}{
		"code":     code,
		"status":   status,
		"duration": duration,
	}
	return r.db.Model(&model.FunctionGenRecord{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// Delete 删除记录（根据 ID）
func (r *FunctionGenRepository) Delete(id int64) error {
	return r.db.Where("id = ?", id).Delete(&model.FunctionGenRecord{}).Error
}

// DeleteBySessionID 根据 SessionID 删除所有记录
func (r *FunctionGenRepository) DeleteBySessionID(sessionID string) error {
	return r.db.Where("session_id = ?", sessionID).Delete(&model.FunctionGenRecord{}).Error
}

