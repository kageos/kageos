package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

type DirectoryUpdateHistoryRepository struct {
	db *gorm.DB
}

func NewDirectoryUpdateHistoryRepository(db *gorm.DB) *DirectoryUpdateHistoryRepository {
	return &DirectoryUpdateHistoryRepository{db: db}
}

// CreateUpdateHistory 创建更新历史记录
func (r *DirectoryUpdateHistoryRepository) CreateUpdateHistory(history *model.DirectoryUpdateHistory) error {
	return r.db.Create(history).Error
}

// GetUpdateHistoryByAppVersion 获取某个版本所有目录的变更（App视角）
func (r *DirectoryUpdateHistoryRepository) GetUpdateHistoryByAppVersion(
	appID int64,
	appVersion string,
) ([]*model.DirectoryUpdateHistory, error) {
	var histories []*model.DirectoryUpdateHistory
	err := r.db.Where("app_id = ? AND app_version = ?", appID, appVersion).
		Order("dir_version_num DESC").Find(&histories).Error
	return histories, err
}

// GetAllVersionsUpdateHistory 获取应用所有版本的更新历史（App视角，返回所有版本）
func (r *DirectoryUpdateHistoryRepository) GetAllVersionsUpdateHistory(
	appID int64,
) ([]*model.DirectoryUpdateHistory, error) {
	var histories []*model.DirectoryUpdateHistory
	err := r.db.Where("app_id = ?", appID).
		Order("app_version_num DESC, dir_version_num DESC").Find(&histories).Error
	return histories, err
}

// GetUpdateHistoryByDirectory 获取目录的所有更新历史（目录视角）
func (r *DirectoryUpdateHistoryRepository) GetUpdateHistoryByDirectory(
	appID int64,
	fullCodePath string,
	limit, offset int,
) ([]*model.DirectoryUpdateHistory, int64, error) {
	var histories []*model.DirectoryUpdateHistory
	var total int64

	query := r.db.Model(&model.DirectoryUpdateHistory{}).
		Where("app_id = ? AND full_code_path = ?", appID, fullCodePath)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("dir_version_num DESC").Limit(limit).Offset(offset).Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}

// GetUpdateHistoryByDirVersion 根据目录版本号获取更新历史
func (r *DirectoryUpdateHistoryRepository) GetUpdateHistoryByDirVersion(
	appID int64,
	fullCodePath string,
	dirVersion string,
) (*model.DirectoryUpdateHistory, error) {
	var history model.DirectoryUpdateHistory
	err := r.db.Where("app_id = ? AND full_code_path = ? AND dir_version = ?",
		appID, fullCodePath, dirVersion).First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

// GetLatestUpdateHistory 获取目录的最新更新历史
func (r *DirectoryUpdateHistoryRepository) GetLatestUpdateHistory(
	appID int64,
	fullCodePath string,
) (*model.DirectoryUpdateHistory, error) {
	var history model.DirectoryUpdateHistory
	err := r.db.Where("app_id = ? AND full_code_path = ?", appID, fullCodePath).
		Order("dir_version_num DESC").First(&history).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

