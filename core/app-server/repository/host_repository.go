package repository

import (
	"github.com/ai-agent-os/ai-agent-os/core/app-server/model"
	"gorm.io/gorm"
)

// HostRepository 主机数据访问层
type HostRepository struct {
	db *gorm.DB
}

// NewHostRepository 创建主机仓库
func NewHostRepository(db *gorm.DB) *HostRepository {
	return &HostRepository{
		db: db,
	}
}

// GetHostList 获取主机列表
func (r *HostRepository) GetHostList() ([]*model.Host, error) {
	var hostList []*model.Host
	err := r.db.Model(&model.Host{}).Preload("Nats").Find(&hostList).Error
	if err != nil {
		return nil, err
	}
	return hostList, nil
}

// GetHostByID 根据ID获取主机
func (r *HostRepository) GetHostByID(id int64) (*model.Host, error) {
	var host model.Host
	err := r.db.Preload("Nats").First(&host, id).Error
	if err != nil {
		return nil, err
	}
	return &host, nil
}

// CreateHost 创建主机
func (r *HostRepository) CreateHost(host *model.Host) error {
	return r.db.Create(host).Error
}

// UpdateHost 更新主机
func (r *HostRepository) UpdateHost(host *model.Host) error {
	return r.db.Save(host).Error
}

// DeleteHost 删除主机
func (r *HostRepository) DeleteHost(id int64) error {
	return r.db.Delete(&model.Host{}, id).Error
}
