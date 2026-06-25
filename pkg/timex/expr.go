package timex

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

const dateTimeLayout = "2006-01-02 15:04:05"

// ResolveDateTimeExpr 将 SQL 风格时间表达式解析为 "YYYY-MM-DD HH:mm:ss" 字符串。
//
// 支持形式：
//   - CURRENT_TIMESTAMP、CURRENT_DATE
//   - DATE_ADD(CURRENT_TIMESTAMP, INTERVAL 1 HOUR)
//   - DATE_SUB(CURRENT_DATE, INTERVAL 7 DAY)
func ResolveDateTimeExpr(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return value, false
	}

	if parsed, ok := resolveSQLTemporalExpr(value); ok {
		return parsed.Format(dateTimeLayout), true
	}

	return value, false
}

func resolveSQLTemporalExpr(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}

	if parsed, ok := resolveSQLBaseTime(value); ok {
		return parsed, true
	}

	name, args, ok := parseFunctionArgs(value)
	if !ok || len(args) != 2 {
		return time.Time{}, false
	}

	name = strings.ToUpper(name)
	if name != "DATE_ADD" && name != "DATE_SUB" {
		return time.Time{}, false
	}

	base, ok := resolveSQLBaseTime(args[0])
	if !ok {
		return time.Time{}, false
	}
	amount, unit, ok := parseSQLInterval(args[1])
	if !ok {
		return time.Time{}, false
	}
	if name == "DATE_SUB" {
		amount = -amount
	}

	return addSQLInterval(base, amount, unit), true
}

func resolveSQLBaseTime(value string) (time.Time, bool) {
	keyword := strings.ToUpper(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
	now := time.Now()

	switch keyword {
	case "CURRENT_TIMESTAMP", "CURRENT_TIMESTAMP()":
		return now, true
	case "CURRENT_DATE", "CURRENT_DATE()":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), true
	default:
		return time.Time{}, false
	}
}

var reSQLInterval = regexp.MustCompile(`(?i)^INTERVAL\s+([+-]?\d+)\s+(SECOND|MINUTE|HOUR|DAY|WEEK|MONTH|YEAR)S?$`)

func parseSQLInterval(value string) (int, string, bool) {
	matches := reSQLInterval.FindStringSubmatch(strings.TrimSpace(value))
	if matches == nil {
		return 0, "", false
	}
	amount, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, "", false
	}
	return amount, strings.ToUpper(matches[2]), true
}

func addSQLInterval(base time.Time, amount int, unit string) time.Time {
	switch strings.ToUpper(unit) {
	case "SECOND":
		return base.Add(time.Duration(amount) * time.Second)
	case "MINUTE":
		return base.Add(time.Duration(amount) * time.Minute)
	case "HOUR":
		return base.Add(time.Duration(amount) * time.Hour)
	case "DAY":
		return base.AddDate(0, 0, amount)
	case "WEEK":
		return base.AddDate(0, 0, amount*7)
	case "MONTH":
		return base.AddDate(0, amount, 0)
	case "YEAR":
		return base.AddDate(amount, 0, 0)
	default:
		return base
	}
}

func parseFunctionArgs(value string) (string, []string, bool) {
	value = strings.TrimSpace(value)
	open := strings.Index(value, "(")
	if open <= 0 || !strings.HasSuffix(value, ")") {
		return "", nil, false
	}
	name := strings.TrimSpace(value[:open])
	if name == "" {
		return "", nil, false
	}
	argsRaw := strings.TrimSpace(value[open+1 : len(value)-1])
	args := splitCommaOutsideParens(argsRaw)
	for i := range args {
		args[i] = strings.TrimSpace(args[i])
	}
	return name, args, true
}

func splitCommaOutsideParens(value string) []string {
	parts := make([]string, 0, 2)
	start := 0
	depth := 0
	var quote rune

	for i, r := range value {
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		switch r {
		case '\'', '"':
			quote = r
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				parts = append(parts, value[start:i])
				start = i + len(string(r))
			}
		}
	}
	parts = append(parts, value[start:])
	return parts
}

// ReplaceTimeExprsInParamValue 将 query 参数值里的 SQL 风格时间表达式替换为
// "YYYY-MM-DD HH:mm:ss"，适用于 datetime 字段。
func ReplaceTimeExprsInParamValue(paramValue string) string {
	if resolved, ok := ResolveDateTimeExpr(paramValue); ok {
		return resolved
	}
	return paramValue
}
