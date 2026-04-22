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

// InitWorkspaceModes 初始化 4 个内置模式；已存在的内置模式也会刷新，确保新工具随版本升级生效。
func InitWorkspaceModes(db *gorm.DB) error {
	codes := []string{"dev", "modify", "execute", "agent"}
	for _, code := range codes {
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
			m.SetToolNames([]string{"read_go_file", "read_go_file_lines", "read_doc", "read_dir", "web_search", "fetch_url_content", "write_doc", "write_go_file", "search_replace_file", "delete_file", "read_app_log", "build_workspace", "create_directory", "run_official_python"})
		case "modify":
			m.Name = "修改模式"
			m.Description = "对已有应用进行修改（代码/配置）"
			m.SystemPromptFragment = "当前为修改模式，请协助用户修改已有代码或配置。"
			m.SetToolNames([]string{"read_go_file", "read_go_file_lines", "read_doc", "read_dir", "web_search", "fetch_url_content", "write_doc", "write_go_file", "search_replace_file", "delete_file", "read_app_log", "build_workspace", "create_directory", "run_official_python"})
		case "execute":
			m.Name = "执行模式"
			m.Description = "操作已生成应用（查数据、提交表单、查图表、增删改和批量导入表格记录等）"
			m.SystemPromptFragment = "当前为执行模式，请协助用户查看数据、提交表单、查询图表、分析结果、增删改和批量导入表格记录等；不写代码、不落盘。"
			m.SetToolNames([]string{"read_go_file", "read_go_file_lines", "read_doc", "read_dir", "web_search", "fetch_url_content", "search_tools", "run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query", "run_on_select_fuzzy", "create_scheduled_task", "list_scheduled_tasks", "cancel_scheduled_task", "list_scheduled_task_executions", "run_official_python"})
		case "agent":
			m.Name = "Agent 模式"
			m.Description = "既可开发修改项目，也可执行查数据/提交表单/查图表/增删改和批量导入表格记录，无需切换模式"
			m.SystemPromptFragment = "当前为 Agent 模式，既可开发（写代码、建目录、编译），也可执行（查表、提交表单、查图表、新增/批量导入/更新/删除记录）；根据用户意图选择对应工具。"
			m.SetToolNames([]string{"read_go_file", "read_go_file_lines", "read_doc", "read_dir", "web_search", "fetch_url_content", "search_tools", "write_doc", "write_go_file", "search_replace_file", "delete_file", "read_app_log", "build_workspace", "create_directory", "run_table_search", "run_table_create", "run_table_batch_create", "run_table_update", "run_table_delete", "run_form_submit", "run_chart_query", "run_on_select_fuzzy", "create_scheduled_task", "list_scheduled_tasks", "cancel_scheduled_task", "list_scheduled_task_executions", "run_official_python"})
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
