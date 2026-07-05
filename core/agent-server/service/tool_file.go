package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/kageos/kageos/core/agent-server/prompt"
	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
)

type ReadFileTool struct{}

type readFileArgs struct {
	Directory    string `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
	FileName     string `json:"file_name" schema_desc:"目标文件名，如 issue.go" schema_required:"true"`
	LineRanges   string `json:"line_ranges" schema_desc:"可选行号范围，如 1-30,120-140；不传则返回全文"`
}

type readFileResultData struct {
	TargetPath      string             `json:"target_path" schema_desc:"实际读取的目录路径" schema_required:"true"`
	FileName        string             `json:"file_name" schema_desc:"文件名" schema_required:"true"`
	RelativePath    string             `json:"relative_path" schema_desc:"文件相对路径" schema_required:"true"`
	FullPath        string             `json:"full_path" schema_desc:"完整文件路径" schema_required:"true"`
	FileType        string             `json:"file_type" schema_desc:"文件类型" schema_required:"true"`
	LineCount       int                `json:"line_count" schema_desc:"完整文件总行数" schema_required:"true"`
	ContentLength   int                `json:"content_length" schema_desc:"完整文件内容长度" schema_required:"true"`
	ContentSHA      string             `json:"content_sha" schema_desc:"完整文件内容 sha256，写入时作为 base_sha 传回" schema_required:"true"`
	Content         string             `json:"content" schema_desc:"读取范围内的真实文件内容，不含行号" schema_required:"true"`
	NumberedContent string             `json:"numbered_content" schema_desc:"读取范围内带行号的内容，用于定位修改位置" schema_required:"true"`
	Ranges          []readFileLineSpan `json:"ranges" schema_desc:"本次返回的行号范围" schema_required:"true"`
}

type readFileLineSpan struct {
	Start int `json:"start" schema_desc:"起始行号，1-based" schema_required:"true"`
	End   int `json:"end" schema_desc:"结束行号，1-based" schema_required:"true"`
}

var readFileToolDef = toolDefinitionWithOutput[readFileArgs, structuredToolResultSchema[readFileResultData]](
	"read_file",
	"读取工作区真实文本/代码文件。必填 file_name；可选 directory 和 line_ranges；.docs 会读取工作台文档库。返回真实 content、带行号 numbered_content、完整文件 content_sha。后续 edit_file/write_file 修改同一文件时必须传回最新 content_sha 作为 base_sha。/system/prompt 文档请用 read_doc。",
)

func (t *ReadFileTool) Definition() dto.ToolDef {
	return readFileToolDef
}

func (t *ReadFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[readFileArgs](call.Args)
	if err != nil {
		return toolResult("read_file 参数解析失败: "+err.Error(), true)
	}
	return runReadFileTool(ctx, args, call.FullCodePath)
}

type EditFileTool struct{}

type editFileLineEditArgs struct {
	StartLine       int    `json:"start_line" schema_desc:"起始行号，1-based，包含" schema_required:"true"`
	EndLine         int    `json:"end_line" schema_desc:"结束行号，1-based，包含" schema_required:"true"`
	ExpectedOldText string `json:"expected_old_text" schema_desc:"可选；替换前该行段的精确旧文本，不含行号"`
	Replacement     string `json:"replacement" schema_desc:"替换后的完整文本；空字符串表示删除该行段" schema_required:"true"`
}

type editFileSearchEditArgs struct {
	OldText       string `json:"old_text" schema_desc:"要精确查找的旧文本，必须包含真实空格、Tab 和换行" schema_required:"true"`
	NewText       string `json:"new_text" schema_desc:"替换后的新文本；空字符串表示删除" schema_required:"true"`
	ExpectedCount int    `json:"expected_count" schema_desc:"期望替换次数；不传或小于 1 时默认为 1"`
}

type editFileArgs struct {
	Directory    string                   `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath string                   `json:"full_code_path" schema_ignore:"true"`
	FileName     string                   `json:"file_name" schema_desc:"目标文件名，如 issue.go" schema_required:"true"`
	BaseSHA      string                   `json:"base_sha" schema_desc:"最近一次 read_file 返回的 content_sha" schema_required:"true"`
	SearchEdits  []editFileSearchEditArgs `json:"search_edits" schema_desc:"精确文本替换列表；推荐用于小范围修改。与 line_edits 二选一"`
	LineEdits    []editFileLineEditArgs   `json:"line_edits" schema_desc:"行号替换列表；用于明确行号的块级修改。与 search_edits 二选一"`
}

type editFileResultData struct {
	TargetPath      string   `json:"target_path" schema_desc:"实际修改目录" schema_required:"true"`
	FileName        string   `json:"file_name" schema_desc:"文件名" schema_required:"true"`
	OldSHA          string   `json:"old_sha" schema_desc:"修改前 sha" schema_required:"true"`
	NewSHA          string   `json:"new_sha" schema_desc:"修改后 sha" schema_required:"true"`
	Mode            string   `json:"mode" schema_desc:"search_edits 或 line_edits" schema_required:"true"`
	AppliedSearches int      `json:"applied_searches,omitempty" schema_desc:"应用的精确文本替换项数"`
	AppliedEdits    int      `json:"applied_edits,omitempty" schema_desc:"应用的行号编辑数"`
	Changed         bool     `json:"changed" schema_desc:"内容是否发生变化" schema_required:"true"`
	Diagnostics     []string `json:"diagnostics,omitempty" schema_desc:"写后诊断摘要"`
}

var editFileToolDef = toolDefinitionWithOutput[editFileArgs, structuredToolResultSchema[editFileResultData]](
	"edit_file",
	"修改已有工作区代码文件。必须先 read_file 获取 content_sha，并作为 base_sha 传入；文件变化则拒绝。推荐 search_edits 精确文本替换；行号明确或块级修改时用 line_edits。两种模式二选一，所有 edit 原子应用，任一不匹配则不落盘。当前版本仅写入 .go 代码文件；工作台文档不通过 edit_file 修改。",
)

func (t *EditFileTool) Definition() dto.ToolDef {
	return editFileToolDef
}

func (t *EditFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[editFileArgs](call.Args)
	if err != nil {
		return toolResult("edit_file 参数解析失败: "+err.Error(), true)
	}
	return runEditFileTool(ctx, args, call.FullCodePath)
}

type WriteFileTool struct{}

type writeFileArgs struct {
	Directory         string `json:"directory" schema_desc:"目标目录，不传则当前工作目录"`
	FullCodePath      string `json:"full_code_path" schema_ignore:"true"`
	FileName          string `json:"file_name" schema_desc:"目标文件名，如 issue.go" schema_required:"true"`
	FileType          string `json:"file_type" schema_desc:"可选文件类型；不传则从 file_name 后缀推断，无扩展名默认 go"`
	Content           string `json:"content" schema_desc:"完整文件内容" schema_required:"true"`
	BaseSHA           string `json:"base_sha" schema_desc:"覆盖已有文件时必须传最近一次 read_file 返回的 content_sha"`
	ReplaceEntireFile bool   `json:"replace_entire_file" schema_desc:"覆盖已有文件时必须显式为 true；新文件可不传"`
	OverwriteReason   string `json:"overwrite_reason" schema_desc:"覆盖已有文件的原因；新文件可不传"`
}

type writeFileResultData struct {
	TargetPath  string   `json:"target_path" schema_desc:"实际写入目录" schema_required:"true"`
	FileName    string   `json:"file_name" schema_desc:"文件名" schema_required:"true"`
	OldSHA      string   `json:"old_sha,omitempty" schema_desc:"覆盖前 sha"`
	NewSHA      string   `json:"new_sha" schema_desc:"写入后 sha" schema_required:"true"`
	Created     bool     `json:"created" schema_desc:"是否新建文件" schema_required:"true"`
	Changed     bool     `json:"changed" schema_desc:"内容是否发生变化" schema_required:"true"`
	Diagnostics []string `json:"diagnostics,omitempty" schema_desc:"写后诊断摘要"`
}

var writeFileToolDef = toolDefinitionWithOutput[writeFileArgs, structuredToolResultSchema[writeFileResultData]](
	"write_file",
	"创建或完整覆盖工作区文本文件。新文件可直接写；覆盖已有文件必须先 read_file，并传 base_sha、replace_entire_file=true、overwrite_reason。.go 文件会做 Go/SDK 诊断；.docs 会写入工作台文档库；其他受支持文本扩展写入应用目录但不触发编译。小范围修改优先用 edit_file。",
)

func (t *WriteFileTool) Definition() dto.ToolDef {
	return writeFileToolDef
}

func (t *WriteFileTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[writeFileArgs](call.Args)
	if err != nil {
		return toolResult("write_file 参数解析失败: "+err.Error(), true)
	}
	return runWriteFileTool(ctx, args, call.FullCodePath)
}

func runReadFileTool(ctx context.Context, args readFileArgs, currentFullCodePath string) ToolResult {
	targetPath, matched, errMsg, isError := readWorkspaceFile(ctx, args.Directory, args.FullCodePath, currentFullCodePath, args.FileName)
	if errMsg != "" {
		return toolResult(errMsg, isError)
	}
	data := buildReadFileResultData(targetPath, matched, args.LineRanges)
	content := fmt.Sprintf("文件 %s（目录：%s）共 %d 行，content_sha=%s\n\n```%s\n%s\n```\n\n带行号内容：\n%s",
		matched.RelativePath, targetPath, data.LineCount, data.ContentSHA, matched.FileType, data.Content, data.NumberedContent)
	return toolResultWithData(content, false, data)
}

func runEditFileTool(ctx context.Context, args editFileArgs, currentFullCodePath string) ToolResult {
	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		return toolResult("edit_file 缺少参数 file_name。", true)
	}
	if !isGoFileName(ensureGoFileName(fileName)) {
		return toolResult("edit_file 当前仅支持 .go 代码文件；工作台文档不通过 edit_file 修改。", true)
	}
	if isGeneratedInitGoFile(fileName) {
		return toolResult("edit_file 不允许修改 init_.go；该文件由目录创建流程自动维护。请修改普通业务 .go 文件。", true)
	}
	if strings.TrimSpace(args.BaseSHA) == "" {
		return toolResult("edit_file 必须传 base_sha；请先 read_file 获取最新 content_sha。", true)
	}
	hasLineEdits := len(args.LineEdits) > 0
	hasSearchEdits := len(args.SearchEdits) > 0
	modeCount := 0
	for _, ok := range []bool{hasSearchEdits, hasLineEdits} {
		if ok {
			modeCount++
		}
	}
	if modeCount != 1 {
		return toolResult("edit_file 的 search_edits 和 line_edits 必须二选一。", true)
	}

	targetPath, matched, errMsg, isError := readWorkspaceFile(ctx, args.Directory, args.FullCodePath, currentFullCodePath, fileName)
	if errMsg != "" {
		return toolResult(errMsg, isError)
	}
	oldSHA := fileContentSHA(matched.Content)
	if normalizeContentSHA(args.BaseSHA) != oldSHA {
		return toolResult(fmt.Sprintf("edit_file 拒绝写入：base_sha=%s 与当前文件 sha=%s 不一致。请重新 read_file 后再修改。", args.BaseSHA, oldSHA), true)
	}

	var newContent string
	var applied int
	var err error
	mode := "search_edits"
	if hasSearchEdits {
		newContent, applied, err = applySearchEditsToContent(matched.Content, args.SearchEdits)
	} else {
		mode = "line_edits"
		newContent, applied, err = applyLineEditsToContent(matched.Content, args.LineEdits)
	}
	if err != nil {
		return toolResult("edit_file 未落盘："+err.Error(), true)
	}
	if msg := blockingGoWriteDiagnostics(targetPath, fileName, newContent); msg != "" {
		return toolResult("edit_file 未落盘：修改后 Go 语法检查失败。\n"+msg, true)
	}

	newSHA := fileContentSHA(newContent)
	msg, writeErr := writeCodeFileContent(ctx, targetPath, fileName, newContent, "edit_file")
	if writeErr {
		return toolResult(msg, true)
	}
	msg = appendGoFileDiagnostics(msg, targetPath, fileName, newContent)
	data := editFileResultData{
		TargetPath: targetPath,
		FileName:   ensureGoFileName(fileName),
		OldSHA:     oldSHA,
		NewSHA:     newSHA,
		Mode:       mode,
		Changed:    oldSHA != newSHA,
		Diagnostics: []string{
			"go 文件已完成文件级自动诊断；最终跨文件/schema 结果仍以 build_workspace 为准。",
		},
	}
	if hasSearchEdits {
		data.AppliedSearches = applied
	} else {
		data.AppliedEdits = applied
	}
	return toolResultWithData(msg+"\n\n"+formatStructuredToolData(data), false, data)
}

func runWriteFileTool(ctx context.Context, args writeFileArgs, currentFullCodePath string) ToolResult {
	fileName := strings.TrimSpace(args.FileName)
	if fileName == "" {
		return toolResult("write_file 缺少参数 file_name。", true)
	}
	if strings.TrimSpace(args.Content) == "" {
		return toolResult("write_file 缺少参数 content，本次未落盘。", true)
	}

	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		return toolResult("write_file 需要 directory（当前目录）。", true)
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}

	if isDocsFileName(fileName) || strings.EqualFold(strings.TrimPrefix(strings.TrimSpace(args.FileType), "."), "docs") {
		return runWriteDocsFileTool(ctx, args, targetPath)
	}

	fileName = normalizeWriteFileName(fileName, args.FileType)
	if isGeneratedInitGoFile(fileName) {
		return toolResult("write_file 不允许修改 init_.go；该文件由目录创建流程自动维护。请修改普通业务文件。", true)
	}
	existing, errMsg, isError := findWorkspaceFile(ctx, targetPath, fileName)
	if errMsg != "" && isError {
		return toolResult(errMsg, true)
	}
	created := existing == nil
	oldSHA := ""
	if existing != nil {
		oldSHA = fileContentSHA(existing.Content)
		if !args.ReplaceEntireFile {
			return toolResult("write_file 拒绝覆盖已有文件：请改用 edit_file；若确需整文件覆盖，先 read_file 后传 replace_entire_file=true、base_sha 和 overwrite_reason。", true)
		}
		if strings.TrimSpace(args.OverwriteReason) == "" {
			return toolResult("write_file 覆盖已有文件必须填写 overwrite_reason。", true)
		}
		if normalizeContentSHA(args.BaseSHA) != oldSHA {
			return toolResult(fmt.Sprintf("write_file 拒绝覆盖：base_sha=%s 与当前文件 sha=%s 不一致。请重新 read_file 后再写。", args.BaseSHA, oldSHA), true)
		}
	}

	newSHA := fileContentSHA(args.Content)
	msg := ""
	diagnostics := []string{"文本文件已落盘；如该文件参与应用运行，最终结果仍以 build_workspace 为准。"}
	if isGoFileName(fileName) {
		if msg := blockingGoWriteDiagnostics(targetPath, fileName, args.Content); msg != "" {
			return toolResult("write_file 未落盘：Go 语法检查失败。\n"+msg, true)
		}
		writeMsg, writeErr := writeCodeFileContent(ctx, targetPath, fileName, args.Content, "write_file")
		if writeErr {
			return toolResult(writeMsg, true)
		}
		msg = appendGoFileDiagnostics(writeMsg, targetPath, fileName, args.Content)
		diagnostics = []string{"go 文件已完成文件级自动诊断；最终跨文件/schema 结果仍以 build_workspace 为准。"}
	} else {
		resp, err := apicall.WriteFileContent(ctx, &dto.WriteFileContentReq{
			FullCodePath: targetPath,
			FileName:     fileName,
			FileType:     args.FileType,
			Content:      args.Content,
		})
		if err != nil {
			return toolResult("write_file 调用失败: "+err.Error(), true)
		}
		if resp != nil && !resp.Success {
			return toolResult("write_file 失败: "+resp.Message, true)
		}
		msg = fmt.Sprintf("已落盘: %s。当前未编译工作空间，仅修改了文本文件。", fileName)
		if resp != nil && resp.RelativePath != "" {
			msg = fmt.Sprintf("已落盘: %s。当前未编译工作空间，仅修改了文本文件。", resp.RelativePath)
		}
	}
	data := writeFileResultData{
		TargetPath:  targetPath,
		FileName:    fileName,
		OldSHA:      oldSHA,
		NewSHA:      newSHA,
		Created:     created,
		Changed:     oldSHA != newSHA,
		Diagnostics: diagnostics,
	}
	return toolResultWithData(msg+"\n\n"+formatStructuredToolData(data), false, data)
}

func runWriteDocsFileTool(ctx context.Context, args writeFileArgs, targetPath string) ToolResult {
	fileName := strings.TrimSpace(args.FileName)
	if !isDocsFileName(fileName) {
		fileName = strings.Trim(strings.TrimSpace(fileName), ".") + writeDocCodeSuffix
	}
	code := withoutWriteDocSuffix(filepath.Base(fileName))
	if code == "" {
		return toolResult("write_file 写 docs 文档时无法从 file_name 推断 code。", true)
	}

	docPath := strings.TrimRight(targetPath, "/") + "/" + withWriteDocSuffix(code)
	var existingContent string
	created := true
	if doc, err := apicall.GetDoc(ctx, docPath); err == nil && doc != nil {
		existingContent = doc.Content
		created = false
	} else if err != nil && !isWorkspaceDocNotFoundError(err) {
		return toolResult("write_file 读取现有文档失败: "+err.Error(), true)
	}
	oldSHA := ""
	if !created {
		oldSHA = fileContentSHA(existingContent)
		if !args.ReplaceEntireFile {
			return toolResult("write_file 拒绝覆盖已有文档：请先 read_file 获取 content_sha，再传 replace_entire_file=true、base_sha 和 overwrite_reason。", true)
		}
		if strings.TrimSpace(args.OverwriteReason) == "" {
			return toolResult("write_file 覆盖已有文档必须填写 overwrite_reason。", true)
		}
		if normalizeContentSHA(args.BaseSHA) != oldSHA {
			return toolResult(fmt.Sprintf("write_file 拒绝覆盖文档：base_sha=%s 与当前文档 sha=%s 不一致。请重新 read_file 后再写。", args.BaseSHA, oldSHA), true)
		}
	}

	docName := code
	if code == "runbook" {
		docName = "运行手册"
	}
	msg, isErr := runWriteDocCommand(ctx, writeDocCommand{
		FullCodePath: targetPath,
		Name:         docName,
		Code:         code,
		Content:      args.Content,
		Format:       "markdown",
	}, targetPath)
	if isErr {
		return toolResult(msg, true)
	}

	newSHA := fileContentSHA(args.Content)
	data := writeFileResultData{
		TargetPath: targetPath,
		FileName:   withWriteDocSuffix(code),
		OldSHA:     oldSHA,
		NewSHA:     newSHA,
		Created:    created,
		Changed:    oldSHA != newSHA,
		Diagnostics: []string{
			"docs 文档已写入工作台文档库；不会触发 build_workspace。",
		},
	}
	return toolResultWithData(msg+"\n\n"+formatStructuredToolData(data), false, data)
}

func isWorkspaceDocNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, marker := range []string{"不存在", "not found", "record not found", "文档不存在"} {
		if strings.Contains(msg, strings.ToLower(marker)) {
			return true
		}
	}
	return false
}

func readWorkspaceFile(ctx context.Context, directory string, fullCodePath string, currentFullCodePath string, fileName string) (string, *dto.WorkspaceContextFile, string, bool) {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return "", nil, "read_file 需传 file_name。", true
	}
	targetPath := resolveDirectoryArg(directory, fullCodePath, currentFullCodePath)
	targetPath = strings.TrimRight(targetPath, "/")
	if targetPath == "" {
		return "", nil, "read_file 需要 directory（当前目录）。", true
	}
	if !strings.HasPrefix(targetPath, "/") {
		targetPath = "/" + targetPath
	}
	if prompt.IsPromptDocPath(targetPath) {
		return "", nil, "该路径是文档路径，请使用 read_doc 读取，不要用 read_file。", true
	}
	if isDocsFileName(fileName) {
		docPath := strings.TrimRight(targetPath, "/") + "/" + filepath.Base(fileName)
		doc, err := apicall.GetDoc(ctx, docPath)
		if err != nil {
			return targetPath, nil, fmt.Sprintf("读取文档失败: %v", err), true
		}
		if doc == nil {
			return targetPath, nil, fmt.Sprintf("在目录 %s 下未找到文档：%s", targetPath, fileName), true
		}
		return targetPath, &dto.WorkspaceContextFile{
			FileName:      withoutWriteDocSuffix(filepath.Base(fileName)),
			RelativePath:  filepath.Base(fileName),
			FileType:      "docs",
			Content:       doc.Content,
			ContentLength: len(doc.Content),
			LineCount:     countContentLines(doc.Content),
		}, "", false
	}
	file, errMsg, _ := findWorkspaceFile(ctx, targetPath, fileName)
	if errMsg != "" {
		return targetPath, nil, errMsg, true
	}
	return targetPath, file, "", false
}

func findWorkspaceFile(ctx context.Context, targetPath string, fileName string) (*dto.WorkspaceContextFile, string, bool) {
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return nil, fmt.Sprintf("获取文件失败: %v", err), true
	}
	if len(workspaceCtx.Files) == 0 {
		return nil, fmt.Sprintf("目录 %s 下没有文件。", targetPath), false
	}
	fileName = strings.TrimSpace(fileName)
	for i := range workspaceCtx.Files {
		f := &workspaceCtx.Files[i]
		if workspaceFileMatchesName(*f, fileName) {
			return f, "", false
		}
	}
	return nil, fmt.Sprintf("在目录 %s 下未找到文件：%s", targetPath, fileName), false
}

func workspaceFileMatchesName(file dto.WorkspaceContextFile, fileName string) bool {
	fileName = strings.TrimSpace(fileName)
	return file.FileName == fileName ||
		file.FileName+"."+file.FileType == fileName ||
		file.RelativePath == fileName ||
		filepath.Base(file.RelativePath) == fileName
}

type lineRange struct {
	Start int
	End   int
}

func parseLineRanges(s string, totalLines int) []lineRange {
	s = strings.TrimSpace(s)
	if totalLines <= 0 {
		totalLines = 1
	}
	if s == "" {
		return []lineRange{{Start: 1, End: totalLines}}
	}
	var out []lineRange
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "-")
		if idx < 0 {
			var n int
			if _, err := fmt.Sscanf(part, "%d", &n); err == nil && n >= 1 {
				if n > totalLines {
					n = totalLines
				}
				out = append(out, lineRange{Start: n, End: n})
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
			out = append(out, lineRange{Start: start, End: end})
		}
	}
	if len(out) == 0 {
		return []lineRange{{Start: 1, End: totalLines}}
	}
	return out
}

func buildReadFileResultData(targetPath string, file *dto.WorkspaceContextFile, lineRangesStr string) *readFileResultData {
	fullLines, _ := splitContentLines(file.Content)
	totalLines := len(fullLines)
	ranges := parseLineRanges(lineRangesStr, totalLines)
	selectedContent, numbered := selectFileLineRanges(fullLines, ranges)
	outRanges := make([]readFileLineSpan, 0, len(ranges))
	for _, rng := range ranges {
		outRanges = append(outRanges, readFileLineSpan{Start: rng.Start, End: rng.End})
	}
	return &readFileResultData{
		TargetPath:      targetPath,
		FileName:        file.FileName,
		RelativePath:    file.RelativePath,
		FullPath:        strings.TrimRight(targetPath, "/") + "/" + file.RelativePath,
		FileType:        file.FileType,
		LineCount:       workspaceFileLineCount(*file),
		ContentLength:   file.ContentLength,
		ContentSHA:      fileContentSHA(file.Content),
		Content:         selectedContent,
		NumberedContent: numbered,
		Ranges:          outRanges,
	}
}

func selectFileLineRanges(lines []string, ranges []lineRange) (string, string) {
	if len(lines) == 0 {
		return "", ""
	}
	width := 1
	for n := len(lines); n >= 10; n /= 10 {
		width++
	}
	var raw strings.Builder
	var numbered strings.Builder
	firstRaw := true
	for idx, rng := range ranges {
		if idx > 0 {
			numbered.WriteString("...\n")
		}
		for lineNo := rng.Start; lineNo <= rng.End && lineNo <= len(lines); lineNo++ {
			if !firstRaw {
				raw.WriteString("\n")
			}
			firstRaw = false
			line := lines[lineNo-1]
			raw.WriteString(line)
			numbered.WriteString(fmt.Sprintf("%*d | %s\n", width, lineNo, line))
		}
	}
	return raw.String(), numbered.String()
}

func fileContentSHA(content string) string {
	sum := sha256.Sum256([]byte(content))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func normalizeContentSHA(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "sha256:") {
		return raw
	}
	return "sha256:" + raw
}

func normalizeWriteFileName(fileName, fileType string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || filepath.Ext(filepath.Base(fileName)) != "" {
		return fileName
	}
	fileType = strings.Trim(strings.TrimSpace(fileType), ".")
	if fileType != "" {
		return fileName + "." + fileType
	}
	return ensureGoFileName(fileName)
}

func ensureGoFileName(fileName string) string {
	fileName = strings.TrimSpace(fileName)
	if fileName == "" || strings.Contains(filepath.Base(fileName), ".") {
		return fileName
	}
	return fileName + ".go"
}

func isDocsFileName(fileName string) bool {
	return strings.EqualFold(filepath.Ext(strings.TrimSpace(fileName)), writeDocCodeSuffix)
}

func isGeneratedInitGoFile(fileName string) bool {
	return filepath.Base(ensureGoFileName(fileName)) == "init_.go"
}

func countContentLines(content string) int {
	if content == "" {
		return 0
	}
	lines := strings.Split(content, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		return len(lines) - 1
	}
	return len(lines)
}

func blockingGoWriteDiagnostics(directory string, fileName string, source string) string {
	if !isGoFileName(fileName) || strings.TrimSpace(source) == "" {
		return ""
	}
	result := checkGoFileLocalSource(directory, goSourceFileForCheck{
		Name:    ensureGoFileName(fileName),
		Content: source,
	})
	var blocking []checkWorkspaceCodeIssue
	for _, issue := range result.Issues {
		if issue.Severity == "error" && issue.Category == "go_syntax" {
			blocking = append(blocking, issue)
		}
	}
	if len(blocking) == 0 {
		return ""
	}
	var b strings.Builder
	limit := len(blocking)
	if limit > maxInlineGoDiagnostics {
		limit = maxInlineGoDiagnostics
	}
	for i := 0; i < limit; i++ {
		issue := blocking[i]
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("- [")
		b.WriteString(issue.Severity)
		b.WriteString("] ")
		b.WriteString(issue.Category)
		if issue.Line > 0 {
			b.WriteString(fmt.Sprintf(":%d", issue.Line))
		}
		b.WriteString(" - ")
		b.WriteString(issue.Message)
	}
	if len(blocking) > limit {
		b.WriteString(fmt.Sprintf("\n- 其余 %d 个阻断语法问题未展开。", len(blocking)-limit))
	}
	return b.String()
}

func writeCodeFileContent(ctx context.Context, targetPath string, fileName string, content string, toolName string) (string, bool) {
	fileName = ensureGoFileName(fileName)
	msg, isError := runAddFunctionsCommandWithToolName(ctx, addFunctionsCommand{
		FileName:   fileName,
		SourceCode: content,
	}, targetPath, false, toolName)
	return msg, isError
}

func splitContentLines(content string) ([]string, bool) {
	hasTrailingNewline := strings.HasSuffix(content, "\n")
	if hasTrailingNewline {
		content = strings.TrimSuffix(content, "\n")
	}
	if content == "" {
		return []string{}, hasTrailingNewline
	}
	return strings.Split(content, "\n"), hasTrailingNewline
}

func joinContentLines(lines []string, trailingNewline bool) string {
	content := strings.Join(lines, "\n")
	if trailingNewline {
		content += "\n"
	}
	return content
}

func applySearchEditsToContent(content string, edits []editFileSearchEditArgs) (string, int, error) {
	if len(edits) == 0 {
		return "", 0, fmt.Errorf("search_edits 不能为空")
	}
	next := content
	for i, edit := range edits {
		if edit.OldText == "" {
			return "", 0, fmt.Errorf("第 %d 个 search_edit 缺少 old_text", i+1)
		}
		expected := edit.ExpectedCount
		if expected < 1 {
			expected = 1
		}
		actual := strings.Count(next, edit.OldText)
		if actual != expected {
			return "", 0, fmt.Errorf("第 %d 个 search_edit 匹配次数不符合预期：old_text 实际匹配 %d 次，期望 %d 次；请重新 read_file 后复制真实片段，或调整 expected_count", i+1, actual, expected)
		}
		next = strings.Replace(next, edit.OldText, edit.NewText, expected)
	}
	return next, len(edits), nil
}

func applyLineEditsToContent(content string, edits []editFileLineEditArgs) (string, int, error) {
	if len(edits) == 0 {
		return "", 0, fmt.Errorf("line_edits 不能为空")
	}
	lines, trailingNewline := splitContentLines(content)
	ascending := append([]editFileLineEditArgs{}, edits...)
	sort.SliceStable(ascending, func(i, j int) bool {
		return ascending[i].StartLine < ascending[j].StartLine
	})
	for i := 1; i < len(ascending); i++ {
		if ascending[i].StartLine <= ascending[i-1].EndLine {
			return "", 0, fmt.Errorf("第 %d 个 line_edit 与前一个编辑范围重叠；请合并为一个 replacement", i+1)
		}
	}
	ordered := append([]editFileLineEditArgs{}, ascending...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].StartLine > ordered[j].StartLine
	})
	for i, edit := range ordered {
		if edit.StartLine < 1 || edit.EndLine < edit.StartLine {
			return "", 0, fmt.Errorf("第 %d 个 line_edit 行号范围无效", i+1)
		}
		if edit.EndLine > len(lines) {
			return "", 0, fmt.Errorf("第 %d 个 line_edit 超出文件总行数 %d", i+1, len(lines))
		}
		oldText := strings.Join(lines[edit.StartLine-1:edit.EndLine], "\n")
		if expected := strings.TrimSuffix(edit.ExpectedOldText, "\n"); strings.TrimSpace(edit.ExpectedOldText) != "" && oldText != expected {
			return "", 0, fmt.Errorf("第 %d 个 line_edit 的 expected_old_text 与当前文件不一致；请重新 read_file 读取相关行", i+1)
		}
		replacement := strings.TrimSuffix(edit.Replacement, "\n")
		var replacementLines []string
		if replacement != "" {
			replacementLines = strings.Split(replacement, "\n")
		}
		lines = append(lines[:edit.StartLine-1], append(replacementLines, lines[edit.EndLine:]...)...)
	}
	return joinContentLines(lines, trailingNewline), len(edits), nil
}
