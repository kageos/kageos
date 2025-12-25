package casbin

import (
	"gorm.io/gorm"
)

// CasbinRule Casbin 规则表模型
// 对应数据库中的 casbin_rule 表
type CasbinRule struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Ptype string `gorm:"type:varchar(100);not null;index"`
	V0    string `gorm:"type:varchar(100);index"`
	V1    string `gorm:"type:varchar(100);index"`
	V2    string `gorm:"type:varchar(100);index"`
	V3    string `gorm:"type:varchar(100)"`
	V4    string `gorm:"type:varchar(100)"`
	V5    string `gorm:"type:varchar(100)"`
}

// TableName 指定表名
func (CasbinRule) TableName() string {
	return "casbin_rule"
}

// InitCasbinTable 初始化 Casbin 表
// 在数据库初始化时调用，确保 casbin_rule 表存在
func InitCasbinTable(db *gorm.DB) error {
	return db.AutoMigrate(&CasbinRule{})
}

