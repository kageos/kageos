package model

import (
	"errors"
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

// InitWorkspaceModes 初始化内置模式；已存在的内置模式也会刷新，确保新工具随版本升级生效。
func InitWorkspaceModes(db *gorm.DB) error {
	codes := []string{"dev"}
	for _, code := range codes {
		m := WorkspaceMode{
			Code:      code,
			SortOrder: 0,
			IsBuiltin: true,
		}
		switch code {
		case "dev":
			m.Name = "开发模式"
			m.Description = "生成、修改、构建、验证工作台应用，也可操作已有函数"
			m.SystemPromptFragment = "当前为开发模式。先调用 change_role 选择或沿用身份；按身份文档包、当前目录和源码完成方案、实现、构建和验证。"
			m.SetToolNames([]string{"change_role", "summarize_task_state", "read_go_file", "read_go_file_lines", "read_doc", "read_dir", "web_search", "fetch_url_content", "search_tools", "write_prd", "write_doc", "write_go_file", "search_replace_file", "delete_file", "read_app_log", "build_workspace", "create_directory", "run_table_search", "run_table_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query", "run_on_select_fuzzy", "run_python"})
		}

		var exist WorkspaceMode
		err := db.Where("code = ?", code).First(&exist).Error
		if err == nil {
			exist.Name = m.Name
			exist.Description = m.Description
			exist.SystemPromptFragment = m.SystemPromptFragment
			exist.ToolNames = m.ToolNames
			exist.SortOrder = m.SortOrder
			exist.IsBuiltin = true
			if err := db.Save(&exist).Error; err != nil {
				return err
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := db.Create(&m).Error; err != nil {
			return err
		}
	}
	return nil
}
