package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// SourceCode 源代码表，存储函数组的源代码
type SourceCode struct {
	models.Base
	FullGroupCode string `json:"full_group_code" gorm:"type:varchar(500);uniqueIndex:idx_full_group_code;comment:完整函数组代码：{full_path}/{group_code}，与 service_tree.full_group_code 对齐"` // 完整函数组代码：{full_path}/{group_code}，与 service_tree.full_group_code 对齐
	FullPath      string `json:"full_path" gorm:"type:varchar(500);comment:完整路径，如：/luobei/testgroup"`                                                                              // 完整路径，如：/luobei/testgroup
	GroupCode     string `json:"group_code" gorm:"type:varchar(255);comment:函数组代码，如：tools_cashier"`                                                                                  // 函数组代码，如：tools_cashier
	Content       string `json:"content" gorm:"type:longtext;comment:源代码内容"`                                                                                                           // 源代码内容
	AppID         int64  `json:"app_id" gorm:"index;comment:所属应用ID"`                                                                                                                   // 所属应用 ID
	Version       string `json:"version" gorm:"type:varchar(50);comment:源代码版本（对应App.Version）"`                                                                                        // 源代码版本（对应 App.Version）
	App           *App   `json:"-" gorm:"foreignKey:AppID;references:ID"`                                                                                                               // 预加载的完整应用对象
}

func (SourceCode) TableName() string {
	return "source_code"
}

