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

	// 1.2 按行号读取代码文件：read_go_file_lines（带行号，便于对照编译错误）
	out = append(out, dto.ToolDef{
		Name:        "read_go_file_lines",
		Description: "按指定行号范围读取工作区内的 Go 代码文件，输出带行号，便于对照编译错误信息。参数：directory（可选）、file_name（必填）、line_ranges（可选，如 \"10-12,20-30\" 表示第 10-12 行和第 20-30 行；不传则返回整个文件并带行号）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 task.go 或 task",
				},
				"line_ranges": map[string]interface{}{
					"type":        "string",
					"description": "行号范围（可选），如 10-12,20-30 表示只返回第 10-12 行和第 20-30 行；不传则返回整个文件（带行号）",
				},
			},
			"required": []interface{}{"file_name"},
		},
	})

	// 2. 读文档：read_doc（directory 唯一定位，内置或工作区）
	out = append(out, dto.ToolDef{
		Name:        "read_doc",
		Description: "读取文档内容。传 directory 唯一定位文档（内置如 /builtin/doc/sdk/agent-app-sdk-readme，工作区如 /user/app/docs/guide）。系统消息中会列出可读文档的 directory 及名称（名称仅说明文档用途）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "文档唯一路径，如 /builtin/doc/sdk/agent-app-sdk-readme 或 /user/app/docs/guide",
				},
			},
			"required": []interface{}{"directory"},
		},
	})

	// 1.1. 读取目录工具：read_dir
	out = append(out, dto.ToolDef{
		Name:        "read_dir",
		Description: "读取指定目录下的所有子目录和文件，以树形方式展开。默认返回当前目录及其下一层的目录、函数、代码文件（tree 格式）。recursive=true 时递归显示整棵目录树；include_files 默认 true 会列出 .go 等代码文件。不传 directory 则使用当前工作目录。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选，不传则当前工作目录），如 /luobei/myapp/task_management",
				},
				"recursive": map[string]interface{}{
					"type":        "boolean",
					"description": "是否递归显示子目录（可选，默认 false）。true 时递归显示所有子目录的树形结构，类似 tree 命令。",
				},
				"max_depth": map[string]interface{}{
					"type":        "integer",
					"description": "最大深度（可选，仅在 recursive=true 时有效，默认不限制），如 3 表示最多显示 3 层",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "输出格式（可选，默认 tree）。可选值：tree（树形格式，推荐）、list（列表格式，适合详细查看）",
				},
				"include_functions": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含函数节点（可选，默认 true），展示函数可以快速了解功能",
				},
				"include_files": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码文件（可选，默认 true），会列出目录下的 .go 等文件；设为 false 则只显示目录和函数节点",
				},
				"include_code": map[string]interface{}{
					"type":        "boolean",
					"description": "是否包含代码内容（可选，默认 false）。true 时在列表中带出文件内容，消耗更多 token。",
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

	// search_replace_file：文件内容 search-replace（不整文件覆盖，实时写盘；仅改代码不编译）
	out = append(out, dto.ToolDef{
		Name:        "search_replace_file",
		Description: "在指定目录下的 .go 文件中做「查找并替换」：只改匹配到的片段，不重写整文件。必填：directory（或当前目录）、file_name、search_string、replace_string。可选：replace_all（是否替换全部出现，默认 true）。search_string 必须与文件内容完全一致（含空格、制表符、换行），否则替换不生效；使用前建议先用 read_go_file 读取文件，从实际内容中复制要替换的原文作为 search_string。仅修改代码、不编译工作空间；若需生效改完后需调用 build_workspace。编辑文件时优先用此工具，避免整文件覆盖。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录，如 /user/app/pkg1",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 handler 或 handler.go",
				},
				"search_string": map[string]interface{}{
					"type":        "string",
					"description": "要被替换的原文（必须与文件内容完全一致，含空格/制表符/换行；建议先用 read_go_file 读取后从实际内容复制，否则空格数量不一致会导致替换失败）",
				},
				"replace_string": map[string]interface{}{
					"type":        "string",
					"description": "替换后的内容",
				},
				"replace_all": map[string]interface{}{
					"type":        "boolean",
					"description": "是否替换全部出现（可选，默认 true）",
				},
				"return_full_content": map[string]interface{}{
					"type":        "boolean",
					"description": "是否在结果中返回替换后的完整文件内容，便于确认（可选，默认 true）",
				},
			},
			"required": []interface{}{"file_name", "search_string"},
		},
	})

	// delete_file：删除目录下指定 .go 文件（删磁盘+删节点）
	out = append(out, dto.ToolDef{
		Name:        "delete_file",
		Description: "删除指定目录下的一个 .go 代码文件。必填：directory（或当前目录）、file_name。会同时删除磁盘文件和 DB 节点。不能删除 init_.go。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"directory": map[string]interface{}{
					"type":        "string",
					"description": "目录（可选），不传则当前工作目录",
				},
				"file_name": map[string]interface{}{
					"type":        "string",
					"description": "文件名，如 handler 或 handler.go",
				},
			},
			"required": []interface{}{"file_name"},
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
	case "read_go_file_lines":
		return r.callReadGoFileLines(ctx, args, fullCodePath)
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
	case "search_replace_file":
		return r.callSearchReplaceFile(ctx, args, fullCodePath)
	case "delete_file":
		return r.callDeleteFile(ctx, args, fullCodePath)
	}
	// 按插件 code 查找
	p, err := r.pluginRepo.GetByCode(name)
	if err != nil || p == nil {
		return "tool not found: " + name, true
	}
	return r.callPlugin(ctx, p, args, files)
}

// callReadGoFile 读取工作区 Go 代码文件；若传入的是文档路径则降级为用 read_doc 拉取并提示
func (r *ToolRegistry) callReadGoFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	fileName := GetStringArg(args, "file_name")

	// 降级：若 directory 是内置文档路径，用文档工具拉取内容并提示应使用 read_doc
	if strings.HasPrefix(targetPath, "/builtin/") {
		docPath := strings.TrimSpace(targetPath)
		if !strings.HasPrefix(docPath, "/") {
			docPath = "/" + docPath
		}
		docName, content := prompt.GetBuiltinDocContent(docPath)
		if content != "" {
			hint := "【提示】你当前用 read_go_file 读取的是文档路径。应使用 read_doc(directory: \"" + docPath + "\") 读取文档；已为你拉取内容，下次请用 read_doc。\n\n"
			if docName == "" {
				docName = docPath
			}
			return hint + "## " + docName + "\n\n" + content, false
		}
		return "该路径是内置文档路径，请使用 read_doc(directory: \"" + docPath + "\") 读取，不要用 read_go_file。", true
	}

	// 读代码文件时从 runtime 磁盘实时读，保证内容与当前磁盘一致（快照表可能不准）
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
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

// callReadGoFileLines 按行号范围读取工作区 Go 文件，输出带行号（便于对照编译错误）
func (r *ToolRegistry) callReadGoFileLines(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	fileName := GetStringArg(args, "file_name")
	lineRangesStr := strings.TrimSpace(GetStringArg(args, "line_ranges"))

	if fileName == "" {
		return "read_go_file_lines 需传 file_name。", true
	}

	// 降级：内置文档路径不处理
	if strings.HasPrefix(targetPath, "/builtin/") {
		return "read_go_file_lines 仅用于工作区 Go 文件，不能读内置文档路径；请用 read_doc 读取文档。", true
	}

	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return fmt.Sprintf("获取代码失败: %v", err), true
	}
	if len(workspaceCtx.Files) == 0 {
		return fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false
	}

	var matched *dto.WorkspaceContextFile
	for i := range workspaceCtx.Files {
		f := &workspaceCtx.Files[i]
		if f.FileName == fileName || f.FileName+"."+f.FileType == fileName || f.RelativePath == fileName {
			matched = f
			break
		}
	}
	if matched == nil {
		return fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
	}

	lines := strings.Split(matched.Content, "\n")
	totalLines := len(lines)
	if totalLines > 0 && lines[totalLines-1] == "" {
		totalLines--
	}

	// 解析 line_ranges：如 "10-12,20-30" -> [{10,12},{20,30}]，行号 1-based
	ranges := parseLineRanges(lineRangesStr, totalLines)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("文件 %s（目录：%s）共 %d 行\n\n", matched.RelativePath, targetPath, totalLines))

	// 行号显示宽度
	width := 1
	for n := totalLines; n >= 10; n /= 10 {
		width++
	}

	for idx, rng := range ranges {
		for i := rng.Start; i <= rng.End && i <= totalLines; i++ {
			lineContent := ""
			if i-1 < len(lines) {
				lineContent = lines[i-1]
			}
			sb.WriteString(fmt.Sprintf("%*d | %s\n", width, i, lineContent))
		}
		if len(ranges) > 1 && idx < len(ranges)-1 {
			sb.WriteString("...\n")
		}
	}

	return sb.String(), false
}

// lineRange 行号范围，1-based 包含两端
type lineRange struct{ Start, End int }

// parseLineRanges 解析 "10-12,20-30" 或 "10,20-22"；空字符串表示全文，返回 []{1, totalLines}
func parseLineRanges(s string, totalLines int) []lineRange {
	s = strings.TrimSpace(s)
	if totalLines <= 0 {
		totalLines = 1
	}
	if s == "" {
		return []lineRange{{1, totalLines}}
	}
	var out []lineRange
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "-")
		if idx < 0 {
			// 单行，如 "10"
			var n int
			if _, err := fmt.Sscanf(part, "%d", &n); err == nil && n >= 1 {
				if n > totalLines {
					n = totalLines
				}
				out = append(out, lineRange{n, n})
			}
			continue
		}
		var start, end int
		if _, err := fmt.Sscanf(part, "%d-%d", &start, &end); err == nil && start >= 1 {
			if start > totalLines {
				start = totalLines
			}
			if end < start {
				end = start
			}
			if end > totalLines {
				end = totalLines
			}
			out = append(out, lineRange{start, end})
		}
	}
	if len(out) == 0 {
		return []lineRange{{1, totalLines}}
	}
	return out
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

	// 调用 app-server 的 GetWorkspaceContext 接口获取代码文件（已废弃，走 read_go_file 时用 runtime）
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
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

	outputFormat := "tree" // 默认树形格式，便于展开查看目录和文件
	if formatArg, ok := args["output_format"].(string); ok && formatArg != "" {
		outputFormat = formatArg
	}

	includeFunctions := true // 默认包含函数
	if includeFunctionsArg, ok := args["include_functions"].(bool); ok {
		includeFunctions = includeFunctionsArg
	}

	includeFiles := true // 默认包含代码文件，与「读取该文件夹下所有文件和目录」预期一致
	if includeFilesArg, ok := args["include_files"].(bool); ok {
		includeFiles = includeFilesArg
	}

	includeCode := false // 默认不包含代码内容
	if includeCodeArg, ok := args["include_code"].(bool); ok {
		includeCode = includeCodeArg
	}

	// 需要文件列表时用 runtime 从磁盘读，否则用快照；默认读取文件
	fileSource := ""
	if includeFiles {
		fileSource = "runtime"
	}
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, fileSource)
	if err != nil {
		return fmt.Sprintf("获取目录信息失败: %v", err), true
	}

	// 树形格式：recursive=true 时整棵树，recursive=false 时只展开当前一层（max_depth=1）
	if outputFormat == "tree" {
		treeMaxDepth := maxDepth
		if !recursive {
			treeMaxDepth = 1
		}
		return r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, treeMaxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
	}
	if recursive {
		return r.buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
	}

	// 列表格式显示当前目录
	return r.buildListFormat(ctx, workspaceCtx, targetPath, includeFunctions, includeFiles, includeCode, outputFormat)
}

// callReadDoc 按文档名称从嵌入的 content/doc/文档目录 查 full_code_path 后返回内置文档正文（兼容 doc_name 调用）
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
func (r *ToolRegistry) buildRecursiveTree(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, fileSource string, outputFormat string) (string, bool) {
	// 检查深度限制
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return "", false
	}

	// 构建树形结构
	treeLines := r.buildTreeLines(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, fileSource, "")

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
func (r *ToolRegistry) buildTreeLines(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, currentPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, fileSource string, prefix string) string {
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
	// 只展示当前目录直接下的文件（RelativePath 不含 "/"），避免 app-runtime 递归返回的子目录文件被重复列在根下
	files := make([]dto.WorkspaceContextFile, 0, len(workspaceCtx.Files))
	for _, f := range workspaceCtx.Files {
		if f.RelativePath != "" && !strings.Contains(f.RelativePath, "/") {
			files = append(files, f)
		}
	}

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

		// 递归查询子目录的内容（需要文件时用 runtime 从磁盘读）
		childCtx, err := apicall.GetWorkspaceContext(ctx, dir.FullCodePath, fileSource)
		if err == nil {
			result += r.buildTreeLines(ctx, childCtx, dir.FullCodePath, currentDepth+1, maxDepth, includeFunctions, includeFiles, fileSource, nextPrefix)
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

// callSearchReplaceFile 文件 search-replace（实时写盘，不整文件覆盖）
func (r *ToolRegistry) callSearchReplaceFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	searchString := GetStringArg(args, "search_string")
	if fileName == "" {
		return "search_replace_file 缺少参数 file_name。", true
	}
	if searchString == "" {
		return "search_replace_file 缺少参数 search_string。", true
	}
	replaceString := GetStringArg(args, "replace_string")
	replaceAll := true
	if v, ok := args["replace_all"]; ok {
		if b, ok := v.(bool); ok {
			replaceAll = b
		}
	}
	returnFullContent := true
	if v, ok := args["return_full_content"]; ok {
		if b, ok := v.(bool); ok {
			returnFullContent = b
		}
	}
	req := &dto.ReplaceFileContentReq{
		FullCodePath:      targetPath,
		FileName:          fileName,
		SearchString:      searchString,
		ReplaceString:     replaceString,
		ReplaceAll:        replaceAll,
		ReturnFullContent: returnFullContent,
	}
	resp, err := apicall.ReplaceFileContent(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[SearchReplaceFile] ReplaceFileContent 失败: %v", err)
		return "search_replace_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "search_replace_file: " + resp.Message, true
	}
	msg := fmt.Sprintf("已替换: 目录=%s, 文件=%s, 替换次数=%d。修改已落盘，但未编译工作空间；若需生效请调用 build_workspace 更新工作空间。", targetPath, fileName, resp.ReplaceCount)
	if resp.FullContent != "" {
		msg += "\n\n替换后完整内容：\n```go\n" + resp.FullContent + "\n```"
	}
	return msg, false
}

// callDeleteFile 删除目录下指定 .go 文件（删磁盘+删节点）
func (r *ToolRegistry) callDeleteFile(ctx context.Context, args map[string]interface{}, currentFullCodePath string) (string, bool) {
	targetPath := getDirectory(args, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		targetPath = currentFullCodePath
	}
	if targetPath != "" && !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	fileName := strings.TrimSpace(GetStringArg(args, "file_name"))
	if fileName == "" {
		return "delete_file 缺少参数 file_name。", true
	}
	req := &dto.DeleteFileReq{
		FullCodePath: targetPath,
		FileName:     fileName,
	}
	resp, err := apicall.DeleteFile(ctx, req)
	if err != nil {
		logger.Errorf(ctx, "[DeleteFile] DeleteFile 失败: %v", err)
		return "delete_file 调用失败: " + err.Error(), true
	}
	if !resp.Success {
		return "delete_file: " + resp.Message, true
	}
	return fmt.Sprintf("已删除: 目录=%s, 文件=%s", targetPath, fileName), false
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
