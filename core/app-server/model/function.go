package model

import (
	"encoding/json"
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

type Function struct {
	models.Base
	Request      json.RawMessage `json:"request" gorm:"type:json"`
	Response     json.RawMessage `json:"response" gorm:"type:json"`
	AppID        int64           `json:"app_id"`
	TreeID       int64           `json:"tree_id"`
	Method       string          `json:"method" gorm:"type:varchar(255);column:method"`
	Router       string          `json:"router" gorm:"type:varchar(255);column:router"`
	HasConfig    bool            `json:"has_config" gorm:"column:has_config;comment:是否存在配置"` // 是否存在配置
	CreateTables string          `json:"create_tables"`                                      //创建该api时候会自动帮忙创建这个数据库表gorm的model列表
	Callbacks    string          `json:"callbacks"`
	RenderType   string          `json:"widget"` // 渲染类型
	Async        bool            `json:"async"`  //是否异步，比较耗时的api，或者需要后台慢慢处理的api
}

func (Function) TableName() string {
	return "function"
}
