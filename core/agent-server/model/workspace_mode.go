package model

import (
	"strings"

	"github.com/ai-agent-os/ai-agent-os/pkg/gormx/models"
	"gorm.io/gorm"
)

// ToolNamesSeparator 工具名列表存储分隔符（分号），方便查询与编辑
const ToolNamesSeparator = ";"

// WorkspaceMode 工作台模式（开发/修改/执行等，可 CRUD）
type WorkspaceMode struct {
	models.Base
	Code                 string `gorm:"type:varchar(64);uniqueIndex;not null;comment:唯一标识" json:"code"`
	Name                 string `gorm:"type:varchar(128);not null;comment:展示名" json:"name"`
	Description          string `gorm:"type:varchar(512);comment:简短说明" json:"description"`
	SystemPromptFragment string `gorm:"type:text;comment:该模式专属补充提示（拼在 Agent 提示后）" json:"system_prompt_fragment"`
	ToolNames            string `gorm:"type:varchar(1024);comment:该模式启用的工具名，分号分隔" json:"tool_names"` // 如 read_go_file;read_doc;read_dir;write_doc;write_go_file
	AgentID              *int64 `gorm:"type:bigint;index;comment:绑定智能体ID，空则用默认" json:"agent_id"`
	SortOrder            int    `gorm:"default:0;comment:排序" json:"sort_order"`
	IsBuiltin            bool   `gorm:"default:false;comment:是否内置，内置不可删除" json:"is_builtin"`
}

// TableName 指定表名
func (WorkspaceMode) TableName() string {
	return "workspace_modes"
}

// GetToolNames 解析 tool_names（分号分隔）为字符串切片；空返回 nil
func (m *WorkspaceMode) GetToolNames() []string {
	s := strings.TrimSpace(m.ToolNames)
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ToolNamesSeparator)
	names := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			names = append(names, p)
		}
	}
	return names
}

// SetToolNames 将字符串切片用分号拼接为 tool_names
func (m *WorkspaceMode) SetToolNames(names []string) {
	if len(names) == 0 {
		m.ToolNames = ""
		return
	}
	m.ToolNames = strings.Join(names, ToolNamesSeparator)
}

// InitWorkspaceModes 初始化 3 个内置模式（若不存在则插入）
func InitWorkspaceModes(db *gorm.DB) error {
	codes := []string{"dev", "modify", "execute"}
	for _, code := range codes {
		var exist WorkspaceMode
		if err := db.Where("code = ?", code).First(&exist).Error; err == nil {
			continue // 已存在
		}
		m := WorkspaceMode{
			Code:      code,
			SortOrder: 0,
			IsBuiltin: true,
		}
		switch code {
		case "dev":
			m.Name = "开发模式"
			m.Description = "生成新应用、新模块、新文件"
			m.SystemPromptFragment = "当前为开发模式，请协助用户生成新代码、新模块。"
			m.SetToolNames([]string{"read_go_file", "read_doc", "read_dir", "write_doc", "write_go_file", "build_workspace", "create_directory"})
		case "modify":
			m.Name = "修改模式"
			m.Description = "对已有应用进行修改（代码/配置）"
			m.SystemPromptFragment = "当前为修改模式，请协助用户修改已有代码或配置。"
			m.SetToolNames([]string{"read_go_file", "read_doc", "read_dir", "write_doc", "write_go_file", "build_workspace", "create_directory"})
		case "execute":
			m.Name = "执行模式"
			m.Description = "操作已生成应用（查数据、分析等）"
			m.SystemPromptFragment = "当前为执行模式，请协助用户查看数据、分析结果等。"
			m.SetToolNames([]string{"read_go_file", "read_doc", "read_dir"})
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}
