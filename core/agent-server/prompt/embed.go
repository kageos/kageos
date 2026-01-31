package prompt

import (
	"encoding/json"
	"strings"

	_ "embed"
)

// WorkspacePrompt 工作台操作提示词（从 工作台操作提示词.md 嵌入，供 buildLLMMessages 拼入 system）
//
//go:embed 工作台操作提示词.md
var WorkspacePrompt string

//go:embed 工作台环境模板.md
var WorkspaceEnvTemplate string

//go:embed 文档目录.json
var docCatalogJSON []byte

//go:embed agent-os的sdk使用文档.md
var builtinDocSDKManual string

// DocCatalogEntry 文档目录项（full_code_path 唯一定位文档，name 仅说明用途）
type DocCatalogEntry struct {
	Name         string `json:"name"`           // 文档用途说明
	FullCodePath string `json:"full_code_path"` // 文档唯一路径，如 /builtin/agent_app_sdk/docs 或 /user/app/docs/guide
	WhenToUse    string `json:"when_to_use"`    // 何时使用，注入系统消息
}

var docCatalog []DocCatalogEntry

func init() {
	_ = json.Unmarshal(docCatalogJSON, &docCatalog)
}

// GetDocCatalog 返回文档目录列表（供系统消息「可用文档」块使用）
func GetDocCatalog() []DocCatalogEntry {
	return docCatalog
}

// GetBuiltinDocContent 按 full_code_path 返回内置文档正文；非内置或未命中返回空
func GetBuiltinDocContent(fullCodePath string) (name, content string) {
	fullCodePath = strings.TrimSpace(fullCodePath)
	if fullCodePath == "" || !strings.HasPrefix(fullCodePath, "/builtin/") {
		return "", ""
	}
	for _, e := range docCatalog {
		if strings.TrimSpace(e.FullCodePath) == fullCodePath {
			return e.Name, getBuiltinDocContentByEntry(&e)
		}
	}
	return "", ""
}

func getBuiltinDocContentByEntry(e *DocCatalogEntry) string {
	if strings.TrimSpace(e.FullCodePath) == "/builtin/agent_app_sdk/docs" {
		return builtinDocSDKManual
	}
	return ""
}

// GetBuiltinDocContentByName 按文档名称返回内置文档正文；未找到返回空
func GetBuiltinDocContentByName(name string) (docName, content string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", ""
	}
	for _, e := range docCatalog {
		if strings.TrimSpace(e.Name) == name || e.Name == name {
			return e.Name, getBuiltinDocContentByEntry(&e)
		}
	}
	return "", ""
}

// WorkspaceEnvData 工作台环境模板占位数据，与 工作台环境模板.md 中的占位符一一对应
type WorkspaceEnvData struct {
	User            string // {{USER}}
	CurrentTime     string // {{CURRENT_TIME}}
	CurrentDate     string // {{CURRENT_DATE}}
	Timestamp       string // {{TIMESTAMP}}
	DirName         string // {{DIR_NAME}}
	DirCode         string // {{DIR_CODE}}
	FullCodePath    string // {{FULL_CODE_PATH}}
	DirType         string // {{DIR_TYPE}}
	DirDescription  string // {{DIR_DESCRIPTION}}
	ChildrenSection string // {{CHILDREN_SECTION}}
	FilesSection    string // {{FILES_SECTION}}
	DirectoryList   string // {{DIRECTORY_LIST}}
}

// FillWorkspaceEnvTemplate 用结构体填充工作台环境模板；占位符格式 {{KEY}}，与 WorkspaceEnvData 字段对应
func FillWorkspaceEnvTemplate(data *WorkspaceEnvData) string {
	m := map[string]string{
		"USER":             data.User,
		"CURRENT_TIME":     data.CurrentTime,
		"CURRENT_DATE":     data.CurrentDate,
		"TIMESTAMP":        data.Timestamp,
		"DIR_NAME":         data.DirName,
		"DIR_CODE":         data.DirCode,
		"FULL_CODE_PATH":   data.FullCodePath,
		"DIR_TYPE":         data.DirType,
		"DIR_DESCRIPTION":  data.DirDescription,
		"CHILDREN_SECTION": data.ChildrenSection,
		"FILES_SECTION":    data.FilesSection,
		"DIRECTORY_LIST":   data.DirectoryList,
	}
	s := WorkspaceEnvTemplate
	for k, v := range m {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}
