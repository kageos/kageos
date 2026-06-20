package service

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	userCodePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,31}$`)
	userCodeCleaner = regexp.MustCompile(`[^a-z0-9_]+`)
)

var reservedUserCodes = map[string]struct{}{
	"admin":       {},
	"api":         {},
	"auth":        {},
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
	"login":       {},
	"logout":      {},
	"main":        {},
	"me":          {},
	"map":         {},
	"package":     {},
	"range":       {},
	"register":    {},
	"return":      {},
	"root":        {},
	"select":      {},
	"storage":     {},
	"struct":      {},
	"switch":      {},
	"system":      {},
	"type":        {},
	"user":        {},
	"var":         {},
	"workspace":   {},
}

func NormalizeUserCodeCandidate(raw string) string {
	code := strings.ToLower(strings.TrimSpace(raw))
	if index := strings.Index(code, "@"); index > 0 {
		code = code[:index]
	}
	code = strings.ReplaceAll(code, "-", "_")
	code = strings.ReplaceAll(code, ".", "_")
	code = userCodeCleaner.ReplaceAllString(code, "_")
	code = strings.Trim(code, "_")
	for strings.Contains(code, "__") {
		code = strings.ReplaceAll(code, "__", "_")
	}
	if code == "" || code[0] < 'a' || code[0] > 'z' {
		code = "u_" + code
	}
	if len(code) > 32 {
		code = strings.Trim(code[:32], "_")
	}
	if len(code) < 3 {
		code = code + strings.Repeat("0", 3-len(code))
	}
	if isReservedUserCode(code) {
		code = trimUserCodeForSuffix(code, "_user") + "_user"
	}
	return code
}

func ValidateUserCode(code string) error {
	code = strings.TrimSpace(code)
	if !userCodePattern.MatchString(code) {
		return fmt.Errorf("用户 code 只能使用 3-32 位小写字母、数字、下划线，且必须以小写字母开头")
	}
	if isReservedUserCode(code) {
		return fmt.Errorf("用户 code %q 是系统保留标识，请更换", code)
	}
	return nil
}

func isReservedUserCode(code string) bool {
	_, ok := reservedUserCodes[strings.ToLower(strings.TrimSpace(code))]
	return ok
}

func trimUserCodeForSuffix(base, suffix string) string {
	maxBaseLength := 32 - len(suffix)
	if maxBaseLength < 3 {
		maxBaseLength = 3
	}
	if len(base) > maxBaseLength {
		base = base[:maxBaseLength]
	}
	return strings.Trim(base, "_")
}
