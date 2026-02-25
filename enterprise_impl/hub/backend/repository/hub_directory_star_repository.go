package repository

import (
	"context"

	"github.com/ai-agent-os/hub/backend/model"
	"gorm.io/gorm"
)

type HubDirectoryStarRepository struct {
	db *gorm.DB
}

func NewHubDirectoryStarRepository(db *gorm.DB) *HubDirectoryStarRepository {
	return &HubDirectoryStarRepository{db: db}
}

// GetStarCountByDirectoryIDs 批量查询目录的星星数，返回 map[hub_directory_id]count
func (r *HubDirectoryStarRepository) GetStarCountByDirectoryIDs(ctx context.Context, ids []int64) (map[int64]int, error) {
	if len(ids) == 0 {
		return map[int64]int{}, nil
	}
	var results []struct {
		HubDirectoryID int64
		C              int64
	}
	err := r.db.Model(&model.HubDirectoryStar{}).
		Select("hub_directory_id, count(*) as c").
		Where("hub_directory_id IN ?", ids).
		Group("hub_directory_id").
		Scan(&results).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int64]int, len(results))
	for _, row := range results {
		out[row.HubDirectoryID] = int(row.C)
	}
	return out, nil
}

// Star 为目录加星（已 star 则忽略）。返回是否新建了记录（调用方据此决定是否对目录表 star_count +1）
func (r *HubDirectoryStarRepository) Star(ctx context.Context, hubDirectoryID int64, username string) (created bool, err error) {
	var existing model.HubDirectoryStar
	err = r.db.WithContext(ctx).Where("hub_directory_id = ? AND username = ?", hubDirectoryID, username).First(&existing).Error
	if err == nil {
		return false, nil // 已存在，幂等
	}
	if err != gorm.ErrRecordNotFound {
		return false, err
	}
	err = r.db.WithContext(ctx).Create(&model.HubDirectoryStar{
		HubDirectoryID: hubDirectoryID,
		Username:       username,
	}).Error
	if err != nil {
		return false, err
	}
	return true, nil
}

// Unstar 取消星星（硬删除，否则软删除记录仍占唯一索引，再次加星会 1062 Duplicate entry）
func (r *HubDirectoryStarRepository) Unstar(ctx context.Context, hubDirectoryID int64, username string) error {
	return r.db.WithContext(ctx).Unscoped().
		Where("hub_directory_id = ? AND username = ?", hubDirectoryID, username).
		Delete(&model.HubDirectoryStar{}).Error
}

// HasStarred 当前用户是否已 star 该目录
func (r *HubDirectoryStarRepository) HasStarred(ctx context.Context, hubDirectoryID int64, username string) (bool, error) {
	var c int64
	err := r.db.Model(&model.HubDirectoryStar{}).
		Where("hub_directory_id = ? AND username = ?", hubDirectoryID, username).
		Count(&c).Error
	return c > 0, err
}
