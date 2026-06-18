package naming

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/mozillazg/go-pinyin"
	"golang.org/x/text/unicode/norm"
)

// DeriveGoPackageName turns a user-facing label such as "用户管理" into a
// deterministic Go package code such as "yong_hu_guan_li".
func DeriveGoPackageName(label, fallback string) string {
	fallback = normalizeGoPackageCodeCandidate(fallback)
	if fallback == "" {
		fallback = "directory"
	}

	code := normalizeGoPackageCodeCandidate(strings.Join(pinyinCodeTokens(label), "_"))
	if code == "" {
		code = fallback
	}
	if !startsWithASCIILetter(code) {
		code = fallback + "_" + code
	}
	if IsGoKeyword(code) {
		code = fallback + "_" + code
	}

	code = trimGoPackageCode(code, MaxGoPackageNameLength)
	if code == "" || !startsWithASCIILetter(code) {
		return fallback
	}
	if IsGoKeyword(code) {
		return trimGoPackageCode(fallback+"_"+code, MaxGoPackageNameLength)
	}
	return code
}

// GoPackageNameWithNumericSuffix appends a suffix like _2 while preserving the
// maximum Go package code length.
func GoPackageNameWithNumericSuffix(base string, index int) string {
	base = normalizeGoPackageCodeCandidate(base)
	if base == "" {
		base = "directory"
	}
	if index <= 1 {
		return trimGoPackageCode(base, MaxGoPackageNameLength)
	}

	suffix := fmt.Sprintf("_%d", index)
	maxBaseLength := MaxGoPackageNameLength - len(suffix)
	if maxBaseLength < 1 {
		maxBaseLength = 1
	}
	base = trimGoPackageCode(base, maxBaseLength)
	if base == "" {
		base = "directory"
	}
	return trimGoPackageCode(base+suffix, MaxGoPackageNameLength)
}

func pinyinCodeTokens(label string) []string {
	args := pinyin.NewArgs()
	args.Style = pinyin.Normal
	args.Heteronym = false
	args.Fallback = func(r rune, a pinyin.Args) []string {
		return nil
	}

	tokens := make([]string, 0)
	var ascii strings.Builder
	flushASCII := func() {
		if ascii.Len() == 0 {
			return
		}
		tokens = append(tokens, ascii.String())
		ascii.Reset()
	}

	for _, r := range norm.NFKD.String(label) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if isASCIIAlphaNumeric(r) {
			ascii.WriteRune(unicode.ToLower(r))
			continue
		}

		flushASCII()
		if pys := pinyin.LazyPinyin(string(r), args); len(pys) > 0 {
			token := normalizeGoPackageCodeCandidate(pys[0])
			if token != "" {
				tokens = append(tokens, token)
			}
		}
	}
	flushASCII()

	return tokens
}

func normalizeGoPackageCodeCandidate(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	var builder strings.Builder
	lastUnderscore := false
	for _, r := range norm.NFKD.String(value) {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if isASCIIAlphaNumeric(r) {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}

	return strings.Trim(builder.String(), "_")
}

func trimGoPackageCode(code string, maxLength int) string {
	if maxLength <= 0 {
		maxLength = MaxGoPackageNameLength
	}
	if len(code) > maxLength {
		code = code[:maxLength]
	}
	return strings.TrimRight(strings.Trim(code, "_"), "_")
}

func startsWithASCIILetter(value string) bool {
	if value == "" {
		return false
	}
	r := rune(value[0])
	return r >= 'a' && r <= 'z'
}

func isASCIIAlphaNumeric(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}
