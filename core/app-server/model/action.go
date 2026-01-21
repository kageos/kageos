package model

import (
	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// Action 权限点模型
// ⭐ 权限点用表管理，支持动态配置和扩展
type Action struct {
	models.Base
	Code         string `json:"code" gorm:"type:varchar(50);not null;uniqueIndex;comment:权限点编码（如 form:read, table:write）"`
	Name         string `json:"name" gorm:"type:varchar(100);not null;comment:权限点名称（如 表单查看、表格写入）"`
	ResourceType string `json:"resource_type" gorm:"type:varchar(20);not null;index;comment:资源类型（form、table、chart、directory、app）"`
	ActionType   string `json:"action_type" gorm:"type:varchar(20);not null;index;comment:操作类型（read、write、update、delete、admin）"`
	Description  string `json:"description" gorm:"type:varchar(500);comment:权限点描述"`
	IsSystem     bool   `json:"is_system" gorm:"type:tinyint(1);not null;default:0;comment:是否系统预设"`
	CreatedBy    string `json:"created_by" gorm:"type:varchar(100);comment:创建者用户名"`
}

func (*Action) TableName() string {
	return "action"
}

// GetCode 获取权限点编码（resource_type:action_type）
func (a *Action) GetCode() string {
	return a.Code
}

// ParseActionCode 解析权限点编码，返回资源类型和操作类型
func ParseActionCode(code string) (resourceType string, actionType string, ok bool) {
	// 格式：resource_type:action_type
	// 例如：form:read, table:write
	for i := 0; i < len(code); i++ {
		if code[i] == ':' {
			resourceType = code[:i]
			actionType = code[i+1:]
			ok = true
			return
		}
	}
	return "", "", false
}

// BuildActionCode 构建权限点编码
func BuildActionCode(resourceType string, actionType string) string {
	return resourceType + ":" + actionType
}
