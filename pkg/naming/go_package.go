package naming

import (
	"fmt"
	"regexp"
	"strings"
)

const MaxGoPackageNameLength = 50

var goPackageNameRegex = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

var goKeywords = map[string]struct{}{
	"break":       {},
	"case":        {},
	"chan":        {},
	"const":       {},
	"continue":    {},
	"default":     {},
	"defer":       {},
	"else":        {},
	"fallthrough": {},
	"for":         {},
	"func":        {},
	"go":          {},
	"goto":        {},
	"if":          {},
	"import":      {},
	"interface":   {},
	"map":         {},
	"package":     {},
	"range":       {},
	"return":      {},
	"select":      {},
	"struct":      {},
	"switch":      {},
	"type":        {},
	"var":         {},
}

func NormalizeGoPackageName(code string) string {
	return strings.TrimSpace(code)
}

func IsGoKeyword(code string) bool {
	_, ok := goKeywords[code]
	return ok
}

func ValidateGoPackageName(code, label string) error {
	return ValidateGoPackageNameLength(code, label, 1, MaxGoPackageNameLength)
}

func ValidateGoPackageNameLength(code, label string, minLength, maxLength int) error {
	label = strings.TrimSpace(label)
	if label == "" {
		label = "代码标识"
	}

	normalized := NormalizeGoPackageName(code)
	if normalized == "" {
		return fmt.Errorf("%s不能为空", label)
	}
	if code != normalized {
		return fmt.Errorf("%s不能包含首尾空格", label)
	}
	code = normalized
	if minLength > 0 && maxLength > 0 && (len(code) < minLength || len(code) > maxLength) {
		return fmt.Errorf("%s长度须为 %d-%d 个字符", label, minLength, maxLength)
	}
	if minLength > 0 && len(code) < minLength {
		return fmt.Errorf("%s长度不能少于 %d 个字符", label, minLength)
	}
	if maxLength > 0 && len(code) > maxLength {
		return fmt.Errorf("%s长度不能超过 %d 个字符", label, maxLength)
	}
	if !goPackageNameRegex.MatchString(code) {
		return fmt.Errorf("%s必须是合法的 Go package 名称：以小写字母开头，只能包含小写字母、数字和下划线，不能包含中划线", label)
	}
	if IsGoKeyword(code) {
		return fmt.Errorf("%s不能使用 Go 保留关键字：%s", label, code)
	}
	return nil
}
