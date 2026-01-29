package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/model"
	"github.com/ai-agent-os/ai-agent-os/core/agent-server/repository"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
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

	out := make([]dto.ToolDef, 0, 2+len(plugins))

	// 1. 读取文件工具：read_file
	out = append(out, dto.ToolDef{
		Name:        "read_file",
		Description: "读取指定文件或目录下的源代码文件内容。**注意：系统消息中已经包含了当前工作目录的文件列表，但只包含文件名和行数，不包含代码内容。** 此工具用于读取具体的代码内容，用于分析代码实现、理解业务逻辑、修改代码等场景。如果提供 file_name，则只读取匹配的文件；如果不提供 file_name，则读取目录下所有文件。如果不提供 full_code_path，则使用当前工作目录。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "目录的完整路径（可选，不提供则使用当前工作目录），如 /luobei/myapp/task_management",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名（可选），支持以下格式：1) 文件名（不含扩展名），如 biz_vote_system；2) 完整文件名（含扩展名），如 biz_vote_system.go；3) 相对路径，如 vote/biz_vote_system.go。如果不提供，则读取目录下所有文件。",
				},
			},
			"required": []interface{}{},
		},
	})

	// 1.1. 读取目录工具：read_dir
	out = append(out, dto.ToolDef{
		Name:        "read_dir",
		Description: "读取指定目录下的所有子节点（子目录和函数）的信息。支持两种模式：1) 列表模式（默认）：显示当前目录的详细信息，包括子目录列表、函数列表、文件列表等；2) 树形模式（recursive=true）：递归显示所有子目录的树形结构，适合快速浏览项目整体结构。**注意：系统消息中已经包含了当前工作目录的结构信息，如果只是查看当前目录，不需要调用此工具。** 此工具主要用于：1) 查看其他目录（非当前工作目录）；2) 递归查看子目录的完整树结构（设置 recursive=true）；3) 需要包含代码内容时（设置 include_code=true）。如果不提供 full_code_path，则读取当前工作目录。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "目录的完整路径（可选，不提供则使用当前工作目录），如 /luobei/myapp/task_management",
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
					"description": "是否包含代码文件（可选，默认 false），如果需要查看代码文件，建议使用 read_file 工具",
				},
				"include_code": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码内容（可选，默认 false）。如果设置为 true，会包含所有代码文件的内容，适合需要深入理解项目、修改代码或分析业务逻辑时使用。注意：包含代码会消耗更多 token。",
				},
			},
			"required": []interface{}{},
		},
	})

	// 2. 代码生成：generate_code + write（统一流程）
	out = append(out, dto.ToolDef{
		Name:        "generate_code",
		Description: "声明将要生成一个代码文件。调用后请在本轮或下一轮输出 markdown 代码块（```go ... ```），输出完成后请调用 write_package_code 工具将代码写入文件。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "要生成的文件名（不含 .go 后缀），如 student",
				},
			},
			"required": []interface{}{"file_name"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "write_package_code",
		Description: "将「generate_code 之后」输出的代码块写入当前目录或子目录。请在输出完 markdown 代码块（```go ... ```）后调用，传入 file_name（不含 .go）。若代码中包含目录/包元数据（directory_code、file），系统会自动创建对应子目录并将文件写入该目录；否则写入当前目录。系统会从会话记录中查找「generate_code 之后」的那条 assistant 消息并解析代码块后写入。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名（不含 .go 后缀），须与 generate_code 时传入的 file_name 一致，如 student",
				},
			},
			"required": []interface{}{"file_name"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "write_doc",
		Description: "在指定路径下写入或更新文档（markdown 等）。文档不要乱放：应优先放在合适的文件夹下（如「文档」文件夹）；若没有请先 create_directory。与 create_directory 一致：树节点必有 code（URL 标识）和 name（中文描述）。写文档到文件夹下时传 full_code_path（文件夹路径）、code（如 readme）、name（如「项目文档」）、content；不传 code 则 full_code_path 视为文档完整路径（最后一段为 code，name 可选）。content 必填。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "父目录路径或文档完整路径（可选）。写文档到文件夹下时填文件夹路径并同时传 code、name；否则填文档完整路径如 /user/app/工单/文档/readme。不传则使用当前目录",
				},
				"code": map[string]interface{}{
					"type":        "string",
					"description": "文档的 code（URL 标识），如 readme、project_doc。写文档到文件夹下时必填；不填则 full_code_path 视为文档完整路径（最后一段为 code）",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "文档的 name（中文描述），如「项目文档」「README」。创建节点时用作显示名；不填则用 code",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "文档内容（必填），支持 markdown",
				},
				"format": map[string]interface{}{
					"type":        "string",
					"description": "文档格式（可选），默认 markdown",
				},
			},
			"required": []interface{}{"content"},
		},
	})
	out = append(out, dto.ToolDef{
		Name:        "create_directory",
		Description: "在当前目录或指定 full_code_path（父目录）下创建一个子目录（package 类型）。参数 name 为显示名称（如「文档」），code 为代码标识（如 docs）。description 为目录描述（可选）；admins 为管理员列表逗号分隔（可选，不填则默认为当前用户，需要帮他人加管理员时可填写）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"full_code_path": map[string]interface{}{
					"type":        "string",
					"description": "父目录完整路径（可选），不传则使用当前目录",
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
				"admins": map[string]interface{}{
					"type":        "string",
					"description": "管理员列表，逗号分隔（可选）；不填则默认为当前用户；需要为他人加管理员时可填写，如 user1,user2",
				},
			},
			"required": []interface{}{"name", "code"},
		},
	})

	// write_file 不暴露给大模型，仅内部使用

	// 2. 测试工具：get_current_time
	out = append(out, dto.ToolDef{
		Name:        "get_current_time",
		Description: "获取当前系统时间（用于测试工具调用功能）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"format": map[string]interface{}{"type": "string", "description": "时间格式，可选值：datetime（默认）、timestamp、date、time"},
			},
			"required": []interface{}{},
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
	case "read_file":
		return r.callReadFile(ctx, args, fullCodePath)
	case "read_dir":
		return r.callReadDir(ctx, args, fullCodePath)
	case "generate_code":
		return r.callGenerateCode(ctx, args)
	case "write_file":
		return r.callWriteFile(ctx, args, fullCodePath)
	case "write_package_code":
		// write_package_code 由工作台对话流程在 executeToolCalls 中处理（需 sessionID、messageRepo），不在此处执行
		return "write_package_code 应在工作台对话流程中调用，不经过 CallTool。", true
	case "write_doc":
		return RunWriteDocTool(ctx, args, fullCodePath)
	case "create_directory":
		return RunCreateDirectoryTool(ctx, args, fullCodePath)
	case "get_current_time":
		return r.callGetCurrentTime(ctx, args)
	}
	// 按插件 code 查找
	p, err := r.pluginRepo.GetByCode(name)
	if err != nil || p == nil {
		return "tool not found: " + name, true
	}
	return r.callPlugin(ctx, p, args, files)
}

// callReadFile 读取文件工具：根据 full_code_path 和 file_name 读取指定的源代码文件
func (r *ToolRegistry) callReadFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	// 获取目标目录路径（如果不提供，使用当前工作目录）
	targetPath := GetStringArg(args, "full_code_path")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}

	// 获取文件名（可选）
	fileName := GetStringArg(args, "file_name")

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
	// 获取目标目录路径（如果不提供，使用当前工作目录）
	targetPath := GetStringArg(args, "full_code_path")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}

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
				filesSection += "\n- 提示：如需查看代码内容，请使用 read_file 工具或设置 include_code=true"
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


// callGenerateCode 代码生成工具（实际处理在 executeToolCalls 中）
func (r *ToolRegistry) callGenerateCode(ctx context.Context, args map[string]interface{}) (string, bool) {
	fileName := GetStringArg(args, "file_name")
	if fileName == "" {
		return "generate_code 缺少 file_name 参数", true
	}
	// 返回确认消息，实际代码提取和处理在 executeToolCalls 中完成
	return fmt.Sprintf("已记录文件名: %s。请在 assistant 消息中输出 markdown 格式的代码块（```go ... ```），系统会自动提取并保存。", fileName), false
}

// callWriteFile 内部工具：将代码内容写入到指定文件
// 注意：此工具不暴露给大模型，仅作为内部工具使用（供服务端内部调用）
// 代码生成场景使用 generate_code 工具，服务端会自动处理文件写入
func (r *ToolRegistry) callWriteFile(ctx context.Context, args map[string]interface{}, fullCodePath string) (string, bool) {
	// 兼容旧参数名 source_code，也支持新参数名 content
	content := GetStringArg(args, "content")
	if content == "" {
		content = GetStringArg(args, "source_code") // 向后兼容
	}
	
	// 构建参数（使用 content，但后端 API 需要 source_code）
	writeArgs := make(map[string]interface{})
	for k, v := range args {
		writeArgs[k] = v
	}
	writeArgs["source_code"] = content // 后端 API 使用 source_code
	
	// 调用内部实现
	return RunAddFunctionsTool(ctx, writeArgs, fullCodePath)
}

// callGetCurrentTime 测试工具：获取当前时间
func (r *ToolRegistry) callGetCurrentTime(ctx context.Context, args map[string]interface{}) (string, bool) {
	format := GetStringArg(args, "format")
	if format == "" {
		format = "datetime"
	}

	now := time.Now()
	switch format {
	case "timestamp":
		return fmt.Sprintf("当前时间戳: %d", now.Unix()), false
	case "date":
		return fmt.Sprintf("当前日期: %s", now.Format("2006-01-02")), false
	case "time":
		return fmt.Sprintf("当前时间: %s", now.Format("15:04:05")), false
	default: // datetime
		return fmt.Sprintf("当前时间: %s", now.Format("2006-01-02 15:04:05")), false
	}
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
