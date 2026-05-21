package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

const DefaultCompanyCode = "default"

type Company struct {
	ID        int64          `json:"id" gorm:"primaryKey;autoIncrement"`
	CreatedAt models.Time    `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt models.Time    `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
	CreatedBy string         `json:"created_by" gorm:"column:created_by;type:varchar(255)"`
	Code      string         `json:"code" gorm:"column:code;type:varchar(64);uniqueIndex;not null;comment:企业唯一代码"`
	Name      string         `json:"name" gorm:"column:name;type:varchar(100);uniqueIndex;not null;comment:企业名称"`
	LogoURL   string         `json:"logo_url" gorm:"column:logo_url;type:text;comment:企业 Logo 地址或 data URL"`
}

func (Company) TableName() string {
	return "company"
}
