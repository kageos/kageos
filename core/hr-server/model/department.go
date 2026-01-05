package model

import (
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
)

// Department 部门模型
type Department struct {
	models.Base
	Name         string `json:"name" gorm:"type:varchar(255);not null;comment:部门名称"`
	Code         string `json:"code" gorm:"type:varchar(255);not null;uniqueIndex;comment:部门编码（全局唯一）"`
	ParentID     *int64 `json:"parent_id" gorm:"default:NULL;index;comment:父部门ID（NULL表示根部门）"`
	FullCodePath string `json:"full_code_path" gorm:"type:varchar(500);uniqueIndex;comment:完整路径：/dept/subdept"`
	FullNamePath string `json:"full_name_path" gorm:"type:varchar(500);comment:完整名称路径：技术部/后端组（用于展示）"`
	Managers     string `json:"managers" gorm:"type:varchar(500);comment:部门负责人（多个用户名逗号分隔，如：zhangsan,lisi）"`
	Description      string `json:"description" gorm:"type:text;comment:部门描述"`
	Status           string `json:"status" gorm:"type:varchar(50);default:active;comment:状态：active(激活), inactive(停用)"`
	Sort             int    `json:"sort" gorm:"default:0;comment:排序（同级部门排序）"`
	IsSystemDefault  bool   `json:"is_system_default" gorm:"default:false;comment:是否为系统默认组织（不可删除）"`

	// 关联字段
	Parent   *Department `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID"`
	Children []*Department `json:"children,omitempty" gorm:"foreignKey:ParentID;references:ID"`
	// ⚠️ 注意：Users 关联已移除，因为 User 使用 DepartmentFullPath 而不是 DepartmentID
}

func (Department) TableName() string {
	return "department"
}

// IsActive 检查部门是否为激活状态
func (d *Department) IsActive() bool {
	return d.Status == "active"
}

// IsRoot 检查是否为根部门
func (d *Department) IsRoot() bool {
	return d.ParentID == nil
}

// GetManagersList 获取负责人列表（解析逗号分隔的字符串）
func (d *Department) GetManagersList() []string {
	if d.Managers == "" {
		return []string{}
	}
	managers := strings.Split(d.Managers, ",")
	result := make([]string, 0, len(managers))
	for _, m := range managers {
		m = strings.TrimSpace(m)
		if m != "" {
			result = append(result, m)
		}
	}
	return result
}

// SetManagersList 设置负责人列表（转换为逗号分隔的字符串）
func (d *Department) SetManagersList(managers []string) {
	if len(managers) == 0 {
		d.Managers = ""
		return
	}
	d.Managers = strings.Join(managers, ",")
}

// IsSystemDefaultDept 检查是否为系统默认组织
func (d *Department) IsSystemDefaultDept() bool {
	return d.IsSystemDefault
}

