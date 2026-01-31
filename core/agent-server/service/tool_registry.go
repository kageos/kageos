package service

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/logger"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types"
)

// ToolRegistry 工作台 MCP 形态的工具注册与调用
// list_tools：内置 + 插件；call_tool(name, args, full_code_path) 路由到对应实现
type ToolRegistry struct {
	pluginRepo *repository.PluginRepository
}

// NewToolRegistry 创建 ToolRegistry
func NewToolRegistry(pluginRepo *repository.PluginRepository) *ToolRegistry {
	return &ToolRegistry{pluginRepo: pluginRepo}
}

// ListTools 返回可用工具定义（内置 + 启用插件）。toolNames 非空时只返回 name 在列表中的工具，空则返回全部。
func (r *ToolRegistry) ListTools(ctx context.Context, toolNames []string) ([]dto.ToolDef, error) {
	enabled := true
	plugins, _, err := r.pluginRepo.List("", "", &enabled, 0, 200)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}

	out := make([]dto.ToolDef, 0, 6+len(plugins))

	// 1. 读代码文件：read_go_file（仅工作区 Go 代码）
	out = append(out, dto.ToolDef{
		Name:        "read_go_file",
		Description: "读取工作区内指定目录下的 Go 代码文件内容。参数：directory（可选，不传则当前工作目录）、file_name（可选，如 biz_vote_system 或 biz_vote_system.go；不传则返回该目录下所有代码文件）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名（可选），如 biz_vote_system 或 biz_vote_system.go；不传则返回该目录下所有代码文件",
				},
			},
			"required": []interface{}{},
		},
	})

	// 2. 读文档：read_doc（directory 唯一定位，内置或工作区）
	out = append(out, dto.ToolDef{
		Name:        "read_doc",
		Description: "读取文档内容。传 directory 唯一定位文档（内置如 /builtin/agent_app_sdk/docs，工作区如 /user/app/docs/guide）。系统消息中会列出可读文档的 directory 及名称（名称仅说明文档用途）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "文档唯一路径，如 /builtin/agent_app_sdk/docs 或 /user/app/docs/guide",
				},
			},
			"required": []interface{}{"directory"},
		},
	})

	// 1.1. 读取目录工具：read_dir
	out = append(out, dto.ToolDef{
		Name:        "read_dir",
		Description: "读取指定目录下的所有子节点（子目录和函数）的信息。支持两种模式：1) 列表模式（默认）：显示当前目录的详细信息；2) 树形模式（recursive=true）：递归显示所有子目录的树形结构。系统消息已包含当前工作目录结构，仅查看当前目录时无需调用。不传 directory 则使用当前工作目录。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选，不传则当前工作目录），如 /luobei/myapp/task_management",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "是否递归显示子目录（可选，默认 false）。如果设置为 true，会递归显示所有子目录的树形结构，类似 tree 命令。",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "最大深度（可选，仅在 recursive=true 时有效，默认不限制），如 3 表示最多显示 3 层",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "输出格式（可选，默认 list）。可选值：list（列表格式，适合详细查看）、tree（树形格式，适合快速浏览）",
				},
				"include_functions": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含函数节点（可选，默认 true），展示函数可以快速了解功能，因为函数有描述信息",
				},
				"include_files": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码文件（可选，默认 false），如果需要查看代码文件，建议使用 read_go_file 工具",
				},
				"include_code": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码内容（可选，默认 false）。如果设置为 true，会包含所有代码文件的内容，适合需要深入理解项目、修改代码或分析业务逻辑时使用。注意：包含代码会消耗更多 token。",
				},
			},
			"required": []interface{}{},
		},
	})

	out = append(out, dto.ToolDef{
		Name:        "create_directory",
		Description: "在当前目录或指定 directory（父目录）下创建一个子目录（package 类型）。必填：name（显示名称）、code（代码标识）。可选：directory（父目录）、description、tags、admins。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "父目录（可选），不传则使用当前目录",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "目录显示名称，如「文档」",
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "目录代码标识，如 docs",
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "目录描述（可选），如「存放项目文档」",
				},
				"tags": map[string]interface{}{
					"type":        "string",
					"description": "标签，逗号分隔（可选），如 api,user,management",
				},
				"admins": map[string]interface{}{
					"type":        "string",
					"description": "管理员列表，逗号分隔（可选）；不填则默认为当前用户；需要为他人加管理员时可填写，如 user1,user2",
				},
			},
			"required": []interface{}{"name", "code"},
		},
	})

	// write_doc：写文档（目录 + name + code + content）
	out = append(out, dto.ToolDef{
		Name:        "write_doc",
		Description: "在指定目录下创建或更新一篇文档。必填：name（显示名称）、code（英文标识）、content（正文）。可选：directory（父目录，不传则当前工作目录）、format（默认 markdown）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "父目录（可选），不传则当前工作目录",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "文档显示名称，如「项目说明」",
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "文档英文标识，用于路径，如 readme、api_docs",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文档正文（Markdown 等）",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "文档格式（可选），默认 markdown",
				},
			},
			"required": []interface{}{"name", "code", "content"},
		},
	})

	// write_go_file：写 Go 代码文件（file_name + content；build_workspace 预留）
	out = append(out, dto.ToolDef{
		Name:        "write_go_file",
		Description: "在当前工作目录或指定 directory 下写入一个 .go 代码文件。必填：file_name（如 attendance.go）、content（Go 源码）。可选：directory（目标目录）、build_workspace（是否立即编译，默认 true）。使用原则：若本次任务只需新增一个文件即可完成，直接写并编译即可（不传或传 true，省事）；若本次任务需要新增多个文件，则每个 write_go_file 传 build_workspace=false 仅写不编译，全部写完后调用一次 build_workspace 再编译。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 attendance.go、biz_vote_system.go",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Go 源码全文",
				},
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目标目录（可选），不传则当前工作目录",
				},
				"build_workspace": map[string]interface{}{
					"type":        "boolean",
					"description": "是否立即编译（可选，默认 true）。单文件任务用默认即可；多文件任务时传 false，全部写完后调用 build_workspace 再编译。",
				},
			},
			"required": []interface{}{"file_name", "content"},
		},
	})

	// build_workspace：编译当前工作空间（不写文件，仅触发编译并部署）；无需参数
	out = append(out, dto.ToolDef{
		Name:        "build_workspace",
		Description: "编译当前工作空间（Go 应用）。不写文件，仅基于当前已落盘的代码触发一次编译并部署。无需传参。连续写多个文件后可调用一次 build_workspace 再编译。",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
			"required":   []interface{}{},
		},
	})

	// 3. 插件：每个 Plugin 一条
	for _, p := range plugins {
		if p.FormPath == "" {
			continue
		}
		desc := p.Description
		if desc == "" {
			desc = p.Name
		}
		out = append(out, dto.ToolDef{
			Name:        p.Code,
			Description: desc,
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{"type": "string", "description": "文本说明或上下文"},
				},
			},
		})
	}

	// 按模式过滤：若指定了 toolNames，只保留 name 在列表中的工具
	if len(toolNames) > 0 {
		nameSet := make(map[string]struct{}, len(toolNames))
		for _, n := range toolNames {
			nameSet[n] = struct{}{}
		}
		filtered := make([]dto.ToolDef, 0, len(out))
		for _, t := range out {
			if _, ok := nameSet[t.Name]; ok {
				filtered = append(filtered, t)
			}
		}
		return filtered, nil
	}
	return out, nil
}

// CallTool 执行工具；full_code_path 从会话上下文传入；files 为当前用户消息附件，供插件 InputFiles，可为 nil
// 返回 content 字符串，若 isError 则表示执行失败，content 为错误信息
func (r *ToolRegistry) CallTool(ctx context.Context, name string, args map[string]interface{}, fullCodePath string, files *types.Files) (content string, isError bool) {
	switch name {
	case "read_go_file":
		return r.callReadGoFile(ctx, args, fullCodePath)
	case "read_doc":
		return r.callReadDocTool(ctx, args, fullCodePath)
	case "read_dir":
		return r.callReadDir(ctx, args, fullCodePath)
	case "write_doc":
		return r.callWriteDoc(ctx, args, fullCodePath)
	case "write_go_file":
		return r.callWriteGoFile(ctx, args, fullCodePath)
	case "create_directory":
		return RunCreateDirectoryTool(ctx, args, fullCodePath)
	case "build_workspace":
		return r.callWorkspaceBuild(ctx, args, fullCodePath)
	}
	// 按插件 code 查找
	p, err := r.pluginRepo.GetByCode(name)
	if err != nil || p == nil {
		return "tool not found: " + name, true
	}
	return r.callPlugin(ctx, p, args, files)
}

// callReadGoFile 读取工作区 Go 代码文件（不处理 /builtin/）
func (r *ToolRegistry) callReadGoFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	fileName := GetStringArg(args, "file_name")

	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath)
	if err != nil {
		return fmt.Sprintf("获取代码失败: %v", err), true
	}

	if len(workspaceCtx.Files) == 0 {
		return fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false
	}

	var matchedFiles []dto.WorkspaceContextFile
	if fileName != "" {
		for _, file := range workspaceCtx.Files {
			if file.FileName == fileName {
				matchedFiles = append(matchedFiles, file)
				continue
			}
			fullFileName := file.FileName + "." + file.FileType
			if fullFileName == fileName {
				matchedFiles = append(matchedFiles, file)
				continue
			}
			if file.RelativePath == fileName {
				matchedFiles = append(matchedFiles, file)
				continue
			}
		}
		if len(matchedFiles) == 0 {
			return fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
		}
	} else {
		matchedFiles = workspaceCtx.Files
	}

	var header string
	if fileName != "" {
		header = fmt.Sprintf("文件 %s 的内容（目录：%s）：\n\n", fileName, targetPath)
	} else {
		header = fmt.Sprintf("目录 %s 下的代码文件（共 %d 个）：\n\n", targetPath, len(matchedFiles))
	}

	var filesContent string
	for i, file := range matchedFiles {
		lineCount := file.LineCount
		if lineCount == 0 && file.Content != "" {
			lines := strings.Split(file.Content, "\n")
			lineCount = len(lines)
			if lineCount > 0 && lines[lineCount-1] == "" {
				lineCount--
			}
		}
		fileHeader := ""
		if len(matchedFiles) > 1 {
			fileHeader = fmt.Sprintf("## 文件 %d: %s\n", i+1, file.RelativePath)
		}
		filesContent += fmt.Sprintf(`%s- 文件名: %s
- 文件路径: %s
- 文件类型: %s
- 总行数: %d 行
- 内容长度: %d 字符
- 代码内容:
`+"```%s\n%s\n```\n\n", fileHeader, file.FileName, file.RelativePath, file.FileType, lineCount, file.ContentLength, file.FileType, file.Content)
	}
	return header + filesContent, false
}

// callReadDocTool 读取文档（directory 唯一定位，内置或工作区）
func (r *ToolRegistry) callReadDocTool(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fullCodePath := strings.TrimSpace(GetStringArg(args, "directory"))
	if fullCodePath == "" {
		fullCodePath = strings.TrimSpace(getDirectory(args, currentFullCodePath))
	}
	if fullCodePath == "" {
		return "read_doc 需传 directory。", true
	}
	if !strings.HasPrefix(fullCodePath, "/") {
		fullCodePath = "/" + fullCodePath
	}

	if strings.HasPrefix(fullCodePath, "/builtin/") {
		docName, content := prompt.GetBuiltinDocContent(fullCodePath)
		if content == "" {
			return fmt.Sprintf("未找到：directory=%s。请使用系统消息中列出的 directory。", fullCodePath), true
		}
		if docName == "" {
			docName = fullCodePath
		}
		return fmt.Sprintf("## %s\n\n%s", docName, content), false
	}

	doc, err := apicall.GetDoc(ctx, fullCodePath)
	if err != nil {
		return fmt.Sprintf("获取文档失败: %v", err), true
	}
	if doc == nil || doc.Content == "" {
		return fmt.Sprintf("文档《%s》无正文内容。", fullCodePath), false
	}
	return fmt.Sprintf("## %s\n\n%s", doc.Name, doc.Content), false
}

// callReadFile 已废弃，请使用 read_go_file / read_doc
func (r *ToolRegistry) callReadFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	fileName := GetStringArg(args, "file_name")

	if strings.HasPrefix(targetPath, "/builtin/") {
		return r.callReadDocTool(ctx, map[string]interface{}{"directory": targetPath}, currentFullCodePath)
	}

	// 工作区：targetPath 为目录，fileName 为文件名

	// 调用 app-server 的 GetWorkspaceContext 接口获取代码文件
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath)
	if err != nil {
		return fmt.Sprintf("获取代码失败: %v", err), true
	}

	if len(workspaceCtx.Files) == 0 {
		return fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false
	}

	// 如果指定了文件名，则只返回匹配的文件
	var matchedFiles []dto.WorkspaceContextFile
	if fileName != "" {
		// 匹配文件：支持多种格式
		// 1. 文件名（不含扩展名），如 biz_vote_system
		// 2. 完整文件名（含扩展名），如 biz_vote_system.go
		// 3. 相对路径，如 vote/biz_vote_system.go
		for _, file := range workspaceCtx.Files {
			// 匹配文件名（不含扩展名）
			if file.FileName == fileName {
				matchedFiles = append(matchedFiles, file)
				continue
			}
			// 匹配完整文件名（含扩展名）
			fullFileName := file.FileName + "." + file.FileType
			if fullFileName == fileName {
				matchedFiles = append(matchedFiles, file)
				continue
			}
			// 匹配相对路径
			if file.RelativePath == fileName {
				matchedFiles = append(matchedFiles, file)
				continue
			}
		}

		if len(matchedFiles) == 0 {
			return fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
		}
	} else {
		// 如果没有指定文件名，返回所有文件
		matchedFiles = workspaceCtx.Files
	}

	// 格式化输出代码文件列表和内容
	var header string
	if fileName != "" {
		header = fmt.Sprintf("文件 %s 的内容（目录：%s）：\n\n", fileName, targetPath)
	} else {
		header = fmt.Sprintf("目录 %s 下的代码文件（共 %d 个）：\n\n", targetPath, len(matchedFiles))
	}

	var filesContent string
	for i, file := range matchedFiles {
		// 降级处理：如果行数为0（可能是旧数据），则动态计算
		lineCount := file.LineCount
		if lineCount == 0 && file.Content != "" {
			lines := strings.Split(file.Content, "\n")
			lineCount = len(lines)
			// 如果最后一行是空行（文件末尾有换行符），不计入总行数
			if lineCount > 0 && lines[lineCount-1] == "" {
				lineCount--
			}
		}

		fileHeader := ""
		if len(matchedFiles) > 1 {
			fileHeader = fmt.Sprintf("## 文件 %d: %s\n", i+1, file.RelativePath)
		}

		filesContent += fmt.Sprintf(`%s- 文件名: %s
- 文件路径: %s
- 文件类型: %s
- 总行数: %d 行
- 内容长度: %d 字符
- 代码内容:
`+"```%s\n%s\n```\n\n", fileHeader, file.FileName, file.RelativePath, file.FileType, lineCount, file.ContentLength, file.FileType, file.Content)
	}

	return header + filesContent, false
}

// callReadDir 读取目录工具：读取指定目录下所有子节点和文件，支持列表模式和递归树形模式
func (r *ToolRegistry) callReadDir(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)

	// 获取参数
	recursive := false
	if recursiveArg, ok := args["recursive"].(bool); ok {
		recursive = recursiveArg
	}

	maxDepth := -1 // 默认不限制
	if maxDepthArg, ok := args["max_depth"].(float64); ok {
		maxDepth = int(maxDepthArg)
	}

	outputFormat := "list" // 默认列表格式
	if formatArg, ok := args["output_format"].(string); ok && formatArg != "" {
		outputFormat = formatArg
	}

	includeFunctions := true // 默认包含函数
	if includeFunctionsArg, ok := args["include_functions"].(bool); ok {
		includeFunctions = includeFunctionsArg
	}

	includeFiles := false // 默认不包含文件
	if includeFilesArg, ok := args["include_files"].(bool); ok {
		includeFiles = includeFilesArg
	}

	includeCode := false // 默认不包含代码内容
	if includeCodeArg, ok := args["include_code"].(bool); ok {
		includeCode = includeCodeArg
	}

	// 调用 app-server 的 GetWorkspaceContext 接口获取目录信息
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath)
	if err != nil {
		return fmt.Sprintf("获取目录信息失败: %v", err), true
	}

	// 如果 recursive=true，使用树形格式递归显示
	if recursive {
		return r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, outputFormat)
	}

	// 否则使用列表格式显示当前目录
	return r.buildListFormat(ctx, workspaceCtx, targetPath, includeFunctions, includeFiles, includeCode, outputFormat)
}

// callReadDoc 按文档名称从 prompt/文档目录 查 full_code_path 后返回内置文档正文（兼容 doc_name 调用）
func (r *ToolRegistry) callReadDoc(ctx context.Context, args map[string]interface{}) (string, bool) {
	docName := strings.TrimSpace(GetStringArg(args, "doc_name"))
	if docName == "" {
		return "read_file 读文档需传 doc_name（与「当前可读文档」列表中的名称一致）", true
	}
	name, content := prompt.GetBuiltinDocContentByName(docName)
	if content == "" {
		return "read_file 未找到文档：\"" + docName + "\"。请使用系统消息「当前可读文档」列表中列出的文档名称。", true
	}
	if name == "" {
		name = docName
	}
	return fmt.Sprintf("## %s\n\n%s", name, content), false
}

// buildListFormat 构建列表格式输出
func (r *ToolRegistry) buildListFormat(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, includeFunctions, includeFiles, includeCode bool, outputFormat string) (string, bool) {

	// 分类子节点
	var directories []dto.WorkspaceContextNode
	var functions []dto.WorkspaceContextNode
	for _, child := range workspaceCtx.Children {
		if child.Type == "package" || child.Type == "docs" {
			directories = append(directories, child)
		} else if child.Type == "function" && includeFunctions {
			functions = append(functions, child)
		}
	}

	// 构建目录信息部分
	dirInfo := fmt.Sprintf(`## 目录信息：%s

- 目录名称：%s
- 目录代码：%s
- 完整路径：%s`, targetPath, workspaceCtx.Directory.Name, workspaceCtx.Directory.Code, workspaceCtx.Directory.FullCodePath)

	if workspaceCtx.Directory.Description != "" {
		dirInfo += fmt.Sprintf("\n- 目录描述：%s", workspaceCtx.Directory.Description)
	}
	dirInfo += "\n\n"

	// 构建子目录部分
	dirsSection := ""
	if len(directories) > 0 {
		dirsSection = fmt.Sprintf("### 子目录（共 %d 个）\n\n", len(directories))
		for i, dir := range directories {
			dirsSection += fmt.Sprintf(`#### 目录 %d: %s
- 目录代码：%s
- 类型：%s
- 完整路径：%s`, i+1, dir.Name, dir.Code, dir.Type, dir.FullCodePath)
			if dir.Description != "" {
				dirsSection += fmt.Sprintf("\n- 描述：%s", dir.Description)
			}
			dirsSection += "\n\n"
		}
	}

	// 构建函数部分
	funcsSection := ""
	if len(functions) > 0 {
		funcsSection = fmt.Sprintf("### 函数/文件（共 %d 个）\n\n", len(functions))
		for i, fn := range functions {
			funcsSection += fmt.Sprintf(`#### 函数 %d: %s
- 函数代码：%s
- 完整路径：%s`, i+1, fn.Name, fn.Code, fn.FullCodePath)
			if fn.Description != "" {
				funcsSection += fmt.Sprintf("\n- 描述：%s", fn.Description)
			}
			funcsSection += "\n\n"
		}
	}

	// 构建文件部分
	filesSection := ""
	if includeFiles && len(workspaceCtx.Files) > 0 {
		filesSection = fmt.Sprintf("### 代码文件（共 %d 个）\n\n", len(workspaceCtx.Files))
		for i, file := range workspaceCtx.Files {
			// 降级处理：如果行数为0（可能是旧数据），则动态计算
			lineCount := file.LineCount
			if lineCount == 0 && file.Content != "" {
				lines := strings.Split(file.Content, "\n")
				lineCount = len(lines)
				if lineCount > 0 && lines[lineCount-1] == "" {
					lineCount--
				}
			}

			filesSection += fmt.Sprintf(`#### 文件 %d: %s
- 文件名：%s
- 文件类型：%s
- 总行数：%d 行
- 内容长度：%d 字符`, i+1, file.RelativePath, file.FileName, file.FileType, lineCount, file.ContentLength)

			if includeCode {
				filesSection += fmt.Sprintf("\n- 代码内容：\n```%s\n%s\n```", file.FileType, file.Content)
			} else {
				filesSection += "\n- 提示：如需查看代码内容，请使用 read_go_file 工具或设置 include_code=true"
			}
			filesSection += "\n\n"
		}
	} else if !includeFiles && len(workspaceCtx.Files) > 0 {
		filesSection = fmt.Sprintf("### 代码文件\n当前目录下有 %d 个代码文件（使用 include_files=true 查看详情）\n\n", len(workspaceCtx.Files))
	}

	// 如果没有子节点
	if len(directories) == 0 && len(functions) == 0 {
		if dirsSection == "" && funcsSection == "" {
			dirsSection = "### 子节点\n当前目录下没有子节点。\n\n"
		}
	}

	return dirInfo + dirsSection + funcsSection + filesSection, false
}

// buildRecursiveTree 构建递归树形结构
func (r *ToolRegistry) buildRecursiveTree(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, outputFormat string) (string, bool) {
	// 检查深度限制
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return "", false
	}

	// 构建树形结构
	treeLines := r.buildTreeLines(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, "")

	if outputFormat == "tree" {
		return fmt.Sprintf(`目录树：%s

%s`, targetPath, treeLines), false
	} else {
		// list 格式的递归输出（可以后续优化）
		return fmt.Sprintf(`目录树（递归）：%s

%s`, targetPath, treeLines), false
	}
}

// buildTreeLines 递归构建树形结构的字符串（不使用 strings.Builder）
func (r *ToolRegistry) buildTreeLines(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, currentPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, prefix string) string {
	// 检查深度限制
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return ""
	}

	var result string

	// 输出当前目录（根目录时显示 full-code-path）
	if currentDepth == 0 {
		result = fmt.Sprintf("%s [%s]\n", workspaceCtx.Directory.Name, workspaceCtx.Directory.FullCodePath)
	}

	// 获取当前目录的子节点
	children := workspaceCtx.Children
	files := workspaceCtx.Files

	// 处理子目录和函数
	directories := make([]dto.WorkspaceContextNode, 0)
	functions := make([]dto.WorkspaceContextNode, 0)
	for _, child := range children {
		if child.Type == "package" || child.Type == "docs" {
			directories = append(directories, child)
		} else if child.Type == "function" && includeFunctions {
			functions = append(functions, child)
		}
	}

	// 计算总项目数（用于判断是否是最后一项）
	totalItems := len(directories)
	if includeFunctions {
		totalItems += len(functions)
	}
	if includeFiles {
		totalItems += len(files)
	}

	// 输出子目录
	for i, dir := range directories {
		isLast := i == len(directories)-1 && (!includeFunctions || len(functions) == 0) && (!includeFiles || len(files) == 0)
		connector := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			nextPrefix = prefix + "    "
		}

		// 格式：code(名称-描述)[类型]
		descPart := ""
		if dir.Description != "" {
			descPart = "-" + dir.Description
		}
		result += fmt.Sprintf("%s%s%s(%s%s)[%s]\n", prefix, connector, dir.Code, dir.Name, descPart, dir.Type)

		// 递归查询子目录的内容
		childCtx, err := apicall.GetWorkspaceContext(ctx, dir.FullCodePath)
		if err == nil {
			result += r.buildTreeLines(ctx, childCtx, dir.FullCodePath, currentDepth+1, maxDepth, includeFunctions, includeFiles, nextPrefix)
		} else {
			result += fmt.Sprintf("%s    (无法获取子目录内容: %v)\n", nextPrefix, err)
		}
	}

	// 输出函数节点
	if includeFunctions && len(functions) > 0 {
		for i, fn := range functions {
			isLast := i == len(functions)-1 && (!includeFiles || len(files) == 0)
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			descPart := ""
			if fn.Description != "" {
				descPart = "-" + fn.Description
			}
			result += fmt.Sprintf("%s%s%s(%s%s)[function]\n", prefix, connector, fn.Code, fn.Name, descPart)
		}
	}

	// 处理文件
	if includeFiles {
		for i, file := range files {
			isLast := i == len(files)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			// 降级处理：如果行数为0（可能是旧数据），则动态计算
			lineCount := file.LineCount
			if lineCount == 0 && file.Content != "" {
				lines := strings.Split(file.Content, "\n")
				lineCount = len(lines)
				if lineCount > 0 && lines[lineCount-1] == "" {
					lineCount--
				}
			}

			result += fmt.Sprintf("%s%s%s.go (%d 行)\n", prefix, connector, file.FileName, lineCount)
		}
	}

	return result
}

// callWriteDoc 写文档：目录 + name + code + content，内部调用 RunWriteDocTool
func (r *ToolRegistry) callWriteDoc(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	// RunWriteDocTool 使用 full_code_path；将 directory 写入 args 供其使用
	argsWithPath := make(map[string]interface{}, len(args)+1)
	for k, v := range args {
		argsWithPath[k] = v
	}
	argsWithPath["full_code_path"] = targetPath
	return RunWriteDocTool(ctx, argsWithPath, currentFullCodePath)
}

// callWriteGoFile 写 Go 代码文件；build_workspace=false 时仅写不编译
func (r *ToolRegistry) callWriteGoFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	if fileName == "" {
		return "write_go_file 缺少参数 file_name。", true
	}
	content := GetStringArg(args, "content")
	if content == "" {
		content = GetStringArg(args, "source_code")
	}
	if content == "" {
		return "write_go_file 缺少参数 content。", true
	}
	if !strings.HasSuffix(fileName, ".go") {
		fileName = fileName + ".go"
	}
	nameWithoutExt := strings.TrimSuffix(fileName, ".go")
	if nameWithoutExt == "init_" {
		return "不允许创建该文件，由脚手架自动生成。", true
	}
	buildWorkspace := true
	if v, ok := args["build_workspace"]; ok {
		if b, ok := v.(bool); ok {
			buildWorkspace = b
		}
	}

	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	writeArgs := map[string]interface{}{
		"file_name":   fileName,
		"source_code": content,
	}
	return RunAddFunctionsTool(ctx, writeArgs, targetPath, true, buildWorkspace)
}

// callWorkspaceBuild 编译当前工作空间（不写文件，仅触发编译并部署）；从当前工作目录解析 user/app，无需参数
func (r *ToolRegistry) callWorkspaceBuild(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	dir := strings.Trim(strings.TrimSpace(currentFullCodePath), "/")
	if dir == "" {
		return "build_workspace 无法获取当前工作目录，请确保在有效的工作台会话中操作", true
	}
	parts := strings.Split(dir, "/")
	if len(parts) < 2 {
		return "build_workspace 当前目录格式应为 /user/app 或更长路径（如 /luobei/demo）", true
	}
	user, app := parts[0], parts[1]
	resp, err := apicall.UpdateAppBuild(ctx, user, app)
	if err != nil {
		logger.Errorf(ctx, "[WorkspaceBuild] UpdateAppBuild 失败: %v", err)
		return "build_workspace 调用失败: " + err.Error(), true
	}
	return fmt.Sprintf("工作空间已编译并部署: app=%s, 旧版本=%s, 新版本=%s", resp.App, resp.OldVersion, resp.NewVersion), false
}

// callWriteFile 已废弃，请使用 write_doc / write_go_file
func (r *ToolRegistry) callWriteFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	if fileName == "" {
		return "write_file 缺少参数 file_name。", true
	}
	content := GetStringArg(args, "content")
	if content == "" {
		content = GetStringArg(args, "source_code")
	}
	if content == "" {
		return "write_file 缺少参数 content。", true
	}

	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	} else if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}

	if strings.HasSuffix(fileName, ".md") {
		return r.callWriteFileDoc(ctx, fileName, content, targetPath)
	}
	if !strings.HasSuffix(fileName, ".go") {
		return "write_file 的 file_name 需带 .go（代码）或 .md（文档）后缀。", true
	}
	nameWithoutExt := strings.TrimSuffix(fileName, ".go")
	if nameWithoutExt == "init_" {
		return "不允许创建该文件，由脚手架自动生成。", true
	}
	writeArgs := map[string]interface{}{
		"file_name":   fileName,
		"source_code": content,
	}
	return RunAddFunctionsTool(ctx, writeArgs, targetPath, true, true)
}

// callWriteFileDoc 将 markdown 内容写入文档节点（write_file 且 file_name 为 .md 时）
func (r *ToolRegistry) callWriteFileDoc(ctx context.Context, fileName, content, targetPath string) (string, bool) {
	docCode := strings.TrimSuffix(filepath.Base(fileName), ".md")
	if docCode == "" {
		return "write_file 的 .md 文件名无效。", true
	}
	docPath := strings.TrimRight(targetPath, "/") + "/" + docCode
	if !strings.HasPrefix(docPath, "/") {
		docPath = "/" + docPath
	}
	detail, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, docPath)
	if err == nil && detail != nil && detail.Type == "docs" {
		format := "markdown"
		err = apicall.UpdateDocs(ctx, detail.ID, &dto.UpdateDocsReq{Content: &content, Format: &format})
		if err != nil {
			logger.Errorf(ctx, "[WriteFile] UpdateDocs 失败: %v", err)
			return "write_file 更新文档失败: " + err.Error(), true
		}
		return fmt.Sprintf("文档已更新: %s", docPath), false
	}
	parent, err := apicall.GetServiceTreeDetailByFullCodePath(ctx, targetPath)
	if err != nil || parent == nil {
		return "父目录不存在，请先 create_directory 再 write_file。", true
	}
	pathParts := strings.Split(strings.Trim(targetPath, "/"), "/")
	if len(pathParts) < 2 {
		return "write_file directory 格式无效", true
	}
	user, app := pathParts[0], pathParts[1]
	req := &dto.CreateDocsReq{
		User:     user,
		App:      app,
		Name:     docCode,
		Code:     docCode,
		ParentID: parent.ID,
		Content:  content,
		Format:   "markdown",
	}
	_, err = apicall.CreateDocs(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[WriteFile] CreateDocs 失败: %v", err)
		return "write_file 创建文档失败: " + err.Error(), true
	}
	return fmt.Sprintf("文档已创建: %s", docPath), false
}

// callPlugin 插件工具：CallFormAPI(FormPath, {Content, InputFiles})
func (r *ToolRegistry) callPlugin(ctx context.Context, p *model.Plugin, args map[string]interface{}, files *types.Files) (string, bool) {
	return RunPluginTool(ctx, p, args, files)
}

// ToToolArgs 将 interface{} 转为 map[string]interface{}，供 CallTool 使用
// JSON 反序列化后，object→map[string]interface{}；nil/null/缺省→nil，按空 map 处理
func ToToolArgs(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{}
	}
	if m, ok := v.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

// GetStringArg 从 args 取 string
func GetStringArg(args map[string]interface{}, key string) string {
	if v, ok := args[key]; ok && v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getDirectory 取目录参数：模型侧用 directory，兼容旧参数 full_code_path；未传则用 defaultPath
func getDirectory(args map[string]interface{}, defaultPath string) string {
	s := strings.TrimSpace(GetStringArg(args, "directory"))
	if s != "" {
		return s
	}
	s = strings.TrimSpace(GetStringArg(args, "full_code_path"))
	if s != "" {
		return s
	}
	return defaultPath
}
