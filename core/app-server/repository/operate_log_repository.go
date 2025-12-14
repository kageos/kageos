package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// OperateLogRepository 操作日志仓库
type OperateLogRepository struct {
	db *gorm.DB
}

// GetDB 获取数据库连接（用于复杂查询）
func (r *OperateLogRepository) GetDB() *gorm.DB {
	return r.db
}

// NewOperateLogRepository 创建操作日志仓库
func NewOperateLogRepository(db *gorm.DB) *OperateLogRepository {
	return &OperateLogRepository{db: db}
}

// CreateTableOperateLog 创建 Table 操作日志
func (r *OperateLogRepository) CreateTableOperateLog(log *model.TableOperateLog) error {
	return r.db.Create(log).Error
}

// CreateFormOperateLog 创建 Form 操作日志
func (r *OperateLogRepository) CreateFormOperateLog(log *model.FormOperateLog) error {
	return r.db.Create(log).Error
}

