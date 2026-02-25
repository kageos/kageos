package prompt

import (
	"embed"
	"encoding/json"
	"io/fs"
	"sort"
	"strings"
)

// 整个 prompt 目录下需嵌入的内容放在 content/ 下，一条 embed 直接嵌入，即可读取 content 下所有文件

//go:embed content
var promptFS embed.FS

// WorkspacePrompt 已废弃（原文件 content/doc/工作台操作提示词.md 已删除，规则迁入 system_prompt + 子文档）；保留变量供兜底分支编译通过，值为空。
var WorkspacePrompt string

// WorkspaceEnvTemplate 工作台环境模板（从 content/doc/工作台环境模板.md 加载）
var WorkspaceEnvTemplate string

// docCatalogJSON 文档目录（从 content/doc/文档目录.json 加载）
var docCatalogJSON []byte

// DocCatalogEntry 文档目录项（full_code_path 唯一定位文档，name 仅说明用途）
type DocCatalogEntry struct {
	Name         string `json:"name"`           // 文档用途说明
	FullCodePath string `json:"full_code_path"` // 文档唯一路径，如 /builtin/sdk/agent-app-sdk-readme 或 /user/app/docs/guide
	WhenToUse    string `json:"when_to_use"`    // 何时使用，注入系统消息
}

var docCatalog []DocCatalogEntry

func init() {
	if b, _ := promptFS.ReadFile("content/doc/工作台环境模板.md"); len(b) > 0 {
		WorkspaceEnvTemplate = string(b)
	}
	if b, _ := promptFS.ReadFile("content/doc/文档目录.json"); len(b) > 0 {
		docCatalogJSON = b
	}
	_ = json.Unmarshal(docCatalogJSON, &docCatalog)
}

// ReadContent 从嵌入的 content/ 下读取文件，path 为相对 content/ 的路径（如 "doc/xxx.md"）
func ReadContent(path string) ([]byte, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, nil
	}
	if !strings.HasPrefix(path, "content/") {
		path = "content/" + path
	}
	return promptFS.ReadFile(path)
}

// GetDocCatalog 返回文档目录列表（供系统消息「可用文档」块使用）
func GetDocCatalog() []DocCatalogEntry {
	return docCatalog
}

// GetBuiltinDocContent 按 full_code_path 返回内置文档正文；非内置或未命中返回空
// 若路径未在文档目录中登记但以 /builtin/doc/workspace/ 开头，则按目录读取该路径下所有 .md（便于新增 workspace 子目录时无需改 文档目录.json）
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
	// 未在目录中登记时，对 /builtin/doc/workspace/ 下路径仍按目录读（列出该目录下所有 .md 并拼接）
	if strings.HasPrefix(fullCodePath, "/builtin/doc/workspace/") {
		content = getBuiltinDocContentByEntry(&DocCatalogEntry{FullCodePath: fullCodePath})
		if content != "" {
			return fullCodePath, content
		}
	}
	return "", ""
}

func getBuiltinDocContentByEntry(e *DocCatalogEntry) string {
	fp := strings.TrimSpace(e.FullCodePath)
	// 所有 /builtin/ 路径：从 content/builtin/ 下按路径读文档；暴露路径与目录对齐，如 /builtin/doc/sdk/agent-app-sdk-readme、/builtin/doc/case_catalog/table/ticket，实际文件在 content/builtin/doc/sdk/、content/builtin/doc/case_catalog/ 下
	if !strings.HasPrefix(fp, "/builtin/") {
		return ""
	}
	rel := strings.TrimPrefix(fp, "/builtin/")
	rel = strings.Trim(rel, "/")
	if rel == "" {
		return ""
	}
	// 1）先按单文件尝试：prd.md、xxx.md、README.md（兼容 case_catalog 等单文件）
	for _, suffix := range []string{rel + "/prd.md", rel + ".md", rel + "/README.md"} {
		if b, _ := promptFS.ReadFile("content/builtin/" + suffix); len(b) > 0 {
			return string(b)
		}
	}
	// 2）再按目录读：若该路径对应目录存在且含 .md，则按文件名排序拼接所有 .md（兼容 workspace 下 01-xxx.md 等多文件）
	dirPath := "content/builtin/" + rel
	entries, err := fs.ReadDir(promptFS, dirPath)
	if err != nil || len(entries) == 0 {
		return ""
	}
	var mdNames []string
	for _, ent := range entries {
		name := ent.Name()
		if !ent.IsDir() && strings.HasSuffix(strings.ToLower(name), ".md") {
			mdNames = append(mdNames, name)
		}
	}
	if len(mdNames) == 0 {
		return ""
	}
	sort.Strings(mdNames)
	var sb strings.Builder
	for i, name := range mdNames {
		b, _ := promptFS.ReadFile(dirPath + "/" + name)
		if len(b) == 0 {
			continue
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString("## ")
		sb.WriteString(name)
		sb.WriteString("\n\n")
		sb.Write(b)
	}
	return sb.String()
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
	HubSection             string // {{HUB_SECTION}} 当前目录是否已上架到应用中心（Hub）
	DirDescription         string // {{DIR_DESCRIPTION}}
	ChildrenSection        string // {{CHILDREN_SECTION}}
	FunctionsSection       string // {{FUNCTIONS_SECTION}} 当前目录下的可执行函数（table/form/chart + full_code_path），执行模式可直接用
	FilesSection           string // {{FILES_SECTION}}
	DirectoryList          string // {{DIRECTORY_LIST}}
	InitGoSection          string // {{INIT_GO_SECTION}} 当前目录的 init_.go 内容（由 full_code_path 构造），便于模型知道已有该文件、无需再写
}

// FillWorkspaceEnvTemplate 用结构体填充工作台环境模板；占位符格式 {{KEY}}，与 WorkspaceEnvData 字段对应
func FillWorkspaceEnvTemplate(data *WorkspaceEnvData) string {
	m := map[string]string{
		"USER":                      data.User,
		"DEPARTMENT_FULL_PATH":      data.DepartmentFullPath,
		"DEPARTMENT_FULL_NAME_PATH": data.DepartmentFullNamePath,
		"CURRENT_TIME":              data.CurrentTime,
		"CURRENT_DATE":              data.CurrentDate,
		"TIMESTAMP":                 data.Timestamp,
		"DIR_NAME":                  data.DirName,
		"DIR_CODE":                  data.DirCode,
		"FULL_CODE_PATH":            data.FullCodePath,
		"DIR_TYPE":                  data.DirType,
		"HUB_SECTION":               data.HubSection,
		"DIR_DESCRIPTION":           data.DirDescription,
		"CHILDREN_SECTION":          data.ChildrenSection,
		"FUNCTIONS_SECTION":         data.FunctionsSection,
		"FILES_SECTION":             data.FilesSection,
		"DIRECTORY_LIST":            data.DirectoryList,
		"INIT_GO_SECTION":           data.InitGoSection,
	}
	s := WorkspaceEnvTemplate
	for k, v := range m {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	return s
}
