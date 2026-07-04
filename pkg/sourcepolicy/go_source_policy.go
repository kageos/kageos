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

const sqlitePolicyHint = "Kageos SDK 已全局注册 database/sql driver \"sqlite3\"。读取用户上传的 SQLite 文件时，请直接使用 database/sql + sql.Open(\"sqlite3\", path)；应用内置数据库请使用 ctx.GetGormDB()。不要在应用代码里额外导入或注册 sqlite3 driver，否则可能在启动时 panic: sql: Register called twice for driver sqlite3。"
const appDBPolicyHint = "Kageos 应用数据库安全规则：ctx.GetGormDB() 得到的数据库对象只能在当前目录业务代码内直接使用；禁止传给第三方库、外部 package、全局变量、struct 字段或 return 出去；db.Raw 仅允许字符串字面量或 const 形式的 SELECT/WITH 只读查询，用户输入必须通过 ? 参数传入；禁止 Exec/Unscoped/Migrator/DB/AutoMigrate。删除记录必须走 Table Delete / OnTableDeleteRows 受控入口，并用 UPDATE 软删除语义同时写入 deleted_at 和 deleted_by；表结构迁移由 SDK/runtime 生命周期处理。"
const sdkBoundaryPolicyHint = "Kageos 应用代码只能依赖 github.com/kageos/kageos-sdk 暴露的公共 API；禁止导入主仓库 github.com/kageos/kageos 的 sdk/pkg/dto/core 等内部实现包。"

// Keep the analyzer available, but do not enforce app DB usage restrictions by default.
const enforceAppDBPolicy = false

var forbiddenSQLiteImports = map[string]string{
	"github.com/mattn/go-sqlite3":          "该包会注册 database/sql driver \"sqlite3\"，会和 Kageos SDK 的全局注册冲突。",
	"github.com/ncruces/go-sqlite3/driver": "Kageos SDK 已经统一导入并注册该 driver，应用代码无需再次导入。",
	"gorm.io/driver/sqlite":                "该 GORM dialect 默认依赖会注册 \"sqlite3\" 的 driver；Kageos 应用如需 GORM 请使用 ctx.GetGormDB()，读取上传 SQLite 文件请使用 database/sql + sql.Open(\"sqlite3\", path)。",
}

var forbiddenAppDBMethods = map[string]string{
	"Exec":        "禁止使用 db.Exec：应用代码不能执行原始 SQL 直通，避免绕过软删除和迁移边界。",
	"Unscoped":    "禁止使用 db.Unscoped：应用记录必须保留软删除语义。",
	"Migrator":    "禁止使用 db.Migrator：表结构迁移只能由 SDK/runtime 生命周期处理。",
	"DB":          "禁止使用 db.DB：应用代码不能拿到底层 *sql.DB 后绕过 SDK/GORM 约束。",
	"AutoMigrate": "禁止在业务代码中调用 db.AutoMigrate：表结构由 Template CreateTables 和 SDK/runtime 生命周期迁移。",
}

// ValidateAppGoSource blocks app code that would conflict with Kageos SDK's
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
	var hints []string
	sqlImportNames := map[string]struct{}{}
	dotImportsDatabaseSQL := false

	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil {
			continue
		}
		if reason, forbidden := forbiddenSQLiteImports[importPath]; forbidden {
			issues = append(issues, fmt.Sprintf("禁止导入 %q：%s", importPath, reason))
			hints = appendUniqueString(hints, sqlitePolicyHint)
		}
		if reason, forbidden := forbiddenKageosImportReason(importPath); forbidden {
			issues = append(issues, fmt.Sprintf("禁止导入 %q：%s", importPath, reason))
			hints = appendUniqueString(hints, sdkBoundaryPolicyHint)
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
		hints = appendUniqueString(hints, sqlitePolicyHint)
	}

	if enforceAppDBPolicy {
		appDBIssues := findAppDBPolicyIssues(file)
		if len(appDBIssues) > 0 {
			issues = append(issues, appDBIssues...)
			hints = appendUniqueString(hints, appDBPolicyHint)
		}
	}

	if len(issues) == 0 {
		return nil
	}
	if parseErr != nil {
		issues = append(issues, fmt.Sprintf("提示：源码当前还有 Go 解析错误，修正上述规范问题后仍需处理语法问题：%v", parseErr))
	}

	return fmt.Errorf("%s\n- %s%s", fileName, strings.Join(issues, "\n- "), formatPolicyHints(hints))
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

type appDBPolicyAnalyzer struct {
	gormImportNames map[string]struct{}
	localFuncs      map[string]*ast.FuncDecl
	globalVars      map[string]struct{}
	stringConsts    map[string]string
	appDBNames      map[string]struct{}
	issues          []string
}

func findAppDBPolicyIssues(file *ast.File) []string {
	analyzer := &appDBPolicyAnalyzer{
		gormImportNames: map[string]struct{}{},
		localFuncs:      map[string]*ast.FuncDecl{},
		globalVars:      map[string]struct{}{},
		stringConsts:    map[string]string{},
		appDBNames:      map[string]struct{}{},
	}
	analyzer.collectImportsAndFunctions(file)
	ast.Inspect(file, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			analyzer.recordGORMDBParams(n.Type.Params)
		case *ast.FuncLit:
			analyzer.recordGORMDBParams(n.Type.Params)
		case *ast.ValueSpec:
			analyzer.inspectValueSpec(n)
		case *ast.AssignStmt:
			analyzer.inspectAssignStmt(n)
		case *ast.ReturnStmt:
			analyzer.inspectReturnStmt(n)
		case *ast.CallExpr:
			analyzer.inspectCallExpr(n)
		case *ast.CompositeLit:
			analyzer.inspectCompositeLit(n)
		}
		return true
	})
	return dedupeStrings(analyzer.issues)
}

func (a *appDBPolicyAnalyzer) collectImportsAndFunctions(file *ast.File) {
	for _, importSpec := range file.Imports {
		importPath, err := strconv.Unquote(importSpec.Path.Value)
		if err != nil || importPath != "gorm.io/gorm" {
			continue
		}
		name := "gorm"
		if importSpec.Name != nil && importSpec.Name.Name != "" && importSpec.Name.Name != "_" && importSpec.Name.Name != "." {
			name = importSpec.Name.Name
		}
		a.gormImportNames[name] = struct{}{}
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if d.Name != nil {
				a.localFuncs[d.Name.Name] = d
			}
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				switch d.Tok {
				case token.CONST:
					a.collectStringConsts(valueSpec)
				case token.VAR:
					for _, name := range valueSpec.Names {
						if name != nil && name.Name != "_" {
							a.globalVars[name.Name] = struct{}{}
						}
					}
				}
			}
		}
	}
}

func forbiddenKageosImportReason(importPath string) (string, bool) {
	switch {
	case importPath == "github.com/kageos/kageos/sdk/agent-app" || strings.HasPrefix(importPath, "github.com/kageos/kageos/sdk/agent-app/"):
		return "请改用 github.com/kageos/kageos-sdk/agent-app/...；主仓库内置 SDK 已停止作为应用公共 API。", true
	case strings.HasPrefix(importPath, "github.com/kageos/kageos/pkg/"):
		return "请改用 github.com/kageos/kageos-sdk/pkg/... 中明确暴露的公共包；主仓库 pkg 是平台内部实现。", true
	case importPath == "github.com/kageos/kageos/dto" || strings.HasPrefix(importPath, "github.com/kageos/kageos/dto/"):
		return "请改用 github.com/kageos/kageos-sdk/dto；主仓库 dto 是平台内部实现。", true
	case strings.HasPrefix(importPath, "github.com/kageos/kageos/core/"):
		return "core 是平台服务端内部实现，应用代码不能导入。", true
	default:
		return "", false
	}
}

func (a *appDBPolicyAnalyzer) recordGORMDBParams(fields *ast.FieldList) {
	if fields == nil {
		return
	}
	for _, field := range fields.List {
		if !a.isGORMDBType(field.Type) {
			continue
		}
		for _, name := range field.Names {
			if name != nil && name.Name != "_" {
				a.trackAppDBName(name.Name)
			}
		}
	}
}

func (a *appDBPolicyAnalyzer) collectStringConsts(spec *ast.ValueSpec) {
	for i, name := range spec.Names {
		if name == nil || name.Name == "_" || i >= len(spec.Values) {
			continue
		}
		value, ok := stringLiteralValue(spec.Values[i])
		if !ok {
			continue
		}
		a.stringConsts[name.Name] = value
	}
}

func (a *appDBPolicyAnalyzer) inspectCompositeLit(lit *ast.CompositeLit) {
	for _, elt := range lit.Elts {
		expr := elt
		if kv, ok := elt.(*ast.KeyValueExpr); ok {
			expr = kv.Value
		}
		if a.isAppDBExpr(expr) {
			a.addIssue("禁止把应用数据库对象保存到 struct/map/slice 字面量；数据库连接能力不能长期持有或转交。")
		}
	}
}

func (a *appDBPolicyAnalyzer) inspectValueSpec(spec *ast.ValueSpec) {
	for i, value := range spec.Values {
		if !a.isAppDBExpr(value) {
			continue
		}
		if len(spec.Names) == 0 {
			continue
		}
		if len(spec.Values) == 1 {
			a.trackAppDBName(spec.Names[0].Name)
			continue
		}
		if i < len(spec.Names) {
			a.trackAppDBName(spec.Names[i].Name)
		}
	}
}

func (a *appDBPolicyAnalyzer) inspectAssignStmt(stmt *ast.AssignStmt) {
	for i, rhs := range stmt.Rhs {
		if !a.isAppDBExpr(rhs) {
			continue
		}
		if len(stmt.Rhs) == 1 {
			a.inspectAppDBAssignmentTargets(stmt.Lhs, rhs)
			continue
		}
		if i < len(stmt.Lhs) {
			a.inspectAppDBAssignmentTargets([]ast.Expr{stmt.Lhs[i]}, rhs)
		}
	}
}

func (a *appDBPolicyAnalyzer) inspectAppDBAssignmentTargets(lhs []ast.Expr, rhs ast.Expr) {
	if len(lhs) == 0 {
		return
	}
	for i, target := range lhs {
		if i > 0 && isGetGormDBCall(rhs) {
			continue
		}
		switch t := unwrapExpr(target).(type) {
		case *ast.Ident:
			if _, global := a.globalVars[t.Name]; global {
				a.addIssue("禁止把应用数据库对象保存到全局变量；请在当前函数内直接使用 ctx.GetGormDB()。")
				continue
			}
			if t.Name != "_" {
				a.trackAppDBName(t.Name)
			}
		case *ast.SelectorExpr:
			a.addIssue("禁止把应用数据库对象保存到 struct 字段或对象属性；请在当前函数内直接使用 ctx.GetGormDB()。")
		case *ast.IndexExpr:
			a.addIssue("禁止把应用数据库对象保存到 map/slice 等容器；数据库连接能力不能长期持有或转交。")
		default:
			a.addIssue("禁止把应用数据库对象保存到非局部变量目标；请在当前函数内直接使用。")
		}
	}
}

func (a *appDBPolicyAnalyzer) inspectReturnStmt(stmt *ast.ReturnStmt) {
	for _, result := range stmt.Results {
		if a.isAppDBExpr(result) {
			a.addIssue("禁止 return 应用数据库对象；ctx.GetGormDB() 的能力不能离开当前业务函数/本地 helper。")
		}
	}
}

func (a *appDBPolicyAnalyzer) inspectCallExpr(call *ast.CallExpr) {
	if method, ok := a.appDBMethodName(call.Fun); ok {
		if method == "Raw" {
			for _, issue := range runtimeAppDBRawSQLPolicy.ValidateCall(call, a.stringConsts) {
				a.addIssue(issue)
			}
			return
		}
		if reason, forbidden := forbiddenAppDBMethods[method]; forbidden {
			a.addIssue(reason)
		}
		return
	}
	if !a.callHasAppDBArg(call) {
		return
	}
	if a.allowsAppDBArg(call) {
		return
	}
	a.addIssue("禁止把应用数据库对象传给第三方库、外部 package 或未知函数；请在当前目录业务代码内直接使用 GORM 链式调用，必要时只传给同文件显式接收 *gorm.DB 的本地 helper。")
}

func (a *appDBPolicyAnalyzer) callHasAppDBArg(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if a.isAppDBExpr(arg) {
			return true
		}
	}
	return false
}

func (a *appDBPolicyAnalyzer) allowsAppDBArg(call *ast.CallExpr) bool {
	switch fun := unwrapExpr(call.Fun).(type) {
	case *ast.Ident:
		decl := a.localFuncs[fun.Name]
		return decl != nil && a.callDBArgsMatchGORMParams(call, decl.Type.Params)
	case *ast.FuncLit:
		return a.callDBArgsMatchGORMParams(call, fun.Type.Params)
	default:
		return false
	}
}

func (a *appDBPolicyAnalyzer) callDBArgsMatchGORMParams(call *ast.CallExpr, params *ast.FieldList) bool {
	paramTypes := flattenParamTypes(params)
	if len(paramTypes) == 0 {
		return false
	}
	for i, arg := range call.Args {
		if !a.isAppDBExpr(arg) {
			continue
		}
		paramIndex := i
		if paramIndex >= len(paramTypes) {
			paramIndex = len(paramTypes) - 1
		}
		if !a.isGORMDBType(paramTypes[paramIndex]) {
			return false
		}
	}
	return true
}

func flattenParamTypes(params *ast.FieldList) []ast.Expr {
	if params == nil {
		return nil
	}
	var out []ast.Expr
	for _, field := range params.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		for i := 0; i < count; i++ {
			out = append(out, field.Type)
		}
	}
	return out
}

func (a *appDBPolicyAnalyzer) appDBMethodName(expr ast.Expr) (string, bool) {
	sel, ok := unwrapExpr(expr).(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return "", false
	}
	if a.isAppDBRootExpr(sel.X) {
		return sel.Sel.Name, true
	}
	return "", false
}

func (a *appDBPolicyAnalyzer) isAppDBExpr(expr ast.Expr) bool {
	expr = unwrapExpr(expr)
	if isGetGormDBCall(expr) {
		return true
	}
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := a.appDBNames[e.Name]
		return ok
	case *ast.CallExpr:
		if len(e.Args) == 1 && a.isAppDBExpr(e.Args[0]) {
			switch unwrapExpr(e.Fun).(type) {
			case *ast.Ident, *ast.InterfaceType:
				return true
			}
		}
		_, ok := a.appDBMethodName(e.Fun)
		return ok
	case *ast.SelectorExpr:
		if e.Sel != nil && (e.Sel.Name == "Error" || e.Sel.Name == "RowsAffected" || e.Sel.Name == "Statement") {
			return false
		}
		return a.isAppDBRootExpr(e.X)
	default:
		return false
	}
}

func (a *appDBPolicyAnalyzer) isAppDBRootExpr(expr ast.Expr) bool {
	expr = unwrapExpr(expr)
	if isGetGormDBCall(expr) {
		return true
	}
	switch e := expr.(type) {
	case *ast.Ident:
		_, ok := a.appDBNames[e.Name]
		return ok
	case *ast.CallExpr:
		_, ok := a.appDBMethodName(e.Fun)
		return ok
	case *ast.SelectorExpr:
		return a.isAppDBRootExpr(e.X)
	default:
		return false
	}
}

func (a *appDBPolicyAnalyzer) isGORMDBType(expr ast.Expr) bool {
	expr = unwrapExpr(expr)
	if star, ok := expr.(*ast.StarExpr); ok {
		expr = unwrapExpr(star.X)
	}
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil || sel.Sel.Name != "DB" {
		return false
	}
	ident, ok := unwrapExpr(sel.X).(*ast.Ident)
	if !ok {
		return false
	}
	_, ok = a.gormImportNames[ident.Name]
	return ok
}

func (a *appDBPolicyAnalyzer) trackAppDBName(name string) {
	if strings.TrimSpace(name) == "" || name == "_" {
		return
	}
	a.appDBNames[name] = struct{}{}
}

func (a *appDBPolicyAnalyzer) addIssue(issue string) {
	if strings.TrimSpace(issue) == "" {
		return
	}
	a.issues = append(a.issues, issue)
}

func isGetGormDBCall(expr ast.Expr) bool {
	call, ok := unwrapExpr(expr).(*ast.CallExpr)
	if !ok || len(call.Args) != 0 {
		return false
	}
	sel, ok := unwrapExpr(call.Fun).(*ast.SelectorExpr)
	return ok && sel.Sel != nil && sel.Sel.Name == "GetGormDB"
}

func unwrapExpr(expr ast.Expr) ast.Expr {
	for {
		switch e := expr.(type) {
		case *ast.ParenExpr:
			expr = e.X
		case *ast.IndexExpr:
			return e
		default:
			return expr
		}
	}
}

func appendUniqueString(values []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return values
	}
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func formatPolicyHints(hints []string) string {
	hints = dedupeStrings(hints)
	if len(hints) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(hints, "\n\n")
}
