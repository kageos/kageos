package service

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNamespaceManifestScheduleDefaultsAreExplicit(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", "namespace"))
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			t.Skip("namespace workspace sources are not present in this checkout")
		}
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var checkedFormSchedules int
	var checkedAgentTasks int

	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != root && strings.HasPrefix(entry.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(data)
		if !strings.Contains(source, "app.FormSchedule") && !strings.Contains(source, "app.AgentTask") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, data, 0)
		if err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			literal, ok := node.(*ast.CompositeLit)
			if !ok {
				return true
			}

			if isAppSelector(literal.Type, "AgentTask") {
				checkedAgentTasks++
				assertManifestEnabledBool(t, fset, path, literal, false, "AgentTask")
				return true
			}

			arrayType, ok := literal.Type.(*ast.ArrayType)
			if !ok || !isAppSelector(arrayType.Elt, "FormSchedule") {
				return true
			}
			for _, element := range literal.Elts {
				schedule, ok := element.(*ast.CompositeLit)
				if !ok {
					continue
				}
				checkedFormSchedules++
				assertManifestEnabledBool(t, fset, path, schedule, true, "FormSchedule")
			}
			return true
		})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checkedFormSchedules == 0 || checkedAgentTasks == 0 {
		t.Skipf("no namespace manifest schedules found, checked FormSchedule=%d AgentTask=%d", checkedFormSchedules, checkedAgentTasks)
	}
}

func isAppSelector(expr ast.Expr, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != name {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "app"
}

func assertManifestEnabledBool(t *testing.T, fset *token.FileSet, path string, literal *ast.CompositeLit, want bool, kind string) {
	t.Helper()
	for _, element := range literal.Elts {
		field, ok := element.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key, ok := field.Key.(*ast.Ident)
		if !ok || key.Name != "Enabled" {
			continue
		}
		value, ok := field.Value.(*ast.Ident)
		if !ok || value.Name != fmt.Sprintf("%t", want) {
			position := fset.Position(field.Value.Pos())
			t.Errorf("%s:%d %s must declare Enabled: %t", path, position.Line, kind, want)
		}
		return
	}
	position := fset.Position(literal.Pos())
	t.Errorf("%s:%d %s must explicitly declare Enabled: %t", path, position.Line, kind, want)
}
