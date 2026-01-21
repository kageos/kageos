package repository

import (
	"context"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

// HubSnapshotRepository Hub 快照仓库
type HubSnapshotRepository struct {
	db *gorm.DB
}

// NewHubSnapshotRepository 创建 Hub 快照仓库
func NewHubSnapshotRepository(db *gorm.DB) *HubSnapshotRepository {
	return &HubSnapshotRepository{db: db}
}

// Create 创建快照
func (r *HubSnapshotRepository) Create(ctx context.Context, snapshot *model.HubSnapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

// GetByID 根据ID获取快照
func (r *HubSnapshotRepository) GetByID(ctx context.Context, id int64) (*model.HubSnapshot, error) {
	var snapshot model.HubSnapshot
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// GetByHubDirectoryID 根据 Hub 目录ID获取所有快照
func (r *HubSnapshotRepository) GetByHubDirectoryID(ctx context.Context, hubDirectoryID int64) ([]*model.HubSnapshot, error) {
	var snapshots []*model.HubSnapshot
	err := r.db.WithContext(ctx).
		Where("hub_directory_id = ?", hubDirectoryID).
		Order("version_num DESC, created_at DESC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

// GetByVersion 根据版本号获取快照
func (r *HubSnapshotRepository) GetByVersion(ctx context.Context, hubDirectoryID int64, version string) (*model.HubSnapshot, error) {
	var snapshot model.HubSnapshot
	err := r.db.WithContext(ctx).
		Where("hub_directory_id = ? AND version = ?", hubDirectoryID, version).
		First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// GetCurrent 获取当前版本快照
func (r *HubSnapshotRepository) GetCurrent(ctx context.Context, hubDirectoryID int64) (*model.HubSnapshot, error) {
	var snapshot model.HubSnapshot
	err := r.db.WithContext(ctx).
		Where("hub_directory_id = ? AND is_current = ?", hubDirectoryID, true).
		First(&snapshot).Error
	if err != nil {
		return nil, err
	}
	return &snapshot, nil
}

// SetCurrent 设置当前版本快照
func (r *HubSnapshotRepository) SetCurrent(ctx context.Context, hubDirectoryID int64, snapshotID int64) error {
	// 先将所有快照的 is_current 设为 false
	if err := r.db.WithContext(ctx).
		Model(&model.HubSnapshot{}).
		Where("hub_directory_id = ?", hubDirectoryID).
		Update("is_current", false).Error; err != nil {
		return err
	}

	// 设置指定快照为当前版本
	return r.db.WithContext(ctx).
		Model(&model.HubSnapshot{}).
		Where("id = ?", snapshotID).
		Update("is_current", true).Error
}

// Update 更新快照
func (r *HubSnapshotRepository) Update(ctx context.Context, snapshot *model.HubSnapshot) error {
	return r.db.WithContext(ctx).Save(snapshot).Error
}

// Delete 删除快照
func (r *HubSnapshotRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&model.HubSnapshot{}, id).Error
}

// DeleteByHubDirectoryID 根据 Hub 目录ID删除所有快照
func (r *HubSnapshotRepository) DeleteByHubDirectoryID(ctx context.Context, hubDirectoryID int64) error {
	return r.db.WithContext(ctx).
		Where("hub_directory_id = ?", hubDirectoryID).
		Delete(&model.HubSnapshot{}).Error
}
