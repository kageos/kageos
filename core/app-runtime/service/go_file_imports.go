package service

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
	"golang.org/x/tools/imports"
)

func addNamedImportToGoFile(filePath, name, importPath string) (bool, error) {
	return rewriteGoFileImports(filePath, func(fset *token.FileSet, file *ast.File) (bool, error) {
		return astutil.AddNamedImport(fset, file, name, importPath), nil
	})
}

func removeNamedImportFromGoFile(filePath, name, importPath string) (bool, error) {
	return rewriteGoFileImports(filePath, func(fset *token.FileSet, file *ast.File) (bool, error) {
		return astutil.DeleteNamedImport(fset, file, name, importPath), nil
	})
}

func removeNamedImportsWithPathPrefixFromGoFile(filePath, name, importPathPrefix string) (bool, error) {
	importPathPrefix = strings.TrimRight(strings.TrimSpace(importPathPrefix), "/")
	if importPathPrefix == "" {
		return false, nil
	}
	return rewriteGoFileImports(filePath, func(fset *token.FileSet, file *ast.File) (bool, error) {
		paths := make([]string, 0)
		for _, spec := range file.Imports {
			if !importNameMatches(spec, name) {
				continue
			}
			importPath, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				continue
			}
			if importPath == importPathPrefix || strings.HasPrefix(importPath, importPathPrefix+"/") {
				paths = append(paths, importPath)
			}
		}
		changed := false
		for _, importPath := range paths {
			if astutil.DeleteNamedImport(fset, file, name, importPath) {
				changed = true
			}
		}
		return changed, nil
	})
}

func importNameMatches(spec *ast.ImportSpec, name string) bool {
	if spec == nil {
		return false
	}
	name = strings.TrimSpace(name)
	if spec.Name == nil {
		return name == ""
	}
	return spec.Name.Name == name
}

func rewriteGoFileImports(filePath string, edit func(fset *token.FileSet, file *ast.File) (bool, error)) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("read go file: %w", err)
	}

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, content, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("parse go file: %w", err)
	}

	changed, err := edit(fset, file)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}

	ast.SortImports(fset, file)

	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return false, fmt.Errorf("format go file: %w", err)
	}

	formatted, err := imports.Process(filePath, buf.Bytes(), &imports.Options{
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	})
	if err != nil {
		return false, fmt.Errorf("process imports: %w", err)
	}

	if err := writeFileAtomic(filePath, formatted, 0644); err != nil {
		return false, fmt.Errorf("write go file: %w", err)
	}

	return true, nil
}
