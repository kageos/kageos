package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/ai-agent-os/ai-agent-os/core/agent-server/prompt"
	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
)

type ReadGoFileTool struct{}

type readGoFileArgs struct {
	Directory    string `json:"directory" schema_desc:"代码目录，不传则当前目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	FileName     string `json:"file_name" schema_desc:"单个或多个文件名，逗号分隔"`
}

type readGoFileResultData struct {
	TargetPath     string                 `json:"target_path" schema_desc:"实际读取的目录路径" schema_required:"true"`
	RequestedNames []string               `json:"requested_names,omitempty" schema_desc:"请求的文件名列表"`
	FileCount      int                    `json:"file_count" schema_desc:"返回的文件数量" schema_required:"true"`
	Files          []readGoFileResultFile `json:"files" schema_desc:"匹配到的代码文件列表" schema_required:"true"`
}

type readGoFileResultFile struct {
	FileName      string `json:"file_name" schema_desc:"文件名" schema_required:"true"`
	RelativePath  string `json:"relative_path" schema_desc:"相对路径" schema_required:"true"`
	FullPath      string `json:"full_path" schema_desc:"完整文件路径" schema_required:"true"`
	FileType      string `json:"file_type" schema_desc:"文件类型" schema_required:"true"`
	LineCount     int    `json:"line_count" schema_desc:"代码总行数" schema_required:"true"`
	ContentLength int    `json:"content_length" schema_desc:"内容长度" schema_required:"true"`
	Content       string `json:"content" schema_desc:"文件内容" schema_required:"true"`
}

var readGoFileToolDef = toolDefinitionWithOutput[readGoFileArgs, structuredToolResultSchema[readGoFileResultData]](
	"read_go_file",
	"读取工作区内指定目录下的 Go 代码文件内容。参数：directory（可选，不传则当前工作目录）、file_name（可选，单文件如 a.go，或多文件逗号分隔如 a.go,b.go；不传则返回该目录下所有代码文件）。",
)

func (t *ReadGoFileTool) Definition() dto.ToolDef {
	return readGoFileToolDef
}

func (t *ReadGoFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readGoFileArgs](call.Args)
	if err != nil {
		return toolResult("read_go_file 参数解析失败: "+err.Error(), true)
	}
	return runReadGoFileTool(ctx, args, call.FullCodePath)
}

// runReadGoFileTool 读取工作区 Go 代码文件；若传入的是文档路径则降级为用 read_doc 拉取并提示
func runReadGoFileTool(ctx context.Context, args readGoFileArgs, currentFullCodePath string) ToolResult {
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	fileName := strings.TrimSpace(args.FileName)

	// 降级：若 directory 是提示词/文档路径，用文档工具拉取内容并提示应使用 read_doc
	if prompt.IsPromptDocPath(targetPath) {
		docPath := strings.TrimSpace(targetPath)
		if !strings.HasPrefix(docPath, "/") {
			docPath = "/" + docPath
		}
		docName, content := prompt.GetPromptDocContent(ctx, docPath)
		if content != "" {
			hint := "【提示】你当前用 read_go_file 读取的是文档路径。应使用 read_doc(directory: \"" + docPath + "\") 读取文档；已为你拉取内容，下次请用 read_doc。\n\n"
			if docName == "" {
				docName = docPath
			}
			return toolResult(hint+"## "+docName+"\n\n"+content, false)
		}
		return toolResult("该路径是文档路径，请使用 read_doc(directory: \""+docPath+"\") 读取，不要用 read_go_file。", true)
	}

	// 读代码文件时从 runtime 磁盘实时读，保证内容与当前磁盘一致（快照表可能不准）
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return toolResult(fmt.Sprintf("获取代码失败: %v", err), true)
	}

	if len(workspaceCtx.Files) == 0 {
		return toolResult(fmt.Sprintf("目录 %s 下没有代码文件。", targetPath), false)
	}

	var matchedFiles []dto.WorkspaceContextFile
	if fileName != "" {
		// 支持逗号分隔多文件，如 a.go,b.go
		names := splitFileNames(fileName)
		seen := make(map[string]bool)
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			for _, file := range workspaceCtx.Files {
				key := file.RelativePath
				if seen[key] {
					continue
				}
				if file.FileName == name {
					matchedFiles = append(matchedFiles, file)
					seen[key] = true
					continue
				}
				fullFileName := file.FileName + "." + file.FileType
				if fullFileName == name {
					matchedFiles = append(matchedFiles, file)
					seen[key] = true
					continue
				}
				if file.RelativePath == name {
					matchedFiles = append(matchedFiles, file)
					seen[key] = true
					continue
				}
			}
		}
		if len(matchedFiles) == 0 {
			return toolResult(fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false)
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
		lineCount := workspaceFileLineCount(file)
		fullFilePath := strings.TrimRight(targetPath, "/") + "/" + file.RelativePath
		fileHeader := ""
		if len(matchedFiles) > 1 {
			fileHeader = fmt.Sprintf("## 文件 %d: %s\n", i+1, fullFilePath)
		}
		filesContent += fmt.Sprintf(`%s- 文件名: %s
- 文件路径: %s
- 文件类型: %s
- 总行数: %d 行
- 内容长度: %d 字符
- 代码内容:
`+"```%s\n%s\n```\n\n", fileHeader, file.FileName, fullFilePath, file.FileType, lineCount, file.ContentLength, file.FileType, file.Content)
	}
	return toolResultWithData(header+filesContent, false, buildReadGoFileResultData(targetPath, fileName, matchedFiles))
}

func buildReadGoFileResultData(targetPath string, fileName string, matchedFiles []dto.WorkspaceContextFile) *readGoFileResultData {
	files := make([]readGoFileResultFile, 0, len(matchedFiles))
	for _, file := range matchedFiles {
		files = append(files, readGoFileResultFile{
			FileName:      file.FileName,
			RelativePath:  file.RelativePath,
			FullPath:      strings.TrimRight(targetPath, "/") + "/" + file.RelativePath,
			FileType:      file.FileType,
			LineCount:     workspaceFileLineCount(file),
			ContentLength: file.ContentLength,
			Content:       file.Content,
		})
	}

	data := &readGoFileResultData{
		TargetPath: targetPath,
		FileCount:  len(files),
		Files:      files,
	}
	if names := splitFileNames(fileName); len(names) > 0 {
		data.RequestedNames = names
	}
	return data
}
