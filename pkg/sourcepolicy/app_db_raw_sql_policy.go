package sourcepolicy

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"
)

// runtimeAppDBRawSQLPolicy is the single extension point for app db Raw SQL constraints.
var runtimeAppDBRawSQLPolicy = newReadOnlyAppDBRawSQLPolicy()

type appDBRawSQLPolicy struct {
	requireStaticSQL     bool
	allowedFirstKeywords map[string]struct{}
	forbiddenKeywords    map[string]struct{}
}

func newReadOnlyAppDBRawSQLPolicy() appDBRawSQLPolicy {
	return appDBRawSQLPolicy{
		requireStaticSQL:     true,
		allowedFirstKeywords: stringSet("select", "with"),
		forbiddenKeywords: stringSet(
			"alter",
			"analyze",
			"call",
			"create",
			"delete",
			"drop",
			"grant",
			"insert",
			"lock",
			"merge",
			"optimize",
			"rename",
			"replace",
			"revoke",
			"set",
			"truncate",
			"unlock",
			"update",
		),
	}
}

func (p appDBRawSQLPolicy) ValidateCall(call *ast.CallExpr, stringConsts map[string]string) []string {
	if len(call.Args) == 0 {
		return []string{"db.Raw 只允许用于 SELECT/WITH 只读查询，并且第一个 SQL 参数必须是字符串字面量或 const。"}
	}
	sqlText, ok := p.staticSQLText(call.Args[0], stringConsts)
	if p.requireStaticSQL && !ok {
		return []string{"db.Raw 的 SQL 必须是字符串字面量或顶层 const；用户输入必须通过 ? 参数传入，不能拼接 SQL。"}
	}
	if !ok {
		return nil
	}
	if issue := p.ValidateSQL(sqlText); issue != "" {
		return []string{issue}
	}
	return nil
}

func (p appDBRawSQLPolicy) ValidateSQL(sqlText string) string {
	tokens := sqlKeywordTokens(stripLeadingSQLComments(sqlText))
	if len(tokens) == 0 {
		return "db.Raw 只允许 SELECT/WITH 只读查询；写入、DDL、会话控制或维护语句请使用 GORM/SDK 受控能力。"
	}
	if _, ok := p.allowedFirstKeywords[tokens[0]]; !ok {
		return "db.Raw 只允许 SELECT/WITH 只读查询；写入、DDL、会话控制或维护语句请使用 GORM/SDK 受控能力。"
	}
	for _, token := range tokens {
		if _, forbidden := p.forbiddenKeywords[token]; forbidden {
			return "db.Raw 只允许 SELECT/WITH 只读查询；写入、DDL、会话控制或维护语句请使用 GORM/SDK 受控能力。"
		}
	}
	return ""
}

func (p appDBRawSQLPolicy) staticSQLText(expr ast.Expr, stringConsts map[string]string) (string, bool) {
	if value, ok := stringLiteralValue(expr); ok {
		return value, true
	}
	ident, ok := unwrapExpr(expr).(*ast.Ident)
	if !ok {
		return "", false
	}
	value, ok := stringConsts[ident.Name]
	return value, ok
}

func stringLiteralValue(expr ast.Expr) (string, bool) {
	lit, ok := unwrapExpr(expr).(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return value, true
}

func stripLeadingSQLComments(sqlText string) string {
	for {
		sqlText = strings.TrimSpace(sqlText)
		switch {
		case strings.HasPrefix(sqlText, "--"):
			idx := strings.IndexByte(sqlText, '\n')
			if idx < 0 {
				return ""
			}
			sqlText = sqlText[idx+1:]
		case strings.HasPrefix(sqlText, "#"):
			idx := strings.IndexByte(sqlText, '\n')
			if idx < 0 {
				return ""
			}
			sqlText = sqlText[idx+1:]
		case strings.HasPrefix(sqlText, "/*"):
			idx := strings.Index(sqlText, "*/")
			if idx < 0 {
				return ""
			}
			sqlText = sqlText[idx+2:]
		default:
			return sqlText
		}
	}
}

func sqlKeywordTokens(sqlText string) []string {
	sqlText = stripSQLQuotedTextAndComments(sqlText)
	var tokens []string
	start := -1
	for i, r := range sqlText {
		if isSQLTokenRune(r) {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			tokens = append(tokens, strings.ToLower(sqlText[start:i]))
			start = -1
		}
	}
	if start >= 0 {
		tokens = append(tokens, strings.ToLower(sqlText[start:]))
	}
	return tokens
}

func stripSQLQuotedTextAndComments(sqlText string) string {
	var out strings.Builder
	for i := 0; i < len(sqlText); {
		switch sqlText[i] {
		case '\'', '"', '`':
			i = skipSQLQuoted(sqlText, i, sqlText[i])
			out.WriteByte(' ')
		case '-':
			if i+1 < len(sqlText) && sqlText[i+1] == '-' {
				i = skipUntilSQLNewline(sqlText, i+2)
				out.WriteByte(' ')
				continue
			}
			out.WriteByte(sqlText[i])
			i++
		case '#':
			i = skipUntilSQLNewline(sqlText, i+1)
			out.WriteByte(' ')
		case '/':
			if i+1 < len(sqlText) && sqlText[i+1] == '*' {
				i = skipSQLBlockComment(sqlText, i+2)
				out.WriteByte(' ')
				continue
			}
			out.WriteByte(sqlText[i])
			i++
		default:
			out.WriteByte(sqlText[i])
			i++
		}
	}
	return out.String()
}

func skipSQLQuoted(sqlText string, start int, quote byte) int {
	for i := start + 1; i < len(sqlText); i++ {
		if sqlText[i] == '\\' {
			i++
			continue
		}
		if sqlText[i] != quote {
			continue
		}
		if i+1 < len(sqlText) && sqlText[i+1] == quote {
			i++
			continue
		}
		return i + 1
	}
	return len(sqlText)
}

func skipUntilSQLNewline(sqlText string, start int) int {
	for i := start; i < len(sqlText); i++ {
		if sqlText[i] == '\n' || sqlText[i] == '\r' {
			return i + 1
		}
	}
	return len(sqlText)
}

func skipSQLBlockComment(sqlText string, start int) int {
	idx := strings.Index(sqlText[start:], "*/")
	if idx < 0 {
		return len(sqlText)
	}
	return start + idx + len("*/")
}

func isSQLTokenRune(r rune) bool {
	return r == '_' || r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z'
}

func stringSet(values ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" {
			continue
		}
		out[value] = struct{}{}
	}
	return out
}
