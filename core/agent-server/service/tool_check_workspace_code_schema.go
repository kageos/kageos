package service

import (
	"fmt"
	"go/ast"
	"go/token"
	"strings"

	"github.com/kageos/kageos-sdk/agent-app/widget"
)

func checkGoFileSchemaPatterns(file goSourceFileForCheck) []checkWorkspaceCodeIssue {
	return checkParsedGoFileSchemaPatterns(parseGoSourceFileForCheck(file))
}

func checkParsedGoFileSchemaPatterns(file parsedGoSourceFileForCheck) []checkWorkspaceCodeIssue {
	if file.ParseErr != nil {
		return nil
	}
	structs := collectStructTagFields(file.Parsed, file.FileSet)
	modelNames := collectTableModelNames(file.Parsed)
	issues := checkTableRequestModelDuplicateFields(file.Source.Name, structs, modelNames)
	issues = append(issues, checkWidgetGoTypeCompatibility(file.Source.Name, structs)...)
	issues = append(issues, checkOnSelectFuzzyMapKeys(file.Source.Name, structs, file.Parsed, file.FileSet)...)
	return issues
}

type structTagField struct {
	StructName string
	FieldName  string
	Code       string
	WidgetType string
	TypeName   string
	TypeExpr   string
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
					TypeExpr:   astTypeExpr(field.Type),
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

func checkWidgetGoTypeCompatibility(fileName string, structs map[string][]structTagField) []checkWorkspaceCodeIssue {
	var issues []checkWorkspaceCodeIssue
	for _, fields := range structs {
		for _, field := range fields {
			if field.Embedded || field.WidgetType == "" || field.TypeExpr == "" {
				continue
			}
			message := widgetGoTypeCompatibilityMessage(field)
			if message == "" {
				continue
			}
			issues = append(issues, checkWorkspaceCodeIssue{
				File:     fileName,
				Line:     field.Line,
				Severity: "error",
				Category: "widget_go_type",
				Message:  message,
			})
		}
	}
	return issues
}

func widgetGoTypeCompatibilityMessage(field structTagField) string {
	typeExpr := field.TypeExpr
	switch field.WidgetType {
	case widget.TypeFiles:
		if isGoSliceOrArrayType(typeExpr) || isKnownNonStringScalarType(typeExpr) {
			return fmt.Sprintf("字段 %s 使用 type:files 时 Go 类型必须是 string；多文件也用逗号分隔 refs 字符串，不要用 %s", field.FieldName, typeExpr)
		}
	case widget.TypeInteger:
		if isGoFloatType(typeExpr) || isGoStringType(typeExpr) || isGoBoolType(typeExpr) || isGoSliceOrArrayType(typeExpr) {
			return fmt.Sprintf("字段 %s 使用 type:integer 时 Go 类型必须是整数；小数请改用 type:float，当前是 %s", field.FieldName, typeExpr)
		}
	case widget.TypeFloat:
		if isGoIntegerType(typeExpr) || isGoStringType(typeExpr) || isGoBoolType(typeExpr) || isGoSliceOrArrayType(typeExpr) {
			return fmt.Sprintf("字段 %s 使用 type:float 时 Go 类型必须是 float32/float64；整数请改用 type:integer，当前是 %s", field.FieldName, typeExpr)
		}
	case widget.TypeSwitch:
		if isKnownNonBoolType(typeExpr) {
			return fmt.Sprintf("字段 %s 使用 type:switch 时 Go 类型必须是 bool，当前是 %s；字符串枚举请用 select/radio", field.FieldName, typeExpr)
		}
	case widget.TypeInput, widget.TypeText, widget.TypeTextArea, widget.TypeRichText, widget.TypeColor, widget.TypeLink, widget.TypeUser, widget.TypeDepartment:
		if isGoSliceOrArrayType(typeExpr) || isKnownNonStringScalarType(typeExpr) {
			return fmt.Sprintf("字段 %s 使用 type:%s 时 Go 类型必须是 string，当前是 %s", field.FieldName, field.WidgetType, typeExpr)
		}
	}
	return ""
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

func astTypeExpr(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		prefix := astTypeExpr(t.X)
		if prefix == "" {
			return t.Sel.Name
		}
		return prefix + "." + t.Sel.Name
	case *ast.StarExpr:
		inner := astTypeExpr(t.X)
		if inner == "" {
			return ""
		}
		return "*" + inner
	case *ast.ArrayType:
		inner := astTypeExpr(t.Elt)
		if inner == "" {
			return ""
		}
		return "[]" + inner
	default:
		return ""
	}
}

func derefGoTypeExpr(typeExpr string) string {
	typeExpr = strings.TrimSpace(typeExpr)
	for strings.HasPrefix(typeExpr, "*") {
		typeExpr = strings.TrimPrefix(typeExpr, "*")
	}
	return typeExpr
}

func isGoSliceOrArrayType(typeExpr string) bool {
	return strings.HasPrefix(derefGoTypeExpr(typeExpr), "[]")
}

func isGoStringType(typeExpr string) bool {
	return derefGoTypeExpr(typeExpr) == "string"
}

func isGoBoolType(typeExpr string) bool {
	return derefGoTypeExpr(typeExpr) == "bool"
}

func isGoIntegerType(typeExpr string) bool {
	switch derefGoTypeExpr(typeExpr) {
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64", "uintptr":
		return true
	default:
		return false
	}
}

func isGoFloatType(typeExpr string) bool {
	switch derefGoTypeExpr(typeExpr) {
	case "float32", "float64":
		return true
	default:
		return false
	}
}

func isKnownNonStringScalarType(typeExpr string) bool {
	return isGoIntegerType(typeExpr) || isGoFloatType(typeExpr) || isGoBoolType(typeExpr)
}

func isKnownNonBoolType(typeExpr string) bool {
	return isGoStringType(typeExpr) || isGoIntegerType(typeExpr) || isGoFloatType(typeExpr) || isGoSliceOrArrayType(typeExpr)
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
