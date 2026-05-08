package service

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/ai-agent-os/ai-agent-os/dto"
	"github.com/ai-agent-os/ai-agent-os/pkg/apicall"
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/widget"
)

type CheckWorkspaceCodeTool struct{}

type checkWorkspaceCodeArgs struct {
	Directory    string `json:"directory" schema_desc:"要检查的代码目录，不传则当前工作目录"`
	FullCodePath string `json:"full_code_path" schema_ignore:"true"`
}

type checkWorkspaceCodeResultData struct {
	Directory  string                    `json:"directory" schema_desc:"实际检查目录" schema_required:"true"`
	FileCount  int                       `json:"file_count" schema_desc:"检查的 Go 文件数量" schema_required:"true"`
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
		issues = append(issues, checkGoFileName(file.Name)...)
		issues = append(issues, checkGoFileSyntaxAndImports(file)...)
		issues = append(issues, checkGoStructTags(file)...)
		issues = append(issues, checkGoFileSchemaPatterns(file)...)
	}
	return buildCheckWorkspaceResult(directory, files, issues)
}

func checkGoFileLocalSource(directory string, file goSourceFileForCheck) checkWorkspaceCodeResultData {
	var issues []checkWorkspaceCodeIssue
	issues = append(issues, checkGoFileName(file.Name)...)
	issues = append(issues, checkGoFileSyntaxAndImports(file)...)
	issues = append(issues, checkGoStructTags(file)...)
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
				Message:  "Go 文件名只用普通 .go，路由后缀只能写在 packageContext.GET/POST 的路由字符串里，不能写成 " + bad,
			}}
		}
	}
	return nil
}

func checkGoFileSyntaxAndImports(file goSourceFileForCheck) []checkWorkspaceCodeIssue {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Name, file.Content, parser.ParseComments)
	if err != nil {
		line := 0
		if pos, ok := err.(interface{ Pos() token.Position }); ok {
			line = pos.Pos().Line
		}
		return []checkWorkspaceCodeIssue{{
			File:     file.Name,
			Line:     line,
			Severity: "error",
			Category: "go_syntax",
			Message:  err.Error(),
		}}
	}

	imports := map[string]goImportForCheck{}
	for _, spec := range parsed.Imports {
		importName := ""
		if spec.Name != nil {
			importName = spec.Name.Name
		}
		if importName == "_" || importName == "." {
			continue
		}
		pathValue := strings.Trim(spec.Path.Value, `"`)
		if importName == "" {
			importName = path.Base(pathValue)
		}
		imports[importName] = goImportForCheck{
			Name: importName,
			Path: pathValue,
			Line: fset.Position(spec.Pos()).Line,
		}
	}

	idents := map[string]struct{}{}
	selectorRoots := map[string]struct{}{}
	selectorUses := map[string]map[string]int{}
	ast.Inspect(parsed, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			idents[node.Name] = struct{}{}
		case *ast.SelectorExpr:
			if ident, ok := node.X.(*ast.Ident); ok {
				selectorRoots[ident.Name] = struct{}{}
				if selectorUses[ident.Name] == nil {
					selectorUses[ident.Name] = map[string]int{}
				}
				selectorUses[ident.Name][node.Sel.Name] = fset.Position(node.Sel.Pos()).Line
			}
		}
		return true
	})

	issues := checkGoFileStructurePatterns(file.Name, parsed, fset)
	for name, imp := range imports {
		if imp.Path == "github.com/ai-agent-os/ai-agent-os/sdk/agent-app" {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file.Name,
				Line:     imp.Line,
				Severity: "warning",
				Category: "sdk_import",
				Message:  "不要导入 sdk/agent-app 根包；Context、Template、ChartType 用 sdk/agent-app/app，图表结构用 sdk/agent-app/chart，响应用 sdk/agent-app/response",
			})
		}
		if _, ok := idents[name]; !ok {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file.Name,
				Line:     imp.Line,
				Severity: "warning",
				Category: "go_import",
				Message:  fmt.Sprintf("import %q 看起来未使用，build 时会报 imported and not used", name),
			})
		}
	}

	for root := range selectorRoots {
		if _, ok := knownImportRoots[root]; !ok {
			continue
		}
		if _, imported := imports[root]; !imported {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file.Name,
				Severity: "warning",
				Category: "go_import",
				Message:  fmt.Sprintf("代码使用了 %s. 但当前文件未导入对应包", root),
			})
		}
	}
	issues = append(issues, checkSDKSelectors(file.Name, imports, selectorUses)...)
	return issues
}

func checkGoFileStructurePatterns(fileName string, parsed *ast.File, fset *token.FileSet) []checkWorkspaceCodeIssue {
	var issues []checkWorkspaceCodeIssue
	nonImportDecls := 0
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if ok && gen.Tok == token.IMPORT {
			continue
		}
		nonImportDecls++
		if ok && gen.Tok == token.VAR {
			for _, spec := range gen.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok || valueSpec.Type != nil || len(valueSpec.Names) == 0 {
					continue
				}
				allBlank := true
				for _, name := range valueSpec.Names {
					if name == nil || name.Name != "_" {
						allBlank = false
						break
					}
				}
				if allBlank {
					issues = append(issues, checkWorkspaceCodeIssue{
						File:     fileName,
						Line:     fset.Position(valueSpec.Pos()).Line,
						Severity: "warning",
						Category: "go_file_structure",
						Message:  "不要创建 import shim 或占位声明（如 var _ = types.Time{}）来“共享导入”；Go import 是文件级的，缺包应在实际使用该符号的文件里导入",
					})
				}
			}
		}
	}
	if nonImportDecls == 0 {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     fileName,
			Line:     1,
			Severity: "warning",
			Category: "go_file_structure",
			Message:  "Go 文件只有 package/import、没有有效声明，通常是空占位文件；不要创建 nps_types.go 这类文件来修 import 或跨文件问题",
		})
	}
	return issues
}

type goImportForCheck struct {
	Name string
	Path string
	Line int
}

var knownImportRoots = map[string]struct{}{
	"fmt": {}, "time": {}, "strings": {}, "errors": {}, "query": {}, "logger": {},
	"app": {}, "callback": {}, "response": {}, "types": {}, "chart": {}, "gorm": {},
}

var sdkPackageDirsForSelectorCheck = map[string]string{
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/app":        "sdk/agent-app/app",
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/callback":   "sdk/agent-app/callback",
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/chart":      "sdk/agent-app/chart",
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/response":   "sdk/agent-app/response",
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/statistics": "sdk/agent-app/statistics",
	"github.com/ai-agent-os/ai-agent-os/sdk/agent-app/types":      "sdk/agent-app/types",
}

var (
	sdkExportedSymbolsOnce sync.Once
	sdkExportedSymbols     map[string]map[string]struct{}
	sdkExportedSymbolsErr  error
)

func checkSDKSelectors(fileName string, imports map[string]goImportForCheck, selectorUses map[string]map[string]int) []checkWorkspaceCodeIssue {
	symbols, err := loadSDKExportedSymbols()
	if err != nil {
		return nil
	}
	var issues []checkWorkspaceCodeIssue
	for alias, uses := range selectorUses {
		imp, ok := imports[alias]
		if !ok {
			continue
		}
		allowed, ok := symbols[imp.Path]
		if !ok {
			continue
		}
		for selector, line := range uses {
			if _, exists := allowed[selector]; exists {
				continue
			}
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     fileName,
				Line:     line,
				Severity: "error",
				Category: "sdk_selector",
				Message:  fmt.Sprintf("代码使用了未在 %s 中导出的 SDK 符号 %s.%s；请改用已读文档、案例或源码中真实存在的符号", imp.Path, alias, selector),
			})
		}
	}
	return issues
}

func loadSDKExportedSymbols() (map[string]map[string]struct{}, error) {
	sdkExportedSymbolsOnce.Do(func() {
		sdkExportedSymbols = map[string]map[string]struct{}{}
		repoRoot, err := findRepoRootForSDKSelectorCheck()
		if err != nil {
			sdkExportedSymbolsErr = err
			return
		}
		for importPath, relDir := range sdkPackageDirsForSelectorCheck {
			dir := filepath.Join(repoRoot, filepath.FromSlash(relDir))
			symbols, err := exportedSymbolsInPackageDir(dir)
			if err != nil {
				sdkExportedSymbolsErr = err
				return
			}
			sdkExportedSymbols[importPath] = symbols
		}
	})
	return sdkExportedSymbols, sdkExportedSymbolsErr
}

func findRepoRootForSDKSelectorCheck() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("无法定位当前源码文件")
	}
	dir := filepath.Dir(currentFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "sdk", "agent-app")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("无法定位仓库根目录")
		}
		dir = parent
	}
}

func exportedSymbolsInPackageDir(dir string) (map[string]struct{}, error) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	out := map[string]struct{}{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				switch node := decl.(type) {
				case *ast.FuncDecl:
					if node.Name != nil && node.Name.IsExported() {
						out[node.Name.Name] = struct{}{}
					}
				case *ast.GenDecl:
					for _, spec := range node.Specs {
						switch typed := spec.(type) {
						case *ast.TypeSpec:
							if typed.Name != nil && typed.Name.IsExported() {
								out[typed.Name.Name] = struct{}{}
							}
						case *ast.ValueSpec:
							for _, name := range typed.Names {
								if name != nil && name.IsExported() {
									out[name.Name] = struct{}{}
								}
							}
						}
					}
				}
			}
		}
	}
	return out, nil
}

func checkGoStructTags(file goSourceFileForCheck) []checkWorkspaceCodeIssue {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Name, file.Content, parser.ParseComments)
	if err != nil {
		return nil
	}
	var issues []checkWorkspaceCodeIssue
	ast.Inspect(parsed, func(n ast.Node) bool {
		field, ok := n.(*ast.Field)
		if !ok || field.Tag == nil {
			return true
		}
		line := fset.Position(field.Tag.Pos()).Line
		tag := strings.Trim(field.Tag.Value, "`")
		widgetTag := structTagValue(tag, "widget")
		callbackTag := structTagValue(tag, "callback")
		if widgetTag != "" && widgetTag != "-" {
			issues = append(issues, checkWidgetTag(file.Name, line, widgetTag, callbackTag)...)
		}
		return true
	})
	return issues
}

func checkWidgetTag(file string, line int, widgetTag string, callbackTag string) []checkWorkspaceCodeIssue {
	parsed := parseSemicolonTag(widgetTag)
	var issues []checkWorkspaceCodeIssue
	for _, invalid := range invalidSemicolonTagSegments(widgetTag) {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "widget_tag",
			Message:  fmt.Sprintf("widget tag 片段必须是 key:value，当前片段为 %q", invalid),
		})
	}
	widgetType := parsed["type"]
	if widgetType == "" {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "widget_type",
			Message:  "widget tag 必须显式包含 type，例如 widget:\"name:标题;type:input\"",
		})
		return issues
	}
	if (widgetType == widget.TypeSelect || widgetType == widget.TypeMultiSelect) &&
		parseSemicolonTag(widgetTag)["options"] == "" &&
		!strings.Contains(callbackTag, "OnSelectFuzzy") {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "warning",
			Category: "widget_select",
			Message:  "select/multiselect 字段必须有静态 options，或添加 callback:\"OnSelectFuzzy\" 并在对应 Template.OnSelectFuzzyMap 注册；纯存储外键不要写成 select",
		})
	}
	for _, badKey := range []string{"readonly", "multiple"} {
		if _, ok := parsed[badKey]; ok {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file,
				Line:     line,
				Severity: "error",
				Category: "widget_tag",
				Message:  fmt.Sprintf("widget 参数 %q 不在当前白名单中；只读用 hide 场景或回调控制，文件多选用 type:files + max_count", badKey),
			})
		}
	}
	if !widget.IsSupportedType(widgetType) {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "widget_type",
			Message:  fmt.Sprintf("unsupported widget type %q；文件上传请用 type:files，不要用 file", widgetType),
		})
	} else {
		allowedKeys := stringSetFromSlice(widget.AllowedTagKeys(widgetType))
		for key := range parsed {
			if _, ok := allowedKeys[key]; ok {
				continue
			}
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file,
				Line:     line,
				Severity: "error",
				Category: "widget_tag",
				Message:  fmt.Sprintf("widget 参数 %q 不在 %q 的白名单中；只读用 hide 场景或回调控制，文件多选用 type:files + max_count", key, widgetType),
			})
		}
	}
	if strings.Contains(callbackTag, "OnSelectFuzzy") && widgetType != widget.TypeSelect && widgetType != widget.TypeMultiSelect {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "onselect_fuzzy",
			Message:  fmt.Sprintf("callback:\"OnSelectFuzzy\" 只能用于 select 或 multiselect 字段，当前 widget type 为 %q", widgetType),
		})
	}
	if colors := parsed["options_colors"]; colors != "" {
		issues = append(issues, checkOptionsColors(file, line, parsed["options"], colors)...)
	}
	return issues
}

var hexColorRe = regexp.MustCompile(`^[0-9A-Fa-f]{6}$`)

func checkGoFileSchemaPatterns(file goSourceFileForCheck) []checkWorkspaceCodeIssue {
	fset := token.NewFileSet()
	parsed, err := parser.ParseFile(fset, file.Name, file.Content, parser.ParseComments)
	if err != nil {
		return nil
	}
	structs := collectStructTagFields(parsed, fset)
	modelNames := collectTableModelNames(parsed)
	issues := checkTableRequestModelDuplicateFields(file.Name, structs, modelNames)
	issues = append(issues, checkOnSelectFuzzyMapKeys(file.Name, structs, parsed, fset)...)
	return issues
}

type structTagField struct {
	StructName string
	FieldName  string
	Code       string
	WidgetType string
	TypeName   string
	Embedded   bool
	Line       int
}

func collectStructTagFields(parsed *ast.File, fset *token.FileSet) map[string][]structTagField {
	out := map[string][]structTagField{}
	for _, decl := range parsed.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range gen.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}
			structName := typeSpec.Name.Name
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 {
					typeName := astTypeName(field.Type)
					if typeName == "" {
						continue
					}
					out[structName] = append(out[structName], structTagField{
						StructName: structName,
						FieldName:  typeName,
						TypeName:   typeName,
						Embedded:   true,
						Line:       fset.Position(field.Pos()).Line,
					})
					continue
				}
				if field.Tag == nil {
					continue
				}
				tag := strings.Trim(field.Tag.Value, "`")
				code := firstNonEmpty(structTagCode(tag, "json"), structTagCode(tag, "form"))
				if code == "" || code == "-" {
					continue
				}
				widgetType := parseSemicolonTag(structTagValue(tag, "widget"))["type"]
				out[structName] = append(out[structName], structTagField{
					StructName: structName,
					FieldName:  field.Names[0].Name,
					Code:       code,
					WidgetType: widgetType,
					TypeName:   astTypeName(field.Type),
					Line:       fset.Position(field.Tag.Pos()).Line,
				})
			}
		}
	}
	return out
}

func collectTableModelNames(parsed *ast.File) map[string]struct{} {
	out := map[string]struct{}{}
	for _, decl := range parsed.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name == nil || fn.Name.Name != "TableName" {
			continue
		}
		for _, recv := range fn.Recv.List {
			switch expr := recv.Type.(type) {
			case *ast.Ident:
				out[expr.Name] = struct{}{}
			case *ast.StarExpr:
				if ident, ok := expr.X.(*ast.Ident); ok {
					out[ident.Name] = struct{}{}
				}
			}
		}
	}
	return out
}

func checkTableRequestModelDuplicateFields(fileName string, structs map[string][]structTagField, modelNames map[string]struct{}) []checkWorkspaceCodeIssue {
	if len(modelNames) == 0 {
		return nil
	}
	var issues []checkWorkspaceCodeIssue
	for structName, fields := range structs {
		if !strings.HasSuffix(structName, "ListReq") && !strings.HasSuffix(structName, "TableReq") {
			continue
		}
		if requestStructUsesPageSortReq(fields) {
			continue
		}
		modelCodes := modelCodesForRequest(structName, structs, modelNames)
		for _, field := range fields {
			if field.Code == "" {
				continue
			}
			modelName, exists := modelCodes[field.Code]
			if !exists {
				continue
			}
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     fileName,
				Line:     field.Line,
				Severity: "error",
				Category: "table_request_duplicate",
				Message:  fmt.Sprintf("Table Request 字段 %q 与 Model %s 的字段 code 冲突；请嵌入 query.PageSortReq，筛选字段在 Handler 中显式处理", field.Code, modelName),
			})
		}
	}
	return issues
}

func requestStructUsesPageSortReq(fields []structTagField) bool {
	usesPageSortReq := false
	for _, field := range fields {
		if !field.Embedded {
			continue
		}
		if field.TypeName == "PageSortReq" {
			usesPageSortReq = true
		}
	}
	return usesPageSortReq
}

func astTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		return t.Sel.Name
	case *ast.StarExpr:
		return astTypeName(t.X)
	default:
		return ""
	}
}

func modelCodesForRequest(requestStruct string, structs map[string][]structTagField, modelNames map[string]struct{}) map[string]string {
	base := strings.TrimSuffix(strings.TrimSuffix(requestStruct, "ListReq"), "TableReq")
	candidateNames := make([]string, 0, len(modelNames))
	if _, ok := modelNames[base]; ok {
		candidateNames = append(candidateNames, base)
	} else {
		for modelName := range modelNames {
			if strings.HasPrefix(modelName, base) || strings.HasPrefix(base, modelName) {
				candidateNames = append(candidateNames, modelName)
			}
		}
	}
	if len(candidateNames) == 0 {
		for modelName := range modelNames {
			candidateNames = append(candidateNames, modelName)
		}
	}
	modelCodes := map[string]string{}
	for _, modelName := range candidateNames {
		for _, field := range structs[modelName] {
			modelCodes[field.Code] = modelName
		}
	}
	return modelCodes
}

func checkOnSelectFuzzyMapKeys(fileName string, structs map[string][]structTagField, parsed *ast.File, fset *token.FileSet) []checkWorkspaceCodeIssue {
	fieldByCode := map[string]structTagField{}
	for _, fields := range structs {
		for _, field := range fields {
			if field.Code == "" {
				continue
			}
			if _, exists := fieldByCode[field.Code]; !exists {
				fieldByCode[field.Code] = field
			}
		}
	}
	var issues []checkWorkspaceCodeIssue
	ast.Inspect(parsed, func(n ast.Node) bool {
		litExpr, ok := n.(*ast.CompositeLit)
		if !ok || !isOnSelectFuzzyMapLiteral(litExpr) {
			return true
		}
		for _, elt := range litExpr.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			lit, ok := kv.Key.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			key := strings.Trim(lit.Value, `"`)
			if key == "" {
				continue
			}
			field, exists := fieldByCode[key]
			line := fset.Position(lit.Pos()).Line
			if !exists {
				issues = append(issues, checkWorkspaceCodeIssue{
					File:     fileName,
					Line:     line,
					Severity: "warning",
					Category: "onselect_fuzzy",
					Message:  fmt.Sprintf("OnSelectFuzzyMap key %q 未在当前文件字段中找到；确认 key 指向真实 select/multiselect 字段", key),
				})
				continue
			}
			if field.WidgetType != widget.TypeSelect && field.WidgetType != widget.TypeMultiSelect {
				issues = append(issues, checkWorkspaceCodeIssue{
					File:     fileName,
					Line:     line,
					Severity: "error",
					Category: "onselect_fuzzy",
					Message:  fmt.Sprintf("OnSelectFuzzyMap key %q 指向字段 %s.%s，但该字段 widget type 为 %q，必须是 select 或 multiselect", key, field.StructName, field.FieldName, field.WidgetType),
				})
			}
		}
		return true
	})
	return issues
}

func isOnSelectFuzzyMapLiteral(lit *ast.CompositeLit) bool {
	mapType, ok := lit.Type.(*ast.MapType)
	if !ok {
		return false
	}
	keyIdent, ok := mapType.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return false
	}
	valueSel, ok := mapType.Value.(*ast.SelectorExpr)
	if !ok || valueSel.Sel == nil || valueSel.Sel.Name != "OnSelectFuzzy" {
		return false
	}
	pkg, ok := valueSel.X.(*ast.Ident)
	return ok && pkg.Name == "app"
}

func checkOptionsColors(file string, line int, options string, colors string) []checkWorkspaceCodeIssue {
	colorParts := splitNonEmpty(colors, ",")
	var issues []checkWorkspaceCodeIssue
	for _, color := range colorParts {
		if !hexColorRe.MatchString(color) {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file,
				Line:     line,
				Severity: "error",
				Category: "options_colors",
				Message:  fmt.Sprintf("options_colors 只支持不带 # 的 6 位十六进制 RRGGBB，当前包含 %q", color),
			})
		}
	}
	optionParts := splitNonEmpty(options, ",")
	if options != "" && len(optionParts) != len(colorParts) {
		issues = append(issues, checkWorkspaceCodeIssue{
			File:     file,
			Line:     line,
			Severity: "error",
			Category: "options_colors",
			Message:  fmt.Sprintf("options_colors 数量必须和 options 一致，options=%d colors=%d", len(optionParts), len(colorParts)),
		})
	}
	return issues
}

func parseSemicolonTag(tag string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(tag, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			continue
		}
		out[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	return out
}

func invalidSemicolonTagSegments(tag string) []string {
	var invalid []string
	for _, part := range strings.Split(tag, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, _, ok := strings.Cut(part, ":")
		if !ok || strings.TrimSpace(key) == "" {
			invalid = append(invalid, part)
		}
	}
	return invalid
}

func splitNonEmpty(s string, sep string) []string {
	var out []string
	for _, part := range strings.Split(s, sep) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func stringSetFromSlice(items []string) map[string]struct{} {
	out := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out[item] = struct{}{}
		}
	}
	return out
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return ""
}

func structTagValue(tag string, key string) string {
	re := regexp.MustCompile(regexp.QuoteMeta(key) + `:"([^"]*)"`)
	match := re.FindStringSubmatch(tag)
	if len(match) != 2 {
		return ""
	}
	return match[1]
}

func structTagCode(tag string, key string) string {
	value := structTagValue(tag, key)
	if value == "" {
		return ""
	}
	code, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(code)
}
