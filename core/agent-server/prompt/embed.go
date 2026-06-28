package prompt

import (
	"embed"
)

// promptFS 嵌入本地 prompt seed；/system/prompt 运行时只从这里读取。

//go:embed system/prompt
var promptFS embed.FS

// WorkspaceEnvTemplate 工作台环境模板（从 system/prompt/doc/workspace-env-template.md 加载）
var WorkspaceEnvTemplate string

// DocCatalogEntry 文档目录项（full_code_path 唯一定位文档，name 仅说明用途）
type DocCatalogEntry struct {
	Name         string `json:"name"`           // 文档用途说明
	FullCodePath string `json:"full_code_path"` // 文档唯一路径，如 /system/prompt/sdk/agent-app-sdk-readme 或 /user/app/docs/guide
	WhenToUse    string `json:"when_to_use"`    // 何时使用，注入系统消息
}

var docCatalog []DocCatalogEntry

func init() {
	if b, _ := promptFS.ReadFile("system/prompt/doc/workspace-env-template.md"); len(b) > 0 {
		WorkspaceEnvTemplate = string(b)
	}
	docCatalog = buildPromptDocCatalogFromSeed()
}

// GetDocCatalog 返回文档目录列表（供系统消息「可用文档」块使用）
func GetDocCatalog() []DocCatalogEntry {
	return docCatalog
}

// WorkspaceEnvData 工作台环境模板占位数据，与 workspace-env-template.md 中的占位符一一对应
type WorkspaceEnvData struct {
	User                   string // {{USER}}
	DepartmentFullPath     string // {{DEPARTMENT_FULL_PATH}} 当前用户部门完整路径（存储/逻辑用，英文 code 路径）
	DepartmentFullNamePath string // {{DEPARTMENT_FULL_NAME_PATH}} 当前用户部门中文名称路径（仅展示用）
	CurrentTime            string // {{CURRENT_TIME}}
	CurrentDate            string // {{CURRENT_DATE}}
	Timestamp              string // {{TIMESTAMP}}
	DirName                string // {{DIR_NAME}}
	DirCode                string // {{DIR_CODE}}
	FullCodePath           string // {{FULL_CODE_PATH}}
	DirType                string // {{DIR_TYPE}}
	DirDescription         string // {{DIR_DESCRIPTION}}
	ChildrenSection        string // {{CHILDREN_SECTION}}
	FunctionsSection       string // {{FUNCTIONS_SECTION}} 当前目录下的可执行函数（table/form/chart + full_code_path），执行模式可直接用
	FilesSection           string // {{FILES_SECTION}}
	ScheduledTasksSection  string // {{SCHEDULED_TASKS_SECTION}} 当前目录下函数任务/Agent 任务轻量摘要，不含 Agent 任务 message/content
	DirectoryList          string // {{DIRECTORY_LIST}}
	InitGoSection          string // {{INIT_GO_SECTION}} 当前目录的 init_.go 内容（由 full_code_path 构造），便于模型知道已有该文件、无需再写
}
