package timex

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ResolveTimestampExpr 将时间表达式解析为毫秒时间戳字符串。
// 用于 run_table_search 等工具中 url_query 的 gte/lte 等参数，大模型无需手写时间戳。
//
// 支持形式（与 sdk/agent-app/widget/timestamp 约定一致）：
//   - Now()：当前时间（毫秒）
//   - Today()：今天 00:00:00（毫秒）
//   - Yesterday()：昨天 00:00:00（毫秒）
//   - Tomorrow()：明天 00:00:00（毫秒）
//   - Now(+1h)、Now(-2d)、Now(-7d) 等：相对当前时间的偏移（单位 s/h/d/w/m/y）
//   - Now(2026-02-01 13:05:05)、Now(2026-02-01)：指定日期时间，解析为本地时间（毫秒）
//
// 若 value 不是上述形式，返回 (value, false)，调用方保持原样。
func ResolveTimestampExpr(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return value, false
	}

	// Now() 当前时间
	if value == "Now()" {
		return strconv.FormatInt(time.Now().UnixMilli(), 10), true
	}
	// Today() 今天 00:00:00
	if value == "Today()" {
		t := time.Now()
		start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return strconv.FormatInt(start.UnixMilli(), 10), true
	}
	// Yesterday() 昨天 00:00:00
	if value == "Yesterday()" {
		t := time.Now().AddDate(0, 0, -1)
		start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return strconv.FormatInt(start.UnixMilli(), 10), true
	}
	// Tomorrow() 明天 00:00:00
	if value == "Tomorrow()" {
		t := time.Now().AddDate(0, 0, 1)
		start := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		return strconv.FormatInt(start.UnixMilli(), 10), true
	}

	// Now(相对偏移) 如 Now(+1h), Now(-2d), Now(-7d)
	if strings.HasPrefix(value, "Now(") && strings.HasSuffix(value, ")") {
		inner := strings.TrimSpace(value[4 : len(value)-1])
		if inner == "" {
			return strconv.FormatInt(time.Now().UnixMilli(), 10), true
		}
		// 相对偏移：+1h, -2d, -7d, +3600s 等
		if d, ok := parseRelativeDuration(inner); ok {
			return strconv.FormatInt(time.Now().Add(d).UnixMilli(), 10), true
		}
		// 绝对日期时间：2026-02-01 13:05:05 或 2026-02-01
		if ms, ok := parseAbsoluteDatetime(inner); ok {
			return strconv.FormatInt(ms, 10), true
		}
	}

	return value, false
}

// parseRelativeDuration 解析相对时间偏移，如 +1h, -2d, -7d, +3600s
func parseRelativeDuration(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}
	sign := int64(1)
	if s[0] == '+' {
		s = strings.TrimSpace(s[1:])
	} else if s[0] == '-' {
		sign = -1
		s = strings.TrimSpace(s[1:])
	}
	if s == "" {
		return 0, false
	}
	// 数字 + 可选单位（如 7d, 3600s, 1h）
	var n int64
	var unit string
	for i, r := range s {
		if r >= '0' && r <= '9' {
			continue
		}
		if i > 0 {
			n, _ = strconv.ParseInt(s[:i], 10, 64)
			unit = strings.ToLower(strings.TrimSpace(s[i:]))
		}
		break
	}
	if unit == "" {
		n, _ = strconv.ParseInt(s, 10, 64)
		unit = "h" // 默认小时
	}
	n *= sign
	var d time.Duration
	switch unit {
	case "s":
		d = time.Duration(n) * time.Second
	case "h":
		d = time.Duration(n) * time.Hour
	case "d":
		d = time.Duration(n) * 24 * time.Hour
	case "w":
		d = time.Duration(n) * 7 * 24 * time.Hour
	case "m":
		d = time.Duration(n) * 30 * 24 * time.Hour // 近似月
	case "y":
		d = time.Duration(n) * 365 * 24 * time.Hour // 近似年
	default:
		return 0, false
	}
	return d, true
}

// 支持 2026-02-01 13:05:05 或 2026-02-01
var reAbsoluteDatetime = regexp.MustCompile(`^(\d{4})-(\d{2})-(\d{2})(?:\s+(\d{1,2}):(\d{1,2})(?::(\d{1,2}))?)?$`)

func parseAbsoluteDatetime(s string) (int64, bool) {
	s = strings.TrimSpace(s)
	matches := reAbsoluteDatetime.FindStringSubmatch(s)
	if matches == nil {
		return 0, false
	}
	year, _ := strconv.Atoi(matches[1])
	month, _ := strconv.Atoi(matches[2])
	day, _ := strconv.Atoi(matches[3])
	hour, min, sec := 0, 0, 0
	if len(matches) > 4 && matches[4] != "" {
		hour, _ = strconv.Atoi(matches[4])
	}
	if len(matches) > 5 && matches[5] != "" {
		min, _ = strconv.Atoi(matches[5])
	}
	if len(matches) > 6 && matches[6] != "" {
		sec, _ = strconv.Atoi(matches[6])
	}
	t := time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
	return t.UnixMilli(), true
}

// ReplaceTimeExprsInParamValue 将 param 值串中的时间表达式替换为时间戳。
// param 值格式为 field:value 或 field1:v1,field2:v2（如 gte=created_at:Now(-7d),updated_at:Now()）。
// 只替换冒号后的 value 部分若匹配时间表达式则替换为毫秒时间戳。
func ReplaceTimeExprsInParamValue(paramValue string) string {
	// 按逗号分割成多段 field:value
	parts := strings.Split(paramValue, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		idx := strings.Index(p, ":")
		if idx <= 0 {
			out = append(out, p)
			continue
		}
		field, val := p[:idx], strings.TrimSpace(p[idx+1:])
		if resolved, ok := ResolveTimestampExpr(val); ok {
			out = append(out, field+":"+resolved)
		} else {
			out = append(out, p)
		}
	}
	return strings.Join(out, ",")
}
