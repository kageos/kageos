package prompt

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WorkspaceEnvInput 构建环境数据所需的输入，调用方从 workspaceCtx 等填充后传入；nil 表示无上下文，仅用 directoryName/fullCodePath 做降级
type WorkspaceEnvInput struct {
	User                   string
	DepartmentFullPath     string // 当前用户部门完整路径（存储/逻辑用，英文 code 路径）
	DepartmentFullNamePath string // 当前用户部门中文名称路径（仅展示用）
	DirName                string
	DirCode                string
	FullCodePath           string
	DirType                string
	DirDescription         string
	PublishedToHub         bool   // 当前目录是否已上架到应用中心（Hub）
	HubFullCodePath        string // 已上架时在 Hub 的目录路径
	Children               []WorkspaceEnvNode
	Files                  []WorkspaceEnvFile
}

// WorkspaceEnvNode 环境子节点（目录或函数）
type WorkspaceEnvNode struct {
	Name         string
	Code         string
	Description  string
	Type         string
	FullCodePath string // 完整路径（执行模式 run_table_search/run_form_submit/run_chart_query 用）
	TemplateType string // 函数类型（仅 function 有效）：table、form、chart
	Callbacks    string // 函数回调能力（仅 function 有效），逗号分隔
}

// WorkspaceEnvFile 环境中的代码文件
type WorkspaceEnvFile struct {
	RelativePath string
	FileType     string
	LineCount    int
}

// BuildInitGoContent 根据 full_code_path 构造该目录下 init_.go 的完整真实代码（与 app-runtime 生成逻辑一致：RouterGroup、Name、Desc 三字段完整，不省略、不用 "..." 占位）
// name、desc 可选；为空时 name 用路径最后一段，desc 用空字符串。单段路径（如仅 code）时也生成有效 init_.go。
func BuildInitGoContent(fullCodePath string, name, desc string) string {
	fullCodePath = strings.TrimSpace(strings.Trim(fullCodePath, "/"))
	if fullCodePath == "" {
		return ""
	}
	parts := strings.Split(fullCodePath, "/")
	pkg := parts[len(parts)-1]
	routerGroup := "/" + fullCodePath // 直接用完整路径，不去掉前两段
	if name == "" {
		name = pkg
	}
	return fmt.Sprintf(`package %s

import (
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app"
)

var packageContext = &app.PackageContext{
	RouterGroup: %s,
	Name:        %s,
	Desc:        %s,
}
`, pkg, strconv.Quote(routerGroup), strconv.Quote(name), strconv.Quote(desc))
}

// BuildWorkspaceEnvData 根据输入构建环境占位数据；in 为 nil 时用 directoryName/fullCodePath 填最小集，内部实现 ChildrenSection、FilesSection、DirectoryList 等。
// 约定：所有构造内容均为完整输出，不截断、不省略（ChildrenSection/FilesSection/DirectoryList/InitGoSection 等）。
func BuildWorkspaceEnvData(in *WorkspaceEnvInput, directoryName, fullCodePath string, now time.Time) *WorkspaceEnvData {
	data := &WorkspaceEnvData{
		CurrentTime:      now.Format("2006-01-02 15:04:05"),
		CurrentDate:      now.Format("2006-01-02"),
		Timestamp:        fmt.Sprintf("%d", now.Unix()),
		DirName:          directoryName,
		FullCodePath:     fullCodePath,
		ChildrenSection:  "当前目录下没有子节点。",
		FunctionsSection: "",
	}
	if in != nil {
		data.User = in.User
		data.DepartmentFullPath = in.DepartmentFullPath
		data.DepartmentFullNamePath = in.DepartmentFullNamePath
		data.DirCode = in.DirCode
		data.DirType = in.DirType
		data.DirDescription = in.DirDescription
		if in.PublishedToHub && in.HubFullCodePath != "" {
			data.HubSection = fmt.Sprintf("已上架，路径：%s（可使用 push_to_hub 推送更新）", in.HubFullCodePath)
		} else {
			data.HubSection = "未上架（可使用 publish_to_hub 发布到应用中心）"
		}
		data.ChildrenSection = buildChildrenSection(in.Children)
		data.FunctionsSection = buildFunctionsSection(in.Children)
		data.FilesSection = buildFilesSection(in.Files)
	} else {
		data.HubSection = "未知（需进入工作目录后刷新环境）"
	}
	data.DirectoryList = buildDirectoryList(GetDocCatalog())
	// 当前目录的 init_.go 完整内容（由 full_code_path + 目录名/描述构造，与 app-runtime 生成一致），便于模型知道已有该文件、无需再写
	name, desc := "", ""
	if in != nil {
		name, desc = in.DirName, in.DirDescription
	} else {
		name = directoryName
	}
	// InitGoSection：注入 init_.go 的完整内容，不省略，便于模型直接看到已有该文件
	if initGo := BuildInitGoContent(fullCodePath, name, desc); initGo != "" {
		data.InitGoSection = "### 当前目录的 init_.go（已由脚手架生成，可直接使用）\n\n```go\n" + initGo + "\n```"
	}
	return data
}

// buildChildrenSection 输出当前目录下全部子节点，完整列表不省略
func buildChildrenSection(children []WorkspaceEnvNode) string {
	if len(children) == 0 {
		return "当前目录下没有子节点。"
	}
	var packages, functions []WorkspaceEnvNode
	for _, c := range children {
		if c.Type == "package" {
			packages = append(packages, c)
		} else if c.Type == "function" {
			functions = append(functions, c)
		}
	}
	var b strings.Builder
	if len(packages) > 0 {
		b.WriteString("\n**子目录：**\n")
		for _, p := range packages {
			b.WriteString(fmt.Sprintf("- %s（%s）", p.Name, p.Code))
			if p.Description != "" {
				b.WriteString(fmt.Sprintf("：%s", p.Description))
			}
			b.WriteString("\n")
		}
	}
	if len(functions) > 0 {
		b.WriteString("\n**函数/文件：**\n")
		for _, f := range functions {
			tpl := f.TemplateType
			if tpl == "" {
				tpl = "function"
			}
			b.WriteString(fmt.Sprintf("- %s（%s）", f.Name, f.Code))
			if f.FullCodePath != "" {
				b.WriteString(fmt.Sprintf("：`%s` [%s]", f.FullCodePath, tpl))
			}
			if f.Description != "" {
				b.WriteString(fmt.Sprintf(" — %s", f.Description))
			}
			b.WriteString("\n")
		}
	}
	return b.String()
}

// buildFunctionsSection 输出当前目录下所有函数及其 full_code_path、template_type（table/form/chart）与能力摘要。
func buildFunctionsSection(children []WorkspaceEnvNode) string {
	var functions []WorkspaceEnvNode
	for _, c := range children {
		if c.Type == "function" && c.FullCodePath != "" {
			functions = append(functions, c)
		}
	}
	if len(functions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n**当前目录下的可执行函数（Table 默认先用 run_table_search；只有能力摘要明确支持写入时，才使用 run_table_create / run_table_update）：**\n")
	for _, f := range functions {
		tpl := f.TemplateType
		if tpl == "" {
			tpl = "function"
		}
		b.WriteString(fmt.Sprintf("- **%s** %s（%s）：`%s`\n", tpl, f.Name, f.Code, f.FullCodePath))
		if f.Description != "" {
			b.WriteString(fmt.Sprintf("  - %s\n", f.Description))
		}
		if caps := formatWorkspaceFunctionCapabilities(f.TemplateType, f.Callbacks); caps != "" {
			b.WriteString(fmt.Sprintf("  - 能力：%s\n", caps))
		}
	}
	return b.String()
}

func formatWorkspaceFunctionCapabilities(templateType, callbacks string) string {
	switch templateType {
	case "table":
		caps := []string{"查询"}
		if hasWorkspaceCallback(callbacks, "OnTableAddRow") {
			caps = append(caps, "新增")
		}
		if hasWorkspaceCallback(callbacks, "OnTableCreateInBatches") {
			caps = append(caps, "批量导入")
		}
		if hasWorkspaceCallback(callbacks, "OnTableUpdateRow") {
			caps = append(caps, "编辑")
		}
		if hasWorkspaceCallback(callbacks, "OnTableDeleteRows") {
			caps = append(caps, "删除")
		}
		if len(caps) == 1 {
			return "只读查询"
		}
		return strings.Join(caps, "、")
	case "form":
		return "表单提交"
	case "chart":
		return "图表查询"
	default:
		return ""
	}
}

func hasWorkspaceCallback(callbacks, target string) bool {
	for _, callback := range strings.Split(callbacks, ",") {
		if strings.TrimSpace(callback) == target {
			return true
		}
	}
	return false
}

// buildFilesSection 输出当前目录下全部可读文件，完整列表不省略
func buildFilesSection(files []WorkspaceEnvFile) string {
	if len(files) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\n### 当前可读代码文件（用 read_go_file 读取）\n")
	b.WriteString("以下文件可直接用 `read_go_file(directory, file_name)` 读取内容（不传 directory 则默认当前目录；file_name 可单文件如 a.go 或逗号分隔多文件如 a.go,b.go）：\n")
	for _, f := range files {
		b.WriteString(fmt.Sprintf("- %s（%s，%d 行）\n", f.RelativePath, f.FileType, f.LineCount))
	}
	return b.String()
}

// buildDirectoryList 输出文档目录完整列表，不省略
func buildDirectoryList(catalog []DocCatalogEntry) string {
	var b strings.Builder
	for _, e := range catalog {
		if strings.TrimSpace(e.FullCodePath) == "" {
			continue
		}
		b.WriteString(fmt.Sprintf("- **%s**（%s）\n", e.FullCodePath, e.Name))
	}
	return b.String()
}

// BuildWorkspaceEnvBlock 根据环境数据生成最终 env 块字符串；hasWorkspaceCtx 为 false 时返回降级文案（仅目录名+路径+可读目录）
func BuildWorkspaceEnvBlock(data *WorkspaceEnvData, hasWorkspaceCtx bool, directoryName, fullCodePath string) string {
	if hasWorkspaceCtx {
		return FillWorkspaceEnvTemplate(data)
	}
	return fmt.Sprintf(`当前工作目录：
- 目录名称：%s
- 完整路径：%s

你可以使用提供的工具来帮助用户完成任务。

---

## 可读的目录（部分）

%s

要生成系统/应用时，必须先 read_doc 拉取上述目录下的 SDK 文档，再按规范写 Go 代码；禁止用 HTML/CSS/JS、localStorage、纯前端等方案。`, directoryName, fullCodePath, data.DirectoryList)
}
