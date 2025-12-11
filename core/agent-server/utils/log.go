package utils

import (
	"fmt"
)

const (
	// MaxLogLength 日志最大长度，超过此长度会被截断
	MaxLogLength = 500
)

// TruncateString 截断字符串，如果超过最大长度，只显示前N个字符和后N个字符，中间用省略号
func TruncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = MaxLogLength
	}
	if len(s) <= maxLen {
		return s
	}
	// 如果超过最大长度，显示前200个字符和后200个字符
	prefixLen := 200
	suffixLen := 200
	if maxLen < prefixLen+suffixLen {
		// 如果最大长度太小，只显示前缀
		return s[:maxLen] + "...(已截断)"
	}
	return s[:prefixLen] + fmt.Sprintf("...(已截断，总长度:%d)...", len(s)) + s[len(s)-suffixLen:]
}

// TruncateForLog 专门用于日志输出的截断函数，默认最大长度500
func TruncateForLog(s string) string {
	return TruncateString(s, MaxLogLength)
}

// SafeLogValue 安全地记录日志值，对于大内容只记录长度或截断后的内容
func SafeLogValue(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	
	switch val := v.(type) {
	case string:
		if len(val) > MaxLogLength {
			return fmt.Sprintf("<string len:%d, truncated:%s>", len(val), TruncateForLog(val))
		}
		return val
	case []byte:
		if len(val) > MaxLogLength {
			return fmt.Sprintf("<[]byte len:%d, truncated:%s>", len(val), TruncateForLog(string(val)))
		}
		return string(val)
	default:
		// 对于其他类型，转换为字符串后截断
		str := fmt.Sprintf("%+v", val)
		if len(str) > MaxLogLength {
			return fmt.Sprintf("<value len:%d, truncated:%s>", len(str), TruncateForLog(str))
		}
		return str
	}
}

// MaskSensitiveField 屏蔽敏感字段，只显示字段名和长度
func MaskSensitiveField(fieldName string, value string) string {
	if value == "" {
		return fmt.Sprintf("%s:<empty>", fieldName)
	}
	return fmt.Sprintf("%s:<len:%d>", fieldName, len(value))
}

// LogAgentInfo 安全地记录 Agent 信息，不打印 SystemPromptTemplate 的完整内容
func LogAgentInfo(agents interface{}) string {
	// 这里返回一个简化的字符串表示，实际使用时需要根据具体的结构体类型来处理
	// 由于是 interface{}，我们只能返回一个提示
	return fmt.Sprintf("<AgentInfo: use SafeLogValue for details>")
}

