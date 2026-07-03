package service

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/kageos/kageos/pkg/sdkmodule"
)

func checkGoFileSyntaxAndImports(file goSourceFileForCheck) []checkWorkspaceCodeIssue {
	return checkParsedGoFileSyntaxAndImports(parseGoSourceFileForCheck(file))
}

func checkParsedGoFileSyntaxAndImports(file parsedGoSourceFileForCheck) []checkWorkspaceCodeIssue {
	if file.ParseErr != nil {
		line := 0
		if pos, ok := file.ParseErr.(interface{ Pos() token.Position }); ok {
			line = pos.Pos().Line
		}
		return []checkWorkspaceCodeIssue{{
			File:     file.Source.Name,
			Line:     line,
			Severity: "error",
			Category: "go_syntax",
			Message:  file.ParseErr.Error(),
		}}
	}

	fset := file.FileSet
	parsed := file.Parsed
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

	issues := checkGoFileStructurePatterns(file.Source.Name, parsed, fset)
	for name, imp := range imports {
		if imp.Path == sdkmodule.AgentAppImport("") {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file.Source.Name,
				Line:     imp.Line,
				Severity: "warning",
				Category: "sdk_import",
				Message:  "不要导入 kageos-sdk/agent-app 根包；Context、Template、ChartType 用 kageos-sdk/agent-app/app，图表结构用 kageos-sdk/agent-app/chart，响应用 kageos-sdk/agent-app/response",
			})
		}
		if _, ok := idents[name]; !ok {
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     file.Source.Name,
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
				File:     file.Source.Name,
				Severity: "warning",
				Category: "go_import",
				Message:  fmt.Sprintf("代码使用了 %s. 但当前文件未导入对应包", root),
			})
		}
	}
	issues = append(issues, checkSDKSelectors(file.Source.Name, imports, selectorUses)...)
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
			Message:  "代码文件只有 package/import、没有有效声明，通常是空占位文件；不要创建 nps_types.go 这类文件来修 import 或跨文件问题",
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
	"github.com/kageos/kageos-sdk/agent-app/app":        "agent-app/app",
	"github.com/kageos/kageos-sdk/agent-app/callback":   "agent-app/callback",
	"github.com/kageos/kageos-sdk/agent-app/chart":      "agent-app/chart",
	"github.com/kageos/kageos-sdk/agent-app/response":   "agent-app/response",
	"github.com/kageos/kageos-sdk/agent-app/statistics": "agent-app/statistics",
	"github.com/kageos/kageos-sdk/agent-app/types":      "agent-app/types",
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
		sdkRoot, err := findSDKRootForSelectorCheck()
		if err != nil {
			sdkExportedSymbolsErr = err
			return
		}
		for importPath, relDir := range sdkPackageDirsForSelectorCheck {
			dir := filepath.Join(sdkRoot, filepath.FromSlash(relDir))
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

func findSDKRootForSelectorCheck() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if ok {
		dir := filepath.Dir(currentFile)
		for {
			if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
				sibling := filepath.Join(filepath.Dir(dir), "kageos-sdk")
				if _, err := os.Stat(filepath.Join(sibling, "agent-app")); err == nil {
					return sibling, nil
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	for _, root := range filepath.SplitList(build.Default.GOPATH) {
		if root == "" {
			continue
		}
		dir := filepath.Join(root, "pkg", "mod", sdkmodule.ModulePath+"@"+sdkmodule.Version)
		if _, err := os.Stat(filepath.Join(dir, "agent-app")); err == nil {
			return dir, nil
		}
	}

	if !ok {
		return "", fmt.Errorf("无法定位当前源码文件，也未找到 %s@%s", sdkmodule.ModulePath, sdkmodule.Version)
	}
	return "", fmt.Errorf("无法定位 %s@%s 源码", sdkmodule.ModulePath, sdkmodule.Version)
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
