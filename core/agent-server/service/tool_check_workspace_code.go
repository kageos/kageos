package service

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strings"

	"github.com/kageos/kageos/dto"
	"github.com/kageos/kageos/pkg/apicall"
)

type CheckWorkspaceCodeTool struct{}

type checkWorkspaceCodeArgs struct {
	Directory    string `json:"directory" schema_desc:"要检查的代码目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
}

type checkWorkspaceCodeResultData struct {
	Directory  string                    `json:"directory" schema_desc:"实际检查目录" schema_required:"true"`
	FileCount  int                       `json:"file_count" schema_desc:"检查的代码文件数量" schema_required:"true"`
	IssueCount int                       `json:"issue_count" schema_desc:"发现的问题数量" schema_required:"true"`
	Passed     bool                      `json:"passed" schema_desc:"是否未发现问题" schema_required:"true"`
	Issues     []checkWorkspaceCodeIssue `json:"issues,omitempty" schema_desc:"发现的问题列表"`
}

type checkWorkspaceCodeIssue struct {
	File     string `json:"file" schema_desc:"文件名或相对路径" schema_required:"true"`
	Line     int    `json:"line,omitempty" schema_desc:"行号"`
	Severity string `json:"severity" schema_desc:"严重程度 error/warning" schema_required:"true"`
	Category string `json:"category" schema_desc:"问题分类" schema_required:"true"`
	Message  string `json:"message" schema_desc:"问题说明" schema_required:"true"`
}

type goSourceFileForCheck struct {
	Name    string
	Content string
}

type parsedGoSourceFileForCheck struct {
	Source   goSourceFileForCheck
	FileSet  *token.FileSet
	Parsed   *ast.File
	ParseErr error
}

var checkWorkspaceCodeToolDef = toolDefinitionWithOutput[checkWorkspaceCodeArgs, structuredToolResultSchema[checkWorkspaceCodeResultData]](
	"check_workspace_code",
	"对当前工作空间 Go 代码做 build 前轻量预检：语法、常见 import 问题、widget type、options_colors、筛选字段、错误文件名等。只读、无副作用，不能替代 build_workspace。",
)

func (t *CheckWorkspaceCodeTool) Definition() dto.ToolDef {
	return checkWorkspaceCodeToolDef
}

func (t *CheckWorkspaceCodeTool) Execute(ctx context.Context, call ToolCall) ToolResult {
	args, err := decodeToolArgs[checkWorkspaceCodeArgs](call.Args)
	if err != nil {
		return toolResult("check_workspace_code 参数解析失败: "+err.Error(), true)
	}
	targetPath := resolveDirectoryArg(args.Directory, args.FullCodePath, call.FullCodePath)
	workspaceCtx, err := apicall.GetWorkspaceContext(ctx, targetPath, "runtime")
	if err != nil {
		return toolResult(fmt.Sprintf("获取代码失败: %v", err), true)
	}
	files := make([]goSourceFileForCheck, 0, len(workspaceCtx.Files))
	for _, file := range workspaceCtx.Files {
		if file.FileType != "go" {
			continue
		}
		files = append(files, goSourceFileForCheck{
			Name:    file.RelativePath,
			Content: file.Content,
		})
	}
	result := checkWorkspaceGoSources(targetPath, files)
	return toolResultWithStructuredData(result, false)
}

func checkWorkspaceGoSources(directory string, files []goSourceFileForCheck) checkWorkspaceCodeResultData {
	var issues []checkWorkspaceCodeIssue
	for _, file := range files {
		parsed := parseGoSourceFileForCheck(file)
		issues = append(issues, checkGoFileName(file.Name)...)
		issues = append(issues, checkParsedGoFileSyntaxAndImports(parsed)...)
		issues = append(issues, checkParsedGoStructTags(parsed)...)
		issues = append(issues, checkParsedGoFileSchemaPatterns(parsed)...)
	}
	return buildCheckWorkspaceResult(directory, files, issues)
}

func checkGoFileLocalSource(directory string, file goSourceFileForCheck) checkWorkspaceCodeResultData {
	parsed := parseGoSourceFileForCheck(file)
	var issues []checkWorkspaceCodeIssue
	issues = append(issues, checkGoFileName(file.Name)...)
	issues = append(issues, checkParsedGoFileSyntaxAndImports(parsed)...)
	issues = append(issues, checkParsedGoStructTags(parsed)...)
	return buildCheckWorkspaceResult(directory, []goSourceFileForCheck{file}, issues)
}

func buildCheckWorkspaceResult(directory string, files []goSourceFileForCheck, issues []checkWorkspaceCodeIssue) checkWorkspaceCodeResultData {
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].File != issues[j].File {
			return issues[i].File < issues[j].File
		}
		return issues[i].Line < issues[j].Line
	})
	return checkWorkspaceCodeResultData{
		Directory:  directory,
		FileCount:  len(files),
		IssueCount: len(issues),
		Passed:     len(issues) == 0,
		Issues:     issues,
	}
}

func checkGoFileName(fileName string) []checkWorkspaceCodeIssue {
	base := path.Base(fileName)
	for _, bad := range []string{".table.go", ".form.go", ".chart.go"} {
		if strings.HasSuffix(base, bad) {
			return []checkWorkspaceCodeIssue{{
				File:     fileName,
				Severity: "error",
				Category: "file_name",
				Message:  "代码文件名只用普通 .go，路由后缀只能写在 packageContext.GET/POST 的路由字符串里，不能写成 " + bad,
			}}
		}
	}
	return nil
}

func parseGoSourceFileForCheck(file goSourceFileForCheck) parsedGoSourceFileForCheck {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Name, file.Content, 0)
	return parsedGoSourceFileForCheck{
		Source:   file,
		FileSet:  fset,
		Parsed:   parsed,
		ParseErr: err,
	}
}
