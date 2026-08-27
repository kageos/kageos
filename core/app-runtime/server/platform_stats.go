package server

import (
	"github.com/kageos/kageos/core/app-runtime/model"
	"github.com/kageos/kageos/dto"
	"gorm.io/gorm"
)

func collectRuntimePlatformStats(db *gorm.DB) (dto.SystemPlatformServiceStats, error) {
	stats := dto.SystemPlatformServiceStats{}
	err := db.Model(&model.AppDatabase{}).Where("status = ?", "active").Count(&stats.AppDatabasesTotal).Error
	return stats, err
}
