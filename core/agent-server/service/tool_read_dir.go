package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/pkg/servicetree"
)

type ReadDirTool struct{}

type readDirArgs struct {
	Directory        string `json:"directory" schema_desc:"要读取的目录，不传则当前工作目录"`
	FullCodePath     string `json:"full_code_path" schema_ignore:"true"`
	Recursive        *bool  `json:"recursive" schema_desc:"是否递归显示子目录"`
	MaxDepth         *int   `json:"max_depth" schema_desc:"递归显示时的最大深度"`
	OutputFormat     string `json:"output_format" schema_desc:"输出格式" schema_enum:"tree,list"`
	IncludeFunctions *bool  `json:"include_functions" schema_desc:"是否包含函数节点"`
	IncludeFiles     *bool  `json:"include_files" schema_desc:"是否包含代码文件"`
	IncludeCode      *bool  `json:"include_code" schema_desc:"是否包含代码内容"`
}

type readDirResultData struct {
	RequestedPath    string               `json:"requested_path,omitempty" schema_desc:"调用时传入的目录路径"`
	ResolvedPath     string               `json:"resolved_path" schema_desc:"最终读取的目录路径" schema_required:"true"`
	DegradedFromFunc bool                 `json:"degraded_from_function" schema_desc:"是否从函数节点自动降级到父目录" schema_required:"true"`
	OutputFormat     string               `json:"output_format" schema_desc:"本次输出格式" schema_required:"true"`
	Recursive        bool                 `json:"recursive" schema_desc:"是否递归读取" schema_required:"true"`
	MaxDepth         *int                 `json:"max_depth,omitempty" schema_desc:"递归读取的最大深度"`
	IncludeFunctions bool                 `json:"include_functions" schema_desc:"是否包含函数节点" schema_required:"true"`
	IncludeFiles     bool                 `json:"include_files" schema_desc:"是否包含代码文件" schema_required:"true"`
	IncludeCode      bool                 `json:"include_code" schema_desc:"是否包含代码内容" schema_required:"true"`
	Directory        readDirDirectoryData `json:"directory" schema_desc:"当前目录信息" schema_required:"true"`
	Summary          readDirSummaryData   `json:"summary" schema_desc:"当前目录结果统计" schema_required:"true"`
	Directories      []readDirNodeData    `json:"directories,omitempty" schema_desc:"当前目录下的直接子目录列表"`
	Functions        []readDirNodeData    `json:"functions,omitempty" schema_desc:"当前目录下的直接函数节点列表"`
	Files            []readDirFileData    `json:"files,omitempty" schema_desc:"当前目录下返回的代码文件列表"`
}

type readDirDirectoryData struct {
	Name         string `json:"name" schema_desc:"目录名称" schema_required:"true"`
	Code         string `json:"code" schema_desc:"目录英文标识" schema_required:"true"`
	FullCodePath string `json:"full_code_path" schema_desc:"目录完整路径" schema_required:"true"`
	Description  string `json:"description,omitempty" schema_desc:"目录描述"`
	Type         string `json:"type" schema_desc:"目录类型" schema_required:"true"`
}

type readDirSummaryData struct {
	DirectoryCount int `json:"directory_count" schema_desc:"子目录数量" schema_required:"true"`
	FunctionCount  int `json:"function_count" schema_desc:"函数节点数量" schema_required:"true"`
	FileCount      int `json:"file_count" schema_desc:"代码文件数量" schema_required:"true"`
}

type readDirNodeData struct {
	Name         string `json:"name" schema_desc:"节点名称" schema_required:"true"`
	Code         string `json:"code" schema_desc:"节点代码" schema_required:"true"`
	Type         string `json:"type" schema_desc:"节点类型" schema_required:"true"`
	Description  string `json:"description,omitempty" schema_desc:"节点描述"`
	FullCodePath string `json:"full_code_path" schema_desc:"节点完整路径" schema_required:"true"`
	TemplateType string `json:"template_type,omitempty" schema_desc:"函数模板类型"`
}

type readDirFileData struct {
	FileName      string `json:"file_name" schema_desc:"文件名" schema_required:"true"`
	RelativePath  string `json:"relative_path" schema_desc:"相对路径" schema_required:"true"`
	FullPath      string `json:"full_path" schema_desc:"完整文件路径" schema_required:"true"`
	FileType      string `json:"file_type" schema_desc:"文件类型" schema_required:"true"`
	LineCount     int    `json:"line_count" schema_desc:"代码总行数" schema_required:"true"`
	ContentLength int    `json:"content_length" schema_desc:"内容长度" schema_required:"true"`
	Content       string `json:"content,omitempty" schema_desc:"代码内容"`
}

var readDirToolDef = toolDefinitionWithOutput[readDirArgs, structuredToolResultSchema[readDirResultData]](
	"read_dir",
	"读取指定目录下的所有子目录和文件，以树形方式展开。默认返回当前目录及其下一层的目录、函数、代码文件（tree 格式）。recursive=true 时递归显示整棵目录树；include_files 默认 true 会列出 .go 等代码文件。不传 directory 则使用当前工作目录。",
)

func (t *ReadDirTool) Definition() dto.ToolDef {
	return readDirToolDef
}

func (t *ReadDirTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readDirArgs](call.Args)
	if err != nil {
		return toolResult("read_dir 参数解析失败: "+err.Error(), true)
	}
	return runReadDirTool(ctx, args, call.FullCodePath)
}

// runReadDirTool 读取指定目录下所有子节点和文件，支持列表模式和递归树形模式
func runReadDirTool(ctx context.Context, args readDirArgs, currentFullCodePath string) ToolResult {
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)

	degraded := false
	originalPath := targetPath
	if isFunctionPath(targetPath) {
		if parentPath := getParentPath(targetPath); parentPath != "" {
			targetPath = parentPath
			degraded = true
		}
	}

	recursive := false
	if args.Recursive != nil {
		recursive = *args.Recursive
	}

	maxDepth := -1
	if args.MaxDepth != nil {
		maxDepth = *args.MaxDepth
	}

	outputFormat := "tree"
	if formatArg := strings.TrimSpace(args.OutputFormat); formatArg != "" {
		outputFormat = formatArg
	}

	includeFunctions := true
	if args.IncludeFunctions != nil {
		includeFunctions = *args.IncludeFunctions
	}

	includeFiles := true
	if args.IncludeFiles != nil {
		includeFiles = *args.IncludeFiles
	}

	includeCode := false
	if args.IncludeCode != nil {
		includeCode = *args.IncludeCode
	}

	fileSource := ""
	if includeFiles {
		fileSource = "runtime"
	}
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, fileSource)
	if err != nil {
		return toolResult(fmt.Sprintf("获取目录信息失败: %v", err), true)
	}

	degradeNotice := ""
	if degraded {
		degradeNotice = fmt.Sprintf("> 注意：`%s` 是一个函数节点（非目录），已自动读取其所在的父目录 `%s`。\n\n", originalPath, targetPath)
	}
	resultData := buildReadDirResultData(originalPath, targetPath, degraded, outputFormat, recursive, maxDepth, includeFunctions, includeFiles, includeCode, workspaceCtx)

	if outputFormat == "tree" {
		treeMaxDepth := maxDepth
		if !recursive {
			treeMaxDepth = 1
		}
		result, hasErr := buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, treeMaxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
		return toolResultWithData(degradeNotice+result, hasErr, resultData)
	}
	if recursive {
		result, hasErr := buildRecursiveTree(ctx, workspaceCtx, targetPath, 0, maxDepth, includeFunctions, includeFiles, fileSource, outputFormat)
		return toolResultWithData(degradeNotice+result, hasErr, resultData)
	}

	result, hasErr := buildListFormat(workspaceCtx, targetPath, includeFunctions, includeFiles, includeCode)
	return toolResultWithData(degradeNotice+result, hasErr, resultData)
}

func buildListFormat(workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, includeFunctions, includeFiles, includeCode bool) (string, bool) {
	var directories []dto.WorkspaceContextNode
	var functions []dto.WorkspaceContextNode
	for _, child := range workspaceCtx.Children {
		if child.Type == servicetree.TypePackage || child.Type == servicetree.TypeDocs {
			directories = append(directories, child)
		} else if child.Type == servicetree.TypeFunction && includeFunctions {
			functions = append(functions, child)
		}
	}

	dirInfo := fmt.Sprintf(`## 目录信息：%s

- 目录名称：%s
- 目录英文标识：%s
- 完整路径：%s`, targetPath, workspaceCtx.Directory.Name, workspaceCtx.Directory.Code, workspaceCtx.Directory.FullCodePath)

	if workspaceCtx.Directory.Description != "" {
		dirInfo += fmt.Sprintf("\n- 目录描述：%s", workspaceCtx.Directory.Description)
	}
	dirInfo += "\n\n"

	dirsSection := ""
	if len(directories) > 0 {
		dirsSection = fmt.Sprintf("### 子目录（共 %d 个）\n\n", len(directories))
		for i, dir := range directories {
			dirsSection += fmt.Sprintf(`#### 目录 %d: %s
- 目录英文标识：%s
- 类型：%s
- 完整路径：%s`, i+1, dir.Name, dir.Code, dir.Type, dir.FullCodePath)
			if dir.Description != "" {
				dirsSection += fmt.Sprintf("\n- 描述：%s", dir.Description)
			}
			dirsSection += "\n\n"
		}
	}

	funcsSection := ""
	if len(functions) > 0 {
		funcsSection = fmt.Sprintf("### 函数/文件（共 %d 个）\n\n", len(functions))
		for i, fn := range functions {
			tpl := fn.TemplateType
			if tpl == "" {
				tpl = servicetree.TypeFunction
			}
			funcsSection += fmt.Sprintf(`#### 函数 %d: %s
- 函数代码：%s
- 类型：%s
- 完整路径：%s`, i+1, fn.Name, fn.Code, tpl, fn.FullCodePath)
			if fn.Description != "" {
				funcsSection += fmt.Sprintf("\n- 描述：%s", fn.Description)
			}
			funcsSection += "\n\n"
		}
	}

	filesSection := ""
	if includeFiles && len(workspaceCtx.Files) > 0 {
		filesSection = fmt.Sprintf("### 代码文件（共 %d 个）\n\n", len(workspaceCtx.Files))
		for i, file := range workspaceCtx.Files {
			lineCount := workspaceFileLineCount(file)

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

	if len(directories) == 0 && len(functions) == 0 {
		if dirsSection == "" && funcsSection == "" {
			dirsSection = "### 子节点\n当前目录下没有子节点。\n\n"
		}
	}

	return dirInfo + dirsSection + funcsSection + filesSection, false
}

func buildRecursiveTree(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, targetPath string, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, fileSource string, outputFormat string) (string, bool) {
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return "", false
	}

	treeLines := buildTreeLines(ctx, workspaceCtx, currentDepth, maxDepth, includeFunctions, includeFiles, fileSource, "")
	if outputFormat == "tree" {
		return fmt.Sprintf(`目录树：%s

%s`, targetPath, treeLines), false
	}
	return fmt.Sprintf(`目录树（递归）：%s

%s`, targetPath, treeLines), false
}

func buildTreeLines(ctx context.Context, workspaceCtx *dto.GetWorkspaceContextResp, currentDepth int, maxDepth int, includeFunctions bool, includeFiles bool, fileSource string, prefix string) string {
	if maxDepth >= 0 && currentDepth >= maxDepth {
		return ""
	}

	var result string
	if currentDepth == 0 {
		result = fmt.Sprintf("%s [%s]\n", workspaceCtx.Directory.Name, workspaceCtx.Directory.FullCodePath)
	}

	children := workspaceCtx.Children
	files := make([]dto.WorkspaceContextFile, 0, len(workspaceCtx.Files))
	for _, f := range workspaceCtx.Files {
		if f.RelativePath != "" && !strings.Contains(f.RelativePath, "/") {
			files = append(files, f)
		}
	}

	directories := make([]dto.WorkspaceContextNode, 0)
	functions := make([]dto.WorkspaceContextNode, 0)
	for _, child := range children {
		if child.Type == servicetree.TypePackage || child.Type == servicetree.TypeDocs {
			directories = append(directories, child)
		} else if child.Type == servicetree.TypeFunction && includeFunctions {
			functions = append(functions, child)
		}
	}

	for i, dir := range directories {
		isLast := i == len(directories)-1 && (!includeFunctions || len(functions) == 0) && (!includeFiles || len(files) == 0)
		connector := "├── "
		nextPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			nextPrefix = prefix + "    "
		}

		descPart := ""
		if dir.Description != "" {
			descPart = "-" + dir.Description
		}
		result += fmt.Sprintf("%s%s%s(%s%s)[%s]\n", prefix, connector, dir.Code, dir.Name, descPart, dir.Type)

		childCtx, err := apicall.GetWorkspaceContext(ctx, dir.FullCodePath, fileSource)
		if err == nil {
			result += buildTreeLines(ctx, childCtx, currentDepth+1, maxDepth, includeFunctions, includeFiles, fileSource, nextPrefix)
		} else {
			result += fmt.Sprintf("%s    (无法获取子目录内容: %v)\n", nextPrefix, err)
		}
	}

	if includeFunctions && len(functions) > 0 {
		for i, fn := range functions {
			isLast := i == len(functions)-1 && (!includeFiles || len(files) == 0)
			connector := "├── "
			if isLast {
				connector = "└── "
			}
			tpl := fn.TemplateType
			if tpl == "" {
				tpl = servicetree.TypeFunction
			}
			descPart := ""
			if fn.Description != "" {
				descPart = "-" + fn.Description
			}
			result += fmt.Sprintf("%s%s%s(%s%s)[%s]", prefix, connector, fn.Code, fn.Name, descPart, tpl)
			if fn.FullCodePath != "" {
				result += fmt.Sprintf(" → %s", fn.FullCodePath)
			}
			result += "\n"
		}
	}

	if includeFiles {
		for i, file := range files {
			isLast := i == len(files)-1
			connector := "├── "
			if isLast {
				connector = "└── "
			}

			lineCount := workspaceFileLineCount(file)

			result += fmt.Sprintf("%s%s%s.go (%d 行)\n", prefix, connector, file.FileName, lineCount)
		}
	}

	return result
}

func buildReadDirResultData(originalPath string, targetPath string, degraded bool, outputFormat string, recursive bool, maxDepth int, includeFunctions bool, includeFiles bool, includeCode bool, workspaceCtx *dto.GetWorkspaceContextResp) *readDirResultData {
	if workspaceCtx == nil {
		return nil
	}

	data := &readDirResultData{
		RequestedPath:    originalPath,
		ResolvedPath:     targetPath,
		DegradedFromFunc: degraded,
		OutputFormat:     outputFormat,
		Recursive:        recursive,
		IncludeFunctions: includeFunctions,
		IncludeFiles:     includeFiles,
		IncludeCode:      includeCode,
		Directory: readDirDirectoryData{
			Name:         workspaceCtx.Directory.Name,
			Code:         workspaceCtx.Directory.Code,
			FullCodePath: workspaceCtx.Directory.FullCodePath,
			Description:  workspaceCtx.Directory.Description,
			Type:         workspaceCtx.Directory.Type,
		},
	}
	if maxDepth >= 0 {
		depth := maxDepth
		data.MaxDepth = &depth
	}

	for _, child := range workspaceCtx.Children {
		node := readDirNodeData{
			Name:         child.Name,
			Code:         child.Code,
			Type:         child.Type,
			Description:  child.Description,
			FullCodePath: child.FullCodePath,
			TemplateType: child.TemplateType,
		}
		if child.Type == servicetree.TypePackage || child.Type == servicetree.TypeDocs {
			data.Directories = append(data.Directories, node)
			continue
		}
		if child.Type == servicetree.TypeFunction && includeFunctions {
			data.Functions = append(data.Functions, node)
		}
	}

	if includeFiles {
		data.Files = make([]readDirFileData, 0, len(workspaceCtx.Files))
		for _, file := range workspaceCtx.Files {
			item := readDirFileData{
				FileName:      file.FileName,
				RelativePath:  file.RelativePath,
				FullPath:      strings.TrimRight(targetPath, "/") + "/" + file.RelativePath,
				FileType:      file.FileType,
				LineCount:     workspaceFileLineCount(file),
				ContentLength: file.ContentLength,
			}
			if includeCode {
				item.Content = file.Content
			}
			data.Files = append(data.Files, item)
		}
	}

	data.Summary = readDirSummaryData{
		DirectoryCount: len(data.Directories),
		FunctionCount:  len(data.Functions),
		FileCount:      len(data.Files),
	}
	return data
}

func isFunctionPath(path string) bool {
	path = strings.TrimSuffix(path, "/")
	lastSlash := strings.LastIndex(path, "/")
	lastSegment := path
	if lastSlash >= 0 {
		lastSegment = path[lastSlash+1:]
	}
	return strings.Contains(lastSegment, ".")
}

func getParentPath(path string) string {
	path = strings.TrimSuffix(path, "/")
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash <= 0 {
		return ""
	}
	return path[:lastSlash]
}
