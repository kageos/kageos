package sourcepolicy

import (
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const sqlitePolicyHint = "KageOS SDK 已全局注册 database/sql driver \"sqlite3\"。读取用户上传的 SQLite 文件时，请直接使用 database/sql + sql.Open(\"sqlite3\", path)；应用内置数据库请使用 ctx.GetGormDB()。不要在应用代码里额外导入或注册 sqlite3 driver，否则可能在启动时 panic: sql: Register called twice for driver sqlite3。"

var forbiddenSQLiteImports = map[string]string{
	"github.com/mattn/go-sqlite3":          "该包会注册 database/sql driver \"sqlite3\"，会和 KageOS SDK 的全局注册冲突。",
	"github.com/ncruces/go-sqlite3/driver": "KageOS SDK 已经统一导入并注册该 driver，应用代码无需再次导入。",
	"gorm.io/driver/sqlite":                "该 GORM dialect 默认依赖会注册 \"sqlite3\" 的 driver；KageOS 应用如需 GORM 请使用 ctx.GetGormDB()，读取上传 SQLite 文件请使用 database/sql + sql.Open(\"sqlite3\", path)。",
}

// ValidateAppGoSource blocks app code that would conflict with KageOS SDK's
// process-wide database/sql driver registration.
func ValidateAppGoSource(fileName, source string) error {
	if !isGoFileName(fileName) {
		return nil
	}

	fset := token.NewFileSet()
	file, parseErr := parser.ParseFile(fset, fileName, source, parser.AllErrors)
	if file == nil {
		return nil
	}

	var issues []string
	sqlImportNames := map[string]struct{}{}
	dotImportsDatabaseSQL := false

	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}
		if reason, forbidden := forbiddenSQLiteImports[importPath]; forbidden {
			issues = append(issues, fmt.Sprintf("禁止导入 %q：%s", importPath, reason))
		}
		if importPath == "database/sql" {
			if importSpec.Name != nil {
				switch importSpec.Name.Name {
				case "_":
				case ".":
					dotImportsDatabaseSQL = true
				default:
					sqlImportNames[importSpec.Name.Name] = struct{}{}
				}
			} else {
				sqlImportNames["sql"] = struct{}{}
			}
		}
	}

	if usesSQLRegister(file, sqlImportNames, dotImportsDatabaseSQL) {
		issues = append(issues, "禁止调用 database/sql.Register：应用代码不能自行注册 SQL driver。")
	}

	if len(issues) == 0 {
		return nil
	}
	if parseErr != nil {
		issues = append(issues, fmt.Sprintf("提示：源码当前还有 Go 解析错误，修正上述规范问题后仍需处理语法问题：%v", parseErr))
	}

	return fmt.Errorf("%s\n- %s\n\n%s", fileName, strings.Join(issues, "\n- "), sqlitePolicyHint)
}

// ValidateAppGoSourceDir validates non-test Go files under an app source tree.
// sourceDir may point at code/cmd/app; in that case code/ is scanned.
func ValidateAppGoSourceDir(sourceDir string) error {
	root := AppSourceRoot(sourceDir)
	buildCtx := build.Default
	buildCtx.GOOS = "linux"
	buildCtx.CgoEnabled = false

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "vendor", "workplace", "node_modules":
				return filepath.SkipDir
			}
			if strings.HasPrefix(d.Name(), ".") && filepath.Clean(path) != filepath.Clean(root) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isGoFileName(path) || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		match, err := buildCtx.MatchFile(filepath.Dir(path), filepath.Base(path))
		if err != nil {
			return fmt.Errorf("检查 Go build 约束失败 (%s): %w", path, err)
		}
		if !match {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取源码失败 (%s): %w", path, err)
		}
		if err := ValidateAppGoSource(path, string(content)); err != nil {
			return fmt.Errorf("源码规范校验失败: %w", err)
		}
		return nil
	})
}

func AppSourceRoot(sourceDir string) string {
	clean := filepath.Clean(sourceDir)
	if filepath.Base(clean) == "app" && filepath.Base(filepath.Dir(clean)) == "cmd" {
		return filepath.Dir(filepath.Dir(clean))
	}
	return clean
}

func isGoFileName(fileName string) bool {
	return strings.HasSuffix(strings.TrimSpace(fileName), ".go") || !strings.Contains(filepath.Base(fileName), ".")
}

func usesSQLRegister(file *ast.File, sqlImportNames map[string]struct{}, dotImport bool) bool {
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		if found {
			return false
		}
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := call.Fun.(type) {
		case *ast.SelectorExpr:
			if fun.Sel.Name != "Register" {
				return true
			}
			ident, ok := fun.X.(*ast.Ident)
			if !ok {
				return true
			}
			_, found = sqlImportNames[ident.Name]
			return !found
		case *ast.Ident:
			found = dotImport && fun.Name == "Register"
			return !found
		default:
			return true
		}
	})
	return found
}
