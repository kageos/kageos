package service

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"

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
