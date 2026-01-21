package repository

import (
	"context"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

// HubFileSnapshotRepository Hub 文件快照仓库
type HubFileSnapshotRepository struct {
	db *gorm.DB
}

// NewHubFileSnapshotRepository 创建 Hub 文件快照仓库
func NewHubFileSnapshotRepository(db *gorm.DB) *HubFileSnapshotRepository {
	return &HubFileSnapshotRepository{db: db}
}

// Create 创建文件快照
func (r *HubFileSnapshotRepository) Create(ctx context.Context, snapshot *model.HubFileSnapshot) error {
	return r.db.WithContext(ctx).Create(snapshot).Error
}

// CreateBatch 批量创建文件快照
func (r *HubFileSnapshotRepository) CreateBatch(ctx context.Context, snapshots []*model.HubFileSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(snapshots, 100).Error
}

// GetBySnapshotID 根据快照ID获取所有文件快照
func (r *HubFileSnapshotRepository) GetBySnapshotID(ctx context.Context, snapshotID int64) ([]*model.HubFileSnapshot, error) {
	var snapshots []*model.HubFileSnapshot
	err := r.db.WithContext(ctx).
		Where("hub_snapshot_id = ?", snapshotID).
		Order("relative_path ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

// GetByHubServiceTreeID 根据服务树节点ID获取文件快照
func (r *HubFileSnapshotRepository) GetByHubServiceTreeID(ctx context.Context, snapshotID int64, hubServiceTreeID int64) ([]*model.HubFileSnapshot, error) {
	var snapshots []*model.HubFileSnapshot
	err := r.db.WithContext(ctx).
		Where("hub_snapshot_id = ? AND hub_service_tree_id = ?", snapshotID, hubServiceTreeID).
		Order("relative_path ASC").
		Find(&snapshots).Error
	if err != nil {
		return nil, err
	}
	return snapshots, nil
}

// DeleteBySnapshotID 根据快照ID删除所有文件快照
func (r *HubFileSnapshotRepository) DeleteBySnapshotID(ctx context.Context, snapshotID int64) error {
	return r.db.WithContext(ctx).
		Where("hub_snapshot_id = ?", snapshotID).
		Delete(&model.HubFileSnapshot{}).Error
}

// DeleteByHubDirectoryID 根据 Hub 目录ID删除所有文件快照（通过快照关联）
func (r *HubFileSnapshotRepository) DeleteByHubDirectoryID(ctx context.Context, hubDirectoryID int64) error {
	return r.db.WithContext(ctx).
		Where("hub_snapshot_id IN (SELECT id FROM hub_snapshots WHERE hub_directory_id = ?)", hubDirectoryID).
		Delete(&model.HubFileSnapshot{}).Error
}

