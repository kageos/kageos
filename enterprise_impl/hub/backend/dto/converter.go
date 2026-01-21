package dto

import (
	"strings"
	"time"
)

// splitTags 将逗号分隔的标签字符串转换为数组
func splitTags(tags string) []string {
	if tags == "" {
		return []string{}
	}
	return strings.Split(tags, ",")
}

// formatTime 格式化时间
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.DateTime)
}
